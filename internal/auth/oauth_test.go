package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce() failed: %v", err)
	}
	if len(nonce1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("nonce length = %d, want 32", len(nonce1))
	}

	// Should be unique
	nonce2, _ := GenerateNonce()
	if nonce1 == nonce2 {
		t.Error("two nonces should be different")
	}
}

func TestCallbackServer(t *testing.T) {
	nonce := "test-nonce-12345"

	port, resultCh, cleanup, err := StartCallbackServer(nonce)
	if err != nil {
		t.Fatalf("StartCallbackServer() failed: %v", err)
	}
	defer cleanup()

	if port < 18787 || port > 18792 {
		t.Errorf("port = %d, want 18787-18792", port)
	}

	// Send a callback
	url := fmt.Sprintf("http://127.0.0.1:%d/?token=test-jwt&nonce=%s", port, nonce)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	select {
	case result := <-resultCh:
		if result.Error != nil {
			t.Fatalf("callback error: %v", result.Error)
		}
		if result.Token != "test-jwt" {
			t.Errorf("token = %q, want %q", result.Token, "test-jwt")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for callback result")
	}
}

func TestCallbackServerBadNonce(t *testing.T) {
	nonce := "correct-nonce"

	port, resultCh, cleanup, err := StartCallbackServer(nonce)
	if err != nil {
		t.Fatalf("StartCallbackServer() failed: %v", err)
	}
	defer cleanup()

	url := fmt.Sprintf("http://127.0.0.1:%d/?token=test-jwt&nonce=wrong-nonce", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	select {
	case result := <-resultCh:
		if result.Error == nil {
			t.Fatal("should have error for bad nonce")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestCallbackServerNoToken(t *testing.T) {
	nonce := "test-nonce"

	port, resultCh, cleanup, err := StartCallbackServer(nonce)
	if err != nil {
		t.Fatalf("StartCallbackServer() failed: %v", err)
	}
	defer cleanup()

	url := fmt.Sprintf("http://127.0.0.1:%d/?nonce=%s", port, nonce)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()

	select {
	case result := <-resultCh:
		if result.Error == nil {
			t.Fatal("should have error for missing token")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
