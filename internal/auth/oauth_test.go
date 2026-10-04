package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"
)

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	if len(nonce1) != 32 {
		t.Errorf("nonce length = %d, want 32", len(nonce1))
	}
	nonce2, _ := GenerateNonce()
	if nonce1 == nonce2 {
		t.Error("two nonces should be different")
	}
}

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(verifier) != 43 || !regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`).MatchString(verifier) {
		t.Fatalf("invalid PKCE verifier: %q", verifier)
	}
	hash := sha256.Sum256([]byte(verifier))
	if want := base64.RawURLEncoding.EncodeToString(hash[:]); challenge != want || len(challenge) != 43 {
		t.Errorf("challenge = %q, want %q", challenge, want)
	}
	other, _, _ := GeneratePKCE()
	if verifier == other {
		t.Error("PKCE verifier reused")
	}
}

func TestCallbackServerAcceptsCodeOnce(t *testing.T) {
	port, resultCh, cleanup, err := StartCallbackServer("correct-nonce")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	callback := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	for _, path := range []string{
		"/?code=valid&nonce=correct-nonce",
		"/callback?token=dd_secret&nonce=correct-nonce",
		"/callback?code=attacker&nonce=wrong",
		"/callback?code=attacker&nonce=correct-nonce&token=dd_secret",
		"/callback?code=attacker&nonce=correct-nonce&code=second",
	} {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Errorf("attacker callback %q was accepted", path)
		}
	}
	select {
	case result := <-resultCh:
		t.Fatalf("invalid callback completed login: %+v", result)
	default:
	}

	resp, err := http.Get(callback + "?code=one-time-code&nonce=correct-nonce")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(page) == "" {
		t.Fatalf("valid callback status %d", resp.StatusCode)
	}
	if result := <-resultCh; result.Code != "one-time-code" || result.Error != nil {
		t.Fatalf("result = %+v", result)
	}
	resp, err = http.Get(callback + "?code=replay&nonce=correct-nonce")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("replay status = %d, want 409", resp.StatusCode)
	}
	select {
	case result := <-resultCh:
		t.Fatalf("replay delivered second code: %+v", result)
	case <-time.After(20 * time.Millisecond):
	}
}
