package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
)

// runRecordingClose runs `close` and returns the JSON body sent for the close.
func runRecordingClose(t *testing.T, args ...string) map[string]interface{} {
	t.Helper()
	var mu sync.Mutex
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(apiPkg.SampleBeads())
			return
		}
		data, _ := io.ReadAll(r.Body)
		mu.Lock()
		json.Unmarshal(data, &body)
		mu.Unlock()
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	deps := &Deps{
		Cfg:    &config.Config{ProjectID: "test-project-id", APIURL: server.URL, Token: "t", ConfigDir: t.TempDir()},
		Client: apiPkg.New(server.URL, "t", Version),
	}
	root := NewRootCmd(deps)
	root.SetArgs(append([]string{"close"}, args...))
	root.SetOut(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	return body
}

func TestCloseSendsServerFieldNames(t *testing.T) {
	body := runRecordingClose(t, "aaaa0000", "--summary=done", "--commit=abc123", "--pr=https://example.com/pr/1")
	cr, ok := body["completionResult"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing completionResult: %v", body)
	}
	if cr["commitSha"] != "abc123" || cr["prUrl"] != "https://example.com/pr/1" || cr["summary"] != "done" {
		t.Errorf("wrong completionResult keys: %v", cr)
	}
	if _, legacy := cr["commit"]; legacy {
		t.Errorf("should not send legacy 'commit' key: %v", cr)
	}
}

func TestCompletionResultDecodesLegacyKeys(t *testing.T) {
	var cr apiPkg.CompletionResult
	if err := json.Unmarshal([]byte(`{"summary":"s","pr":"u","commit":"c"}`), &cr); err != nil {
		t.Fatal(err)
	}
	if cr.PRURL != "u" || cr.CommitSHA != "c" {
		t.Errorf("legacy keys not decoded: %+v", cr)
	}
	out, _ := json.Marshal(cr)
	if !strings.Contains(string(out), `"prUrl":"u"`) || !strings.Contains(string(out), `"commitSha":"c"`) {
		t.Errorf("should re-encode with server keys: %s", out)
	}
}
