package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
)

const (
	closeParentID = "pa000000-0000-0000-0000-000000000001"
	closeChildID  = "ch000000-0000-0000-0000-000000000002"
	closeDoneID   = "dn000000-0000-0000-0000-000000000003"
)

// runCloseReport runs close against a server where the parent is pending in
// the list but completed when fetched afterwards (server auto-close).
func runCloseReport(t *testing.T, args ...string) string {
	t.Helper()
	listed := []apiPkg.Bead{
		{ID: closeParentID, Subject: "Parent feature", Status: "in_progress"},
		{ID: closeChildID, Subject: "Last child", Status: "in_progress", ParentBeadID: closeParentID},
		{ID: closeDoneID, Subject: "Already done", Status: "completed"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/beads":
			json.NewEncoder(w).Encode(listed)
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, closeParentID):
			json.NewEncoder(w).Encode(apiPkg.Bead{ID: closeParentID, Subject: "Parent feature", Status: "completed"})
		default:
			w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	deps := &Deps{
		Cfg:    &config.Config{ProjectID: "test-project-id", APIURL: server.URL, Token: "t", ConfigDir: t.TempDir()},
		Client: apiPkg.New(server.URL, "t", Version),
	}
	root := NewRootCmd(deps)
	root.SetArgs(append([]string{"close"}, args...))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := root.Execute()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	return string(out)
}

func TestCloseReportsAutoClosedParent(t *testing.T) {
	out := runCloseReport(t, "ch000000", "--summary=done")
	if !strings.Contains(out, "Closed: "+closeChildID) {
		t.Errorf("should report the close:\n%s", out)
	}
	if !strings.Contains(out, "closed automatically") || !strings.Contains(out, "devdash close pa000000 --summary") {
		t.Errorf("should point out the auto-closed parent and how to summarize it:\n%s", out)
	}
}

func TestCloseReportsAlreadyClosed(t *testing.T) {
	out := runCloseReport(t, "dn000000", "--summary=again")
	if !strings.Contains(out, "already closed") || strings.Contains(out, "Closed: "+closeDoneID) {
		t.Errorf("should say the issue was already closed and details replaced:\n%s", out)
	}
}
