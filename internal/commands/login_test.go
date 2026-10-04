package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devdashproject/devdash-cli/internal/config"
)

// runLogin runs `login` against a server that accepts only goodToken.
func runLogin(t *testing.T, goodToken string, stdin string, args ...string) (string, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+goodToken {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	t.Setenv("DD_TOKEN_FILE", filepath.Join(dir, "token"))
	t.Setenv(config.TokenEnvVar, "")
	deps := &Deps{Cfg: &config.Config{APIURL: server.URL, ConfigDir: dir}}

	rootCmd := NewRootCmd(deps)
	rootCmd.SetArgs(append([]string{"login"}, args...))
	rootCmd.SetIn(strings.NewReader(stdin))
	var errBuf bytes.Buffer
	rootCmd.SetErr(&errBuf)
	rootCmd.SetOut(&errBuf)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := rootCmd.Execute()
	w.Close()
	os.Stdout = oldStdout
	var out bytes.Buffer
	out.ReadFrom(r)
	return out.String() + errBuf.String(), err
}

func TestExchangeLoginCode(t *testing.T) {
	const code = "one-time-code"
	const verifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	const nonce = "client-nonce"
	const redirectURI = "http://127.0.0.1:18787/callback"
	token := "dd_" + strings.Repeat("a", 64)
	used := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/cli/exchange" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected exchange request: %s %s", r.Method, r.URL)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("invalid exchange body: %v", err)
		}
		if len(payload) != 4 || payload["code"] != code || payload["code_verifier"] != verifier || payload["redirect_uri"] != redirectURI || payload["nonce"] != nonce {
			t.Errorf("unexpected exchange body: %#v", payload)
			http.Error(w, "bad proof", http.StatusUnauthorized)
			return
		}
		if used {
			http.Error(w, "expired or replayed", http.StatusUnauthorized)
			return
		}
		used = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
	}))
	defer server.Close()

	got, err := exchangeLoginCode(server.URL, code, verifier, redirectURI, nonce)
	if err != nil || got != token {
		t.Fatalf("exchange did not return the expected token: %v", err)
	}
	got, err = exchangeLoginCode(server.URL, code, verifier, redirectURI, nonce)
	if got != "" || err == nil || !strings.Contains(err.Error(), "HTTP 401") || strings.Contains(err.Error(), code) {
		t.Fatalf("replayed exchange was not rejected: %v", err)
	}
}

func TestCLIAuthURLContainsChallengeWithoutCredentials(t *testing.T) {
	authURL, err := cliAuthURL("https://devdash.example", 18787, "nonce", strings.Repeat("A", 43))
	if err != nil {
		t.Fatal(err)
	}
	query := authURL.Query()
	if authURL.Path != "/api/auth/cli-token" || len(query) != 3 || query.Get("port") != "18787" || query.Get("nonce") != "nonce" || query.Get("code_challenge") != strings.Repeat("A", 43) {
		t.Fatalf("unexpected browser URL: %s", authURL)
	}
	if strings.Contains(authURL.String(), "token=") || strings.Contains(authURL.String(), "code_verifier") {
		t.Fatalf("browser URL contains credentials: %s", authURL)
	}
}

func TestExchangeLoginCodeRejectsRedirect(t *testing.T) {
	verifierExposed := false
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		verifierExposed = true
	}))
	defer attacker.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, attacker.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	_, err := exchangeLoginCode(server.URL, "code", "secret-verifier", "http://127.0.0.1:18787/callback", "nonce")
	if err == nil || verifierExposed {
		t.Fatalf("cross-origin redirect accepted or proof exposed: %v", err)
	}
}

func TestExchangeLoginCodeRejectsEmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token":""}`))
	}))
	defer server.Close()
	got, err := exchangeLoginCode(server.URL, "code", "verifier", "http://127.0.0.1:18787/callback", "nonce")
	if got != "" || err == nil {
		t.Fatalf("empty exchange token accepted: %v", err)
	}
}

