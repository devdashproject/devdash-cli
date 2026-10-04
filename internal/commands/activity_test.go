package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
)

// runActivity runs `activity` against a server that records the query and
// returns one filtered page with a next cursor.
func runActivity(t *testing.T, args ...string) (url.Values, string) {
	t.Helper()
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/activity") {
			got = r.URL.Query()
			w.Write([]byte(`{"data":[{"id":"a1","action":"bead.created","actorType":"user","targetId":"aaaa0000-0000-0000-0000-000000000001","createdAt":"2026-10-04T07:00:00Z"}],"nextCursor":"cur-2","hasMore":true}`))
			return
		}
		json.NewEncoder(w).Encode(apiPkg.SampleBeads())
	}))
	t.Cleanup(server.Close)

	deps := &Deps{
		Cfg:    &config.Config{ProjectID: "test-project-id", APIURL: server.URL, Token: "t", ConfigDir: t.TempDir()},
		Client: apiPkg.New(server.URL, "t", Version),
	}
	root := NewRootCmd(deps)
	root.SetArgs(append([]string{"activity"}, args...))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := root.Execute()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if err != nil {
		t.Fatalf("activity failed: %v", err)
	}
	return got, string(out)
}

func TestActivityForBeadUsesServerFilter(t *testing.T) {
	q, out := runActivity(t, "aaaa0000", "--limit=2", "--cursor=cur-1")
	if q.Get("targetId") != "aaaa0000-0000-0000-0000-000000000001" {
		t.Errorf("should send the resolved targetId, got %v", q)
	}
	if q.Get("limit") != "2" || q.Get("cursor") != "cur-1" {
		t.Errorf("should pass --limit and --cursor through to the server, got %v", q)
	}
	if !strings.Contains(out, `"nextCursor": "cur-2"`) {
		t.Errorf("JSON output should keep the server's pagination fields:\n%s", out)
	}
}

func TestActivityWithoutFlagsSendsNoQuery(t *testing.T) {
	q, _ := runActivity(t)
	if len(q) != 0 {
		t.Errorf("plain 'activity' should use server defaults, got %v", q)
	}
}

func TestActivityPrettyShowsNextCursor(t *testing.T) {
	_, out := runActivity(t, "aaaa0000", "--pretty")
	if !strings.Contains(out, "bead.created by user") || !strings.Contains(out, "More: add --cursor=cur-2") {
		t.Errorf("pretty view should list events and how to page:\n%s", out)
	}
}
