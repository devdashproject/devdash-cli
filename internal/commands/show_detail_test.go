package commands

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
)

const uuidC = "cccc3333-0000-0000-0000-000000000003"

const detailJSON = `{"id":"aaaa1111-0000-0000-0000-000000000001","subject":"Main","status":"pending","priority":1,"beadType":"task",
 "description":"Do the thing","project":{"id":"p1","name":"CLI Repo"},
 "parent":{"id":"pppp0000-0000-0000-0000-000000000009","subject":"Epic","status":"in_progress"},
 "children":[{"id":"cccc0000-0000-0000-0000-00000000000c","subject":"Kid","status":"pending"}],
 "blockedBy":["bbbb0000-0000-0000-0000-00000000000b","dddd0000-0000-0000-0000-00000000000d","eeee0000-0000-0000-0000-00000000000e"],
 "blockedByDetails":[
  {"id":"bbbb0000-0000-0000-0000-00000000000b","subject":"Server fix","status":"pending","projectId":"p2","projectName":"Server Repo"},
  {"id":"dddd0000-0000-0000-0000-00000000000d","subject":"Same project","status":"pending","projectId":"p1","projectName":"CLI Repo"},
  {"id":"eeee0000-0000-0000-0000-00000000000e","subject":"Issue in an inaccessible project","status":"pending","projectId":null,"projectName":null,"inaccessible":true}],
 "blocksDetails":[]}`

func TestPrettyBeadRichDetail(t *testing.T) {
	out, err := prettyBead([]byte(detailJSON))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Project:     CLI Repo (p1)",
		"Parent:      pppp0000 [in_progress] Epic",
		"Children:\n  cccc0000 [pending] Kid",
		"bbbb0000 [pending] Server fix (project: Server Repo)",
		"dddd0000 [pending] Same project\n",
		"eeee0000 (no access)",
		"Blocks:      none",
		"Do the thing",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "inaccessible project") {
		t.Errorf("redacted subject must not be shown:\n%s", out)
	}
	if strings.Contains(out, "Same project (project:") {
		t.Errorf("same-project blocker should not repeat the project name:\n%s", out)
	}
}

// An older server sends none of the new fields: output must still be sane.
func TestPrettyBeadOldServerShape(t *testing.T) {
	out, err := prettyBead([]byte(`{"id":"aaaa1111-0","subject":"Old","status":"pending","priority":2,"beadType":"task","blockedBy":["bbbb0000-1"],"description":""}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Parent:      none", "Blocked by:  bbbb0000", "Blocks:      none", "Description: none"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Project:") {
		t.Errorf("no project info available, should not print Project:\n%s", out)
	}
}

func TestAPIErrorShowsNotFoundHintAndCandidates(t *testing.T) {
	e := &apiPkg.APIError{
		StatusCode: 404, Message: "Bead not found", Hint: "Treated as an ID prefix. Use the full UUID.",
		SearchedProjectID: "p1", ActualProjectID: "p2", ActualProjectName: "Server Repo",
		Candidates: []apiPkg.Candidate{{ID8: "abcd1234", Subject: "Alpha one", ProjectID: "p2"}},
	}
	msg := e.Error()
	for _, want := range []string{"API error (404): Bead not found", "searched project: p1", "found in project: p2 (Server Repo)",
		"hint: Treated as an ID prefix", "abcd1234  Alpha one"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
	// Old-server body: nothing extra.
	if got := (&apiPkg.APIError{StatusCode: 404, Message: "Bead not found"}).Error(); got != "API error (404): Bead not found" {
		t.Errorf("legacy error changed: %q", got)
	}
}

// crossServer is a tiny API: an issue in project p1 (CLI Repo) blocked by one
// in p2 (Server Repo) and by one the caller cannot read (404).
func crossEnv(t *testing.T, serverBlockerStatus string) func(args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case path == "/beads" && r.Method == "GET":
			_, _ = w.Write([]byte(`[{"id":"` + uuidA + `","subject":"Main work","status":"pending","priority":1,"beadType":"task","blockedBy":["` + uuidB + `","` + uuidC + `"]}]`))
		case path == "/beads/"+uuidB && r.Method == "GET":
			_, _ = w.Write([]byte(`{"id":"` + uuidB + `","subject":"Server fix","status":"` + serverBlockerStatus + `","priority":1,"beadType":"task","project":{"id":"` + projP2 + `","name":"Server Repo"}}`))
		case path == "/beads/"+uuidA && r.Method == "GET":
			_, _ = w.Write([]byte(`{"id":"` + uuidA + `","subject":"Main work","status":"pending","priority":1,"beadType":"task","project":{"id":"` + projP1 + `","name":"CLI Repo"}}`))
		case path == "/beads/"+uuidA+"/dependencies" && r.Method == "POST":
			_, _ = w.Write([]byte(`{}`))
		case path == "/beads/aaaa1111" || path == "/beads/"+uuidC:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"Bead not found","inputKind":"prefix","searchedProjectId":"` + projP1 + `","hint":"Treated as an ID prefix. Use a longer prefix or a full UUID.","ambiguous":true,"candidates":[{"id8":"aaaa1111","subject":"Alpha one","projectId":"` + projP2 + `"}]}`))
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"Bead not found"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return envAgainst(t, srv.URL, projP1)
}

