package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
)

func newSessionEnv(t *testing.T, onReq func(r *http.Request, body string)) func(args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		onReq(r, string(b))
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"sess-1","summary":"x"}`))
			return
		}
		_, _ = w.Write([]byte(`[{"id":"sess-1","summary":"x"}]`))
	}))
	t.Cleanup(srv.Close)
	deps := &Deps{
		Cfg:    &config.Config{ProjectID: "test-project-id", APIURL: srv.URL, Token: "t", FrontendURL: "https://example.com", CloseGate: "push", ConfigDir: t.TempDir()},
		Client: apiPkg.New(srv.URL, "t", Version),
	}
	return func(args ...string) (string, error) {
		cmd := NewRootCmd(deps)
		cmd.SetArgs(args)
		var out strings.Builder
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		err := cmd.Execute()
		return out.String(), err
	}
}

func TestSessionSavePostsSummary(t *testing.T) {
	var gotPath, gotBody, gotMethod string
	run := newSessionEnv(t, func(r *http.Request, body string) { gotPath, gotBody, gotMethod = r.URL.Path, body, r.Method })
	if _, err := run("session", "save", "--summary=did the thing"); err != nil {
		t.Fatalf("session save failed: %v", err)
	}
	if gotMethod != http.MethodPost || !strings.HasSuffix(gotPath, "/memory/sessions/test-project-id") {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil || body["summary"] != "did the thing" {
		t.Fatalf("unexpected body: %s (%v)", gotBody, err)
	}
}

func TestSessionSaveRequiresSummary(t *testing.T) {
	run := newSessionEnv(t, func(*http.Request, string) {})
	_, err := run("session", "save")
	if err == nil || !strings.Contains(err.Error(), "--summary is required") {
		t.Fatalf("expected summary-required error, got: %v", err)
	}
}

func TestSessionListPassesLimit(t *testing.T) {
	var gotQuery string
	run := newSessionEnv(t, func(r *http.Request, _ string) { gotQuery = r.URL.RawQuery })
	if _, err := run("session", "list", "--limit=3"); err != nil {
		t.Fatalf("session list failed: %v", err)
	}
	if gotQuery != "limit=3" {
		t.Fatalf("expected limit=3, got %q", gotQuery)
	}
}
