package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var callbackPage = template.Must(template.New("login-success").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Sign-in received · DevDash</title>
<style>
body{margin:0;min-height:100vh;display:grid;place-items:center;background:#f5f7fb;color:#172033;font:17px/1.6 system-ui,sans-serif}
main{box-sizing:border-box;max-width:520px;margin:24px;padding:40px;background:white;border:1px solid #e2e8f0;border-radius:20px}
.brand{font-weight:700;color:#2563eb}h1{font-size:30px;line-height:1.2;margin:20px 0 16px}p{color:#526077}
a{display:inline-block;margin-top:12px;padding:10px 18px;background:#2563eb;color:white;border-radius:8px;text-decoration:none;font-weight:600}
</style></head><body><main><div class="brand">DevDash</div><h1>Sign-in received.</h1>
<p>You can close this browser window and return to your terminal to confirm authentication finished.</p>
{{if .FrontendURL}}<a href="{{.FrontendURL}}" rel="noreferrer">Open DevDash</a>{{end}}
</main></body></html>`))

func successPageURL(frontendURL string) string {
	target, err := url.Parse(frontendURL)
	if err != nil || target.Hostname() == "" || target.User != nil ||
		(target.Scheme != "https" && target.Scheme != "http") {
		return ""
	}
	return target.String()
}

// OAuthResult holds the result of an OAuth flow.
type OAuthResult struct {
	Code  string
	Error error
}

// StartCallbackServer starts a local HTTP server to receive the OAuth callback.
// Returns the port, a channel that will receive one code, and a cleanup function.
func StartCallbackServer(nonce string, frontendURLs ...string) (int, <-chan OAuthResult, func(), error) {
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

	frontendURL := ""
	if len(frontendURLs) > 0 {
		frontendURL = successPageURL(frontendURLs[0])
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		_ = callbackPage.Execute(w, struct{ FrontendURL string }{frontendURL})
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