func TestExchangeLoginCodeRejectsMalformedToken(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
	}{
		{"wrong prefix", "api_" + strings.Repeat("a", 64)},
		{"short token", "dd_" + strings.Repeat("a", 63)},
		{"uppercase hex", "dd_" + strings.Repeat("A", 64)},
		{"nonhex", "dd_" + strings.Repeat("g", 64)},
		{"extra suffix", "dd_" + strings.Repeat("a", 64) + "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]string{"token": tc.token})
			}))
			defer server.Close()
			got, err := exchangeLoginCode(server.URL, "code", "verifier", "http://127.0.0.1:18787/callback", "nonce")
			if got != "" || err == nil || !strings.Contains(err.Error(), "invalid code exchange response") {
				t.Fatalf("malformed exchange token accepted: %v", err)
			}
		})
	}
}

func TestExchangeLoginCodeRejectsExpiredCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "expired", http.StatusUnauthorized)
	}))
	defer server.Close()
	got, err := exchangeLoginCode(server.URL, "expired-code", "verifier", "http://127.0.0.1:18787/callback", "nonce")
	if got != "" || err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("expired code accepted: %v", err)
	}
}

func savedToken(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(os.Getenv("DD_TOKEN_FILE"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func TestLoginWithTokenFlag(t *testing.T) {
	out, err := runLogin(t, "dd_good123456", "", "--token=dd_good123456")
	if err != nil {
		t.Fatalf("login --token failed: %v\n%s", err, out)
	}
	if got := savedToken(t); got != "dd_good123456" {
		t.Errorf("saved token = %q", got)
	}
	if !strings.Contains(out, "token verified") {
		t.Errorf("expected verification message, got: %s", out)
	}
}

func TestLoginWithTokenStdin(t *testing.T) {
	out, err := runLogin(t, "dd_stdin123456", "dd_stdin123456\n", "--with-token")
	if err != nil {
		t.Fatalf("login --with-token failed: %v\n%s", err, out)
	}
	if got := savedToken(t); got != "dd_stdin123456" {
		t.Errorf("saved token = %q", got)
	}
	if strings.Contains(out, "dd_stdin123456") {
		t.Errorf("token echoed in output: %s", out)
	}
}

func TestLoginWithTokenEmptyStdin(t *testing.T) {
	_, err := runLogin(t, "dd_x", "", "--with-token")
	if err == nil || !strings.Contains(err.Error(), "no token read from stdin") {
		t.Errorf("expected empty-stdin error, got: %v", err)
	}
}

func TestLoginWithRejectedToken(t *testing.T) {
	_, err := runLogin(t, "dd_good123456", "", "--token=dd_bad1234567")
	if err == nil || !strings.Contains(err.Error(), "token rejected") {
		t.Fatalf("expected rejection, got: %v", err)
	}
	if got := savedToken(t); got != "" {
		t.Errorf("rejected token was saved: %q", got)
	}
}

func TestLoginPositionalTokenIsMasked(t *testing.T) {
	out, err := runLogin(t, "dd_x", "", "dd_abcdef0123456789")
	if err == nil {
		t.Fatal("expected error for positional argument")
	}
	if strings.Contains(err.Error()+out, "dd_abcdef0123456789") {
		t.Errorf("token not masked: %v %s", err, out)
	}
	if !strings.Contains(err.Error(), "--token") {
		t.Errorf("error should suggest --token: %v", err)
	}
}

func TestLoginHelpMentionsToken(t *testing.T) {
	out, err := runLogin(t, "dd_x", "", "--help")
	if err != nil {
		t.Fatalf("login --help failed: %v", err)
	}
	for _, want := range []string{"--token", "--with-token", "DEVDASH_TOKEN"} {
		if !strings.Contains(out, want) {
			t.Errorf("login --help missing %q", want)
		}
	}
}
