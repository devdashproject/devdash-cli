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

func TestActivityForBeadFiltersClientSide(t *testing.T) {
	target := "aaaa0000-0000-0000-0000-000000000001"
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/activity") {
			gotQuery = r.URL.RawQuery
			// Server ignores targetId and returns the whole feed
			w.Write([]byte(`{"data":[
				{"id":"a1","action":"bead.created","targetId":"` + target + `"},
				{"id":"a2","action":"bead.created","targetId":"bbbb0000-0000-0000-0000-000000000002"},
				{"id":"a3","action":"bead.status_changed","targetId":"` + target + `"}
			],"nextCursor":"x","hasMore":true}`))
			return
		}
		json.NewEncoder(w).Encode(apiPkg.SampleBeads())
	}))
	t.Cleanup(server.Close)

	deps := &Deps{
		Cfg:    &config.Config{ProjectID: "test-project-id", APIURL: server.URL, Token: "t", ConfigDir: t.TempDir()},
		Client: apiPkg.New(server.URL, "t", Version),
	}
	run := func(args ...string) string {
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
		return string(out)
	}

	out := run("aaaa0000")
	if !strings.Contains(gotQuery, "targetId="+target) || !strings.Contains(gotQuery, "limit=500") {
		t.Errorf("should request targetId and max window, got query %q", gotQuery)
	}
	if strings.Contains(out, `"a2"`) {
		t.Errorf("should drop other beads' events: %s", out)
	}
	if !strings.Contains(out, `"a1"`) || !strings.Contains(out, `"a3"`) {
		t.Errorf("should keep this bead's events: %s", out)
	}

	out = run("aaaa0000", "--limit=1")
	if strings.Contains(out, `"a3"`) || !strings.Contains(out, `"a1"`) {
		t.Errorf("--limit should cap filtered results: %s", out)
	}
}
