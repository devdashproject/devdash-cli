package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	callbackHTML = `<!DOCTYPE html><html><body><h2>Authentication successful!</h2><p>You can close this window.</p><script>window.close()</script></body></html>`
)

// OAuthResult holds the result of an OAuth flow.
type OAuthResult struct {
	Token string
	Error error
}

// StartCallbackServer starts a local HTTP server to receive the OAuth callback.
// Returns the port, a channel that will receive the token, and a cleanup function.
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		receivedNonce := r.URL.Query().Get("nonce")

		if receivedNonce != nonce {
			http.Error(w, "Invalid nonce", http.StatusBadRequest)
			resultCh <- OAuthResult{Error: fmt.Errorf("nonce mismatch")}
			return
		}

		if token == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no token received"
			}
			http.Error(w, errMsg, http.StatusBadRequest)
			resultCh <- OAuthResult{Error: fmt.Errorf("%s", errMsg)}
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, callbackHTML)
		resultCh <- OAuthResult{Token: token}
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
		server.Shutdown(ctx)
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
