package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
)

const (
	uuidA  = "aaaa1111-0000-0000-0000-000000000001"
	uuidB  = "bbbb2222-0000-0000-0000-000000000002"
	projP1 = "11111111-1111-1111-1111-111111111111"
	projP2 = "22222222-2222-2222-2222-222222222222"
)

type recordedReq struct {
	Method string
	Path   string
	Query  string
	Body   map[string]interface{}
}

// scopeServer records every request. oldServer makes writes without a
// projectId fail with 400 like a pre-#757 server; both beads live in projP2.
type scopeServer struct {
	mu        sync.Mutex
	reqs      []recordedReq
	oldServer bool
}

func (s *scopeServer) writes() []recordedReq {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []recordedReq
	for _, r := range s.reqs {
		if r.Method != "GET" {
			out = append(out, r)
		}
	}
	return out
}

func (s *scopeServer) has(method, path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.reqs {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}

func newScopeEnv(t *testing.T, projectID string, old bool) (*scopeServer, func(args ...string) (string, error)) {
	t.Helper()
	ss := &scopeServer{oldServer: old}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := recordedReq{Method: r.Method, Path: strings.TrimPrefix(r.URL.Path, "/api"), Query: r.URL.RawQuery}
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &rec.Body)
		}
		ss.mu.Lock()
		ss.reqs = append(ss.reqs, rec)
		ss.mu.Unlock()

		hasPID := strings.Contains(rec.Query, "projectId=")
		if rec.Body != nil {
			_, hasPID = rec.Body["projectId"]
		}
		if r.Method != "GET" && !hasPID && ss.oldServer {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"Validation failed: projectId is required"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && rec.Path == "/beads":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == "GET" && strings.HasSuffix(rec.Path, "/comments"):
			_, _ = w.Write([]byte(`[]`))
		case r.Method == "GET":
			id := strings.TrimPrefix(rec.Path, "/beads/")
			_, _ = w.Write([]byte(`{"id":"` + id + `","subject":"S","status":"pending","priority":1,"beadType":"task","projectId":"` + projP2 + `"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"x","status":"completed"}`))
		}
	}))
	t.Cleanup(srv.Close)

	return ss, envAgainst(t, srv.URL, projectID)
}

// envAgainst runs commands against the API at url with the given project.
func envAgainst(t *testing.T, url, projectID string) func(args ...string) (string, error) {
	t.Helper()
	deps := &Deps{
		Cfg:    &config.Config{ProjectID: projectID, APIURL: url, Token: "t", ConfigDir: t.TempDir()},
		Client: apiPkg.New(url, "t", Version),
	}
	return func(args ...string) (string, error) {
		root := NewRootCmd(deps)
		root.SetArgs(args)
		oldOut, oldErr := os.Stdout, os.Stderr
		r, w, _ := os.Pipe()
		os.Stdout, os.Stderr = w, w
		var buf bytes.Buffer
		done := make(chan struct{})
		go func() { _, _ = buf.ReadFrom(r); close(done) }()
		err := root.Execute()
		w.Close()
		os.Stdout, os.Stderr = oldOut, oldErr
		<-done
		return buf.String(), err
	}
}

// Full UUIDs must work with no project configured, sending no projectId.
func TestFullUUIDWorksWithNoProject(t *testing.T) {
	cases := map[string][]string{
		"show":        {"show", uuidA},
		"show-pretty": {"show", uuidA, "--pretty"},
		"comment":     {"comment", uuidA, "hello"},
		"comments":    {"comments", uuidA},
		"update":      {"update", uuidA, "--priority=1"},
		"archive":     {"update", uuidA, "--status=archived"},
		"close":       {"close", uuidA, "--summary=done"},
		"close-many":  {"close", uuidA, uuidB},
		"delete":      {"delete", uuidA, "--force"},
		"report":      {"report", uuidA, "--status=committed"},
		"dep-add":     {"dep", "add", uuidA, uuidB},
		"dep-remove":  {"dep", "remove", uuidA, uuidB},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			ss, run := newScopeEnv(t, "", false)
			if out, err := run(args...); err != nil {
				t.Fatalf("%v with no project failed: %v\n%s", args, err, out)
			}
			ss.mu.Lock()
			defer ss.mu.Unlock()
			for _, r := range ss.reqs {
				if strings.Contains(r.Query, "projectId") {
					t.Errorf("%s %s sent projectId in query: %s", r.Method, r.Path, r.Query)
				}
				if _, ok := r.Body["projectId"]; ok {
					t.Errorf("%s %s sent projectId in body", r.Method, r.Path)
				}
				if r.Path == "/beads" {
					t.Errorf("full UUID should not list the project's beads")
				}
			}
		})
	}
}

// Prefixes still need (and use) the current project.
func TestPrefixStillRequiresProject(t *testing.T) {
	for _, args := range [][]string{
		{"show", "aaaa1111"}, {"comment", "aaaa1111", "x"}, {"update", "aaaa1111", "--priority=1"},
		{"close", "aaaa1111"}, {"close", uuidA, "bbbb2222"}, {"dep", "add", uuidA, "bbbb2222"},
	} {
		_, run := newScopeEnv(t, "", false)
		_, err := run(args...)
		if err == nil || !strings.Contains(err.Error(), "no project configured") {
			t.Errorf("%v: want 'no project configured', got %v", args, err)
		}
	}
}

// A UUID from another project is sent without projectId even when a project
// is configured, so the server checks the bead's own project.
func TestUUIDIgnoresConfiguredProject(t *testing.T) {
	ss, run := newScopeEnv(t, projP1, false)
	if out, err := run("update", uuidA, "--priority=2"); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	w := ss.writes()
	if len(w) != 1 || w[0].Body["projectId"] != nil {
		t.Errorf("expected one PATCH without projectId, got %+v", w)
	}
}

// Against an older server that requires projectId, the CLI resolves the
// bead's project with one GET /beads/:id and retries with it.
func TestOldServerFallbackResolvesProject(t *testing.T) {
	cases := map[string][]string{
		"update":  {"update", uuidA, "--priority=1"},
		"comment": {"comment", uuidA, "hi"},
		"close":   {"close", uuidA},
		"close2":  {"close", uuidA, uuidB},
		"delete":  {"delete", uuidA, "--force"},
		"report":  {"report", uuidA, "--status=pushed"},
		"dep-add": {"dep", "add", uuidA, uuidB},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			ss, run := newScopeEnv(t, "", true)
			if out, err := run(args...); err != nil {
				t.Fatalf("%v against old server: %v\n%s", args, err, out)
			}
			if !ss.has("GET", "/beads/"+uuidA) {
				t.Errorf("expected GET /beads/<uuid> to resolve the project")
			}
			last := ss.writes()
			ok := false
			for _, wr := range last {
				p, _ := wr.Body["projectId"].(string)
				if p == projP2 || strings.Contains(wr.Query, "projectId="+projP2) {
					ok = true
				}
			}
			if !ok {
				t.Errorf("expected a retry carrying projectId %s, writes: %+v", projP2, last)
			}
		})
	}
}