func TestShowNotFoundPrintsHintAndCandidates(t *testing.T) {
	run := crossEnv(t, "pending")
	_, err := run("show", uuidC)
	if err == nil {
		t.Fatal("expected not-found error")
	}
	for _, want := range []string{"Bead not found", "hint: Treated as an ID prefix", "aaaa1111  Alpha one"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in %q", want, err.Error())
		}
	}
}

func TestBlockedShowsForeignProjectAndNoAccess(t *testing.T) {
	run := crossEnv(t, "pending")
	out, err := run("blocked")
	if err != nil {
		t.Fatalf("blocked: %v\n%s", err, out)
	}
	if !strings.Contains(out, "bbbb2222 (Server Repo)") || !strings.Contains(out, "cccc3333 (no access)") {
		t.Errorf("blocked should label foreign blockers:\n%s", out)
	}
}

func TestReadyTreatsCompletedForeignBlockerAsDone(t *testing.T) {
	// uuidC stays inaccessible, so the issue is still blocked...
	out, _ := crossEnv(t, "completed")("ready")
	if strings.Contains(out, "Main work") {
		t.Errorf("issue with an inaccessible blocker must not be ready:\n%s", out)
	}
	// ...and a pending foreign blocker blocks too.
	out, _ = crossEnv(t, "pending")("ready")
	if strings.Contains(out, "Main work") {
		t.Errorf("issue with a pending foreign blocker must not be ready:\n%s", out)
	}
}

func TestReadyShowsCompletedForeignBlockerProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api")
		switch path {
		case "/beads":
			_, _ = w.Write([]byte(`[{"id":"` + uuidA + `","subject":"Main work","status":"pending","priority":1,"beadType":"task","blockedBy":["` + uuidB + `"]}]`))
		case "/beads/" + uuidB:
			_, _ = w.Write([]byte(`{"id":"` + uuidB + `","subject":"Server fix","status":"completed","project":{"id":"` + projP2 + `","name":"Server Repo"}}`))
		}
	}))
	t.Cleanup(srv.Close)
	out, err := envAgainst(t, srv.URL, projP1)("ready")
	if err != nil || !strings.Contains(out, "Main work") || !strings.Contains(out, "bbbb2222 (Server Repo)") {
		t.Errorf("ready should list it and name the foreign blocker's project (%v):\n%s", err, out)
	}
}

func TestDepAddShowsForeignBlockerProject(t *testing.T) {
	out, err := crossEnv(t, "pending")("dep", "add", uuidA, uuidB)
	if err != nil {
		t.Fatalf("dep add: %v\n%s", err, out)
	}
	if !strings.Contains(out, "blocker is in project Server Repo") {
		t.Errorf("dep add should name the foreign project:\n%s", out)
	}
}
