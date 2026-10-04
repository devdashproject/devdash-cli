package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	callbackHTML = `<!DOCTYPE html><html><body><h2>Sign-in received</h2><p>Return to the CLI to finish authentication. You can close this window.</p><script>window.close()</script></body></html>`
)

// OAuthResult holds the result of an OAuth flow.
type OAuthResult struct {
	Code  string
	Error error
}

// StartCallbackServer starts a local HTTP server to receive the OAuth callback.
// Returns the port, a channel that will receive one code, and a cleanup function.
func StartCallbackServer(nonce string) (int, <-chan OAuthResult, func(), error) {
	// Try ports 18787-18792
	var listener net.Listener
	var port int
	for p := 18787; p <= 18792; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			listener = l
			port = p
			break
		}
	}
	if listener == nil {
		return 0, nil, nil, fmt.Errorf("could not find available port (tried 18787-18792)")
	}

	resultCh := make(chan OAuthResult, 1)
	var once sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Invalid callback", http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query()
		if len(query) != 2 || len(query["code"]) != 1 || len(query["nonce"]) != 1 || query.Get("code") == "" || query.Get("nonce") != nonce {
			http.Error(w, "Invalid callback", http.StatusBadRequest)
			return
		}

		accepted := false
		once.Do(func() {
			accepted = true
			resultCh <- OAuthResult{Code: query.Get("code")}
		})
		if !accepted {
			http.Error(w, "Callback already received", http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, callbackHTML)
	})

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			resultCh <- OAuthResult{Error: err}
		}
	}()

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}

	return port, resultCh, cleanup, nil
}

// GenerateNonce creates a random hex string for OAuth state.
func GenerateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GeneratePKCE creates a 43-character RFC 7636 verifier and its S256 challenge.
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return verifier, challenge, nil
}
