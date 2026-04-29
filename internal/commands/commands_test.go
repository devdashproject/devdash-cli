package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
)

// newTestEnv creates a fresh command tree backed by a mock API server.
// Each test gets isolated state — no global leakage.
func newTestEnv(t *testing.T, beads []apiPkg.Bead) func(args ...string) (string, error) {
	t.Helper()

	mux := apiMux(beads)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	deps := &Deps{
		Cfg: &config.Config{
			ProjectID:   "test-project-id",
			APIURL:      server.URL,
			Token:       "test-token",
			FrontendURL: "https://example.com",
			CloseGate:   "push",
			ConfigDir:   t.TempDir(),
		},
		Client: apiPkg.New(server.URL, "test-token", Version),
	}

	return func(args ...string) (string, error) {
		rootCmd := NewRootCmd(deps)
		rootCmd.SetArgs(args)

		// Capture stdout/stderr via pipes.
		// Drain concurrently to avoid deadlock when output exceeds
		// the OS pipe buffer (small on Windows, ~4KB).
		oldStdout, oldStderr := os.Stdout, os.Stderr
		rOut, wOut, _ := os.Pipe()
		rErr, wErr, _ := os.Pipe()
		os.Stdout = wOut
		os.Stderr = wErr

		var capturedOut, capturedErr bytes.Buffer
		done := make(chan struct{})
		go func() {
			capturedOut.ReadFrom(rOut)
			capturedErr.ReadFrom(rErr)
			close(done)
		}()

		err := rootCmd.Execute()

		wOut.Close()
		wErr.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr

		<-done

		return capturedOut.String() + capturedErr.String(), err
	}
}

func apiMux(beads []apiPkg.Bead) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/beads", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			json.NewEncoder(w).Encode(beads)
		case "POST":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			resp := apiPkg.Bead{
				ID:      "new-bead-0000-0000-0000-000000000001",
				Subject: req["subject"].(string),
				Status:  "pending",
			}
			json.NewEncoder(w).Encode(resp)
		}
	})

	mux.HandleFunc("/api/beads/bulk/close", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/beads/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/beads/")
		beadID := strings.SplitN(path, "/", 2)[0]

		for _, b := range beads {
			if b.ID == beadID || strings.HasPrefix(b.ID, beadID) {
				switch r.Method {
				case "GET":
					json.NewEncoder(w).Encode(b)
				case "PATCH":
					json.NewEncoder(w).Encode(b)
				case "DELETE":
					json.NewEncoder(w).Encode(map[string]string{"deleted": beadID})
				case "POST":
					json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				}
				return
			}
		}
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})

	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiPkg.Project{ID: "test-project-id", Name: "test-project", GithubRepo: "user/test"})
	})

	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiPkg.SampleProjects())
	})

	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]apiPkg.Job{
			{ID: "job-001", Status: "completed", BeadID: beads[0].ID, Prompt: "test", CreatedAt: "2026-03-27"},
		})
	})

	mux.HandleFunc("/api/jobs/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiPkg.Job{ID: "job-001", Status: "completed", OutputLog: "line1\nline2\nline3"})
	})

	mux.HandleFunc("/api/activity", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]string{{"action": "created"}})
	})

	mux.HandleFunc("/api/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			json.NewEncoder(w).Encode([]map[string]string{{"id": "tok-1", "name": "test"}})
		case "POST":
			json.NewEncoder(w).Encode(map[string]string{"id": "tok-new", "token": "dd_secret"})
		}
	})

	mux.HandleFunc("/api/auth/tokens/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"deleted": "ok"})
	})

	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiPkg.CurrentUser{
			ID:    "test-user-id",
			Email: "test@example.com",
			Name:  "Test User",
		})
	})

	return mux
}

// --- Core Command Tests ---

func TestVersionCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(out, "devdash") {
		t.Errorf("output should contain 'devdash', got: %s", out)
	}
}

func TestReadyCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("ready")
	if err != nil {
		t.Fatalf("ready failed: %v", err)
	}
	if !strings.Contains(out, "Ready task") {
		t.Errorf("should contain 'Ready task', got: %s", out)
	}
	if !strings.Contains(out, "Scored task") {
		t.Errorf("should contain 'Scored task', got: %s", out)
	}
	if strings.Contains(out, "Blocked task") {
		t.Errorf("should not contain blocked task")
	}
	if strings.Contains(out, "In progress") {
		t.Errorf("should not contain in-progress")
	}
	if strings.Contains(out, "Completed") {
		t.Errorf("should not contain completed")
	}
	if strings.Contains(out, "Thought item") {
		t.Errorf("should not contain thought")
	}
}

func TestShowCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("show", beads[0].ID)
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if !strings.Contains(out, "Ready task") {
		t.Errorf("should contain bead subject, got: %s", out)
	}
}

func TestCreateCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("create", "--title=New task", "--type=bug", "--priority=1")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !strings.Contains(out, "Created:") {
		t.Errorf("should contain 'Created:', got: %s", out)
	}
}

func TestCreateCommandMissingTitle(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("create")
	if err == nil && !strings.Contains(out, "--title is required") {
		t.Fatal("create without --title should fail")
	}
}

func TestCreateCommandDashTitle(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	_, err := run("create", "--title=-urgent")
	if err == nil {
		t.Fatal("create with dash-prefixed title should fail")
	}
	if !strings.Contains(err.Error(), "cannot start with '-'") {
		t.Errorf("should mention cannot start with '-', got: %v", err)
	}
	if !strings.Contains(err.Error(), "--title=") {
		t.Errorf("should suggest --title= syntax, got: %v", err)
	}
}

func TestAnalyzeCommandAsync(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("analyze", beads[0].ID)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if !strings.Contains(out, "Analysis queued:") {
		t.Errorf("should contain 'Analysis queued:', got: %s", out)
	}
	if !strings.Contains(out, "devdash jobs show") {
		t.Errorf("should contain check-status hint, got: %s", out)
	}
	if !strings.Contains(out, "job-001") {
		t.Errorf("should contain full job ID on stdout, got: %s", out)
	}
	if strings.Contains(out, "Complexity:") {
		t.Errorf("should not contain analysis results (async mode), got: %s", out)
	}
}

func TestUpdateCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("update", beads[0].ID, "--status=in_progress")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(out, "Updated:") {
		t.Errorf("should contain 'Updated:', got: %s", out)
	}
}

func TestUpdateCommandNoChanges(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("update", beads[0].ID)
	if err == nil && !strings.Contains(out, "no changes") {
		t.Fatal("update with no changes should fail")
	}
}

func TestCloseCommandSingle(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("close", beads[0].ID, "--summary=Done")
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if !strings.Contains(out, "Closed:") {
		t.Errorf("should contain 'Closed:', got: %s", out)
	}
}

func TestCloseCommandBulk(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("close", beads[0].ID, beads[2].ID, "--summary=Done")
	if err != nil {
		t.Fatalf("close bulk failed: %v", err)
	}
	count := strings.Count(out, "Closed:")
	if count != 2 {
		t.Errorf("expected 2 'Closed:' lines, got %d in: %s", count, out)
	}
}

func TestMoveCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("move", beads[0].ID, "--to=proj-0002")
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if !strings.Contains(out, "Moved:") {
		t.Errorf("want 'Moved:' in output, got: %s", out)
	}
}

func TestMoveCommandShortID(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("move", beads[0].ID[:8], "--to=proj-0002")
	if err != nil {
		t.Fatalf("move with short ID failed: %v", err)
	}
	if !strings.Contains(out, "Moved:") {
		t.Errorf("want 'Moved:' in output, got: %s", out)
	}
}

func TestMoveCommandMissingTo(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	_, err := run("move", beads[0].ID)
	if err == nil || !strings.Contains(err.Error(), "--to is required") {
		t.Errorf("expected '--to is required' error, got: %v", err)
	}
}

func TestMoveCommandSameProject(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	_, err := run("move", beads[0].ID, "--to=test")
	if err == nil || !strings.Contains(err.Error(), "same") {
		t.Errorf("expected same-project error, got: %v", err)
	}
}

func TestMoveCommandNotFound(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	_, err := run("move", "zzz-unknown-prefix", "--to=proj-0001")
	if err == nil {
		t.Fatal("move with unknown ID should fail")
	}
}

func TestMoveCommandShortToPrefix(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("move", beads[0].ID, "--to=proj-0001")
	if err != nil {
		t.Fatalf("move with short --to failed: %v", err)
	}
	if !strings.Contains(out, "Moved:") {
		t.Errorf("want 'Moved:' in output, got: %s", out)
	}
}

func TestMoveCommandFromFlag(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("move", beads[0].ID, "--from=test-project-id", "--to=proj-0001")
	if err != nil {
		t.Fatalf("move with --from failed: %v", err)
	}
	if !strings.Contains(out, "Moved:") {
		t.Errorf("want 'Moved:' in output, got: %s", out)
	}
}

func TestMoveCommandUnknownTo(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	_, err := run("move", apiPkg.SampleBeads()[0].ID, "--to=zzzzzzzz")
	if err == nil || !strings.Contains(err.Error(), "no project found") {
		t.Errorf("expected 'no project found' error, got: %v", err)
	}
}

func TestMoveCommandAmbiguousTo(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	_, err := run("move", apiPkg.SampleBeads()[0].ID, "--to=proj")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' error, got: %v", err)
	}
}

func TestMoveCommandWrongSourceProject(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	_, err := run("move", "zzzzzzzz", "--to=proj-0001")
	if err == nil || !strings.Contains(err.Error(), "different project") {
		t.Errorf("expected source project hint in error, got: %v", err)
	}
}

func TestListCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(out, "Ready task") {
		t.Errorf("should contain tasks, got: %s", out)
	}
}

func TestListCommandStatusFilter(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("list", "--status=completed")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(out, "Completed") {
		t.Errorf("should contain completed, got: %s", out)
	}
	if strings.Contains(out, "Ready task") {
		t.Errorf("should not contain pending")
	}
}

func TestListCommandMineFilter(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("list", "--mine")
	if err != nil {
		t.Fatalf("list --mine failed: %v", err)
	}
	if !strings.Contains(out, "Ready task") {
		t.Errorf("should contain Ready task assigned to test user, got: %s", out)
	}
	if !strings.Contains(out, "In progress") {
		t.Errorf("should contain In progress assigned to test user, got: %s", out)
	}
	if strings.Contains(out, "Blocked task") {
		t.Errorf("should not contain tasks assigned to other users")
	}
	if strings.Contains(out, "Thought item") {
		t.Errorf("should not contain tasks assigned to other users")
	}
}

func TestReportCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("report", beads[0].ID, "--status=code_complete", "--summary=Done coding")
	if err != nil {
		t.Fatalf("report failed: %v", err)
	}
	if !strings.Contains(out, "Report submitted") {
		t.Errorf("should contain 'Report submitted', got: %s", out)
	}
}

func TestReportCommandMissingStatus(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("report", beads[0].ID)
	if err == nil && !strings.Contains(out, "--status is required") {
		t.Fatal("report without --status should fail")
	}
}

// --- Secondary Command Tests ---

func TestBlockedCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("blocked")
	if err != nil {
		t.Fatalf("blocked failed: %v", err)
	}
	if !strings.Contains(out, "Blocked task") {
		t.Errorf("should contain blocked task, got: %s", out)
	}
	if !strings.Contains(out, "blocked by:") {
		t.Errorf("should contain 'blocked by:', got: %s", out)
	}
}

func TestStaleCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("stale")
	if err != nil {
		t.Fatalf("stale failed: %v", err)
	}
	if !strings.Contains(out, "Stale task") {
		t.Errorf("should contain stale task, got: %s", out)
	}
	if !strings.Contains(out, "45m") {
		t.Errorf("should contain stale minutes, got: %s", out)
	}
}

func TestStatsCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("stats")
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if !strings.Contains(out, "Total:") {
		t.Errorf("should contain 'Total:', got: %s", out)
	}
	if !strings.Contains(out, "Blocked:") {
		t.Errorf("should contain 'Blocked:', got: %s", out)
	}
}

func TestDepAddCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("dep", "add", beads[0].ID, beads[2].ID)
	if err != nil {
		t.Fatalf("dep add failed: %v", err)
	}
	if !strings.Contains(out, "Added dependency") {
		t.Errorf("should contain 'Added dependency', got: %s", out)
	}
}

func TestDepRemoveCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("dep", "remove", beads[1].ID, beads[2].ID)
	if err != nil {
		t.Fatalf("dep remove failed: %v", err)
	}
	if !strings.Contains(out, "Removed dependency") {
		t.Errorf("should contain 'Removed dependency', got: %s", out)
	}
}

func TestCommentCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	_, err := run("comment", beads[0].ID, "--body=Test comment")
	if err != nil {
		t.Fatalf("comment failed: %v", err)
	}
}

func TestCommentCommandMissingBody(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("comment", beads[0].ID)
	if err == nil && !strings.Contains(out, "--body is required") {
		t.Fatal("comment without --body should fail")
	}
}

func TestCommentsCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	_, err := run("comments", beads[0].ID)
	if err != nil {
		t.Fatalf("comments failed: %v", err)
	}
}

func TestActivityCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	_, err := run("activity")
	if err != nil {
		t.Fatalf("activity failed: %v", err)
	}
}

func TestFindCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("find", beads[0].ID)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if !strings.Contains(out, "Ready task") {
		t.Errorf("should contain bead subject, got: %s", out)
	}
}

func TestFindCommandShortID(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("find", "aaaa")
	if err == nil && !strings.Contains(out, "full UUID") {
		t.Fatal("find with short ID should fail")
	}
}

func TestDeleteCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("delete", beads[0].ID, "--force")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if !strings.Contains(out, "Deleted:") {
		t.Errorf("should contain 'Deleted:', got: %s", out)
	}
}

func TestJobsCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("jobs")
	if err != nil {
		t.Fatalf("jobs failed: %v", err)
	}
	if !strings.Contains(out, "job-001") {
		t.Errorf("should contain job ID, got: %s", out)
	}
}

func TestDiagnoseCommand(t *testing.T) {
	beads := apiPkg.SampleBeads()
	run := newTestEnv(t, beads)
	out, err := run("diagnose", beads[0].ID)
	if err != nil {
		t.Fatalf("diagnose failed: %v", err)
	}
	if !strings.Contains(out, "── Bead ──") {
		t.Errorf("should contain bead header, got: %s", out)
	}
	if !strings.Contains(out, "Ready task") {
		t.Errorf("should contain bead subject, got: %s", out)
	}
}

// --- Auth & Setup Tests ---

func TestDoctorCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, _ := run("doctor")
	if !strings.Contains(out, "devdash") {
		t.Errorf("should contain version, got: %s", out)
	}
}

func TestTokenListCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("token", "list")
	if err != nil {
		t.Fatalf("token list failed: %v", err)
	}
	if !strings.Contains(out, "tok-1") {
		t.Errorf("should list tokens, got: %s", out)
	}
}

func TestTokenCreateCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("token", "create", "my-token")
	if err != nil {
		t.Fatalf("token create failed: %v", err)
	}
	if !strings.Contains(out, "tok-new") {
		t.Errorf("should contain new token ID, got: %s", out)
	}
}

func TestTokenRevokeCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("token", "revoke", "tok-1")
	if err != nil {
		t.Fatalf("token revoke failed: %v", err)
	}
	if !strings.Contains(out, "Revoked") {
		t.Errorf("should contain 'Revoked', got: %s", out)
	}
}

func TestProjectListCommand(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("project", "list")
	if err != nil {
		t.Fatalf("project list failed: %v", err)
	}
	if !strings.Contains(out, "test-project") {
		t.Errorf("should list projects, got: %s", out)
	}
}

// --- Help Topics Tests ---

func TestHelpTopicCLI(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("help", "cli")
	if err != nil {
		t.Fatalf("help cli failed: %v", err)
	}
	if !strings.Contains(out, "CLI Reference") {
		t.Errorf("should contain CLI Reference, got: %s", out)
	}
}

func TestHelpTopicWorkflow(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("help", "workflow")
	if err != nil {
		t.Fatalf("help workflow failed: %v", err)
	}
	if !strings.Contains(out, "Workflow") {
		t.Errorf("should contain Workflow, got: %s", out)
	}
}

func TestPrimeCommandSessionContext(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("prime")
	if err != nil {
		t.Fatalf("prime failed: %v", err)
	}
	if !strings.Contains(out, "Health") {
		t.Errorf("should contain project health stats, got: %s", out)
	}
	if !strings.Contains(out, "Output Formats") {
		t.Errorf("should contain output format guidance, got: %s", out)
	}
	if strings.Contains(out, "Rules (MANDATORY)") {
		t.Errorf("should NOT contain workflow rules (moved to static instructions), got: %s", out)
	}
}

func TestAgentSetupCodexDetection(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(origDir) //nolint:errcheck

	_ = os.WriteFile("AGENTS.md", []byte("# My project"), 0644)
	agents := detectAgents()
	found := false
	for _, a := range agents {
		if a == "codex" {
			found = true
		}
	}
	if !found {
		t.Errorf("detectAgents should find codex when AGENTS.md exists, got: %v", agents)
	}
}

func TestAgentSetupCodexWrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(origDir) //nolint:errcheck

	err := setupCodex("test-pid", "push", false)
	if err != nil {
		t.Fatalf("setupCodex failed: %v", err)
	}

	data, err := os.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "devdash") {
		t.Errorf("AGENTS.md should contain 'devdash', got: %s", content)
	}
	if !strings.Contains(content, "devdash prime") {
		t.Errorf("should mention devdash prime, got: %s", content)
	}
	if !strings.Contains(content, "test-pid") {
		t.Errorf("should contain project ID, got: %s", content)
	}
}

func TestAgentSetupCodexNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(origDir) //nolint:errcheck

	_ = os.WriteFile("AGENTS.md", []byte("# Existing devdash instructions"), 0644)
	err := setupCodex("test-pid", "push", false)
	if err != nil {
		t.Fatalf("setupCodex failed: %v", err)
	}

	data, _ := os.ReadFile("AGENTS.md")
	if !strings.HasPrefix(string(data), "# Existing") {
		t.Errorf("should preserve existing content, got: %s", string(data))
	}
}

func TestBuildInstructionsSharedCore(t *testing.T) {
	out := buildInstructions("test-pid", "push", agentConfig{})

	// Workflow section
	if !strings.Contains(out, "The Workflow") {
		t.Error("shared core should contain The Workflow section")
	}
	if !strings.Contains(out, "devdash create --title") {
		t.Error("shared core should contain create workflow step")
	}
	if !strings.Contains(out, "devdash update <id> --status=in_progress") {
		t.Error("shared core should contain update workflow step")
	}

	// Rules
	if !strings.Contains(out, "Issue-first") {
		t.Error("shared core should contain Issue-first rule")
	}
	if !strings.Contains(out, "Capture reflex") {
		t.Error("shared core should contain Capture reflex rule")
	}
	if !strings.Contains(out, "Close after push") {
		t.Error("shared core should respect closeOn parameter")
	}
	if !strings.Contains(out, "Never run git and devdash close in parallel") {
		t.Error("shared core should contain parallel execution warning")
	}

	// Close summaries
	if !strings.Contains(out, "Close Summaries") {
		t.Error("shared core should contain Close Summaries section")
	}
	if !strings.Contains(out, "--pr=URL") {
		t.Error("shared core should mention --pr=URL on close")
	}

	// Quick reference
	if !strings.Contains(out, "Quick Reference") {
		t.Error("shared core should contain Quick Reference section")
	}
	if !strings.Contains(out, "devdash ready") {
		t.Error("shared core should mention devdash ready in quick ref")
	}

	// On-demand references (without Session Startup)
	if !strings.Contains(out, "devdash help cli") {
		t.Error("shared core should contain on-demand help references")
	}
}

func TestBuildInstructionsCloseOnCommit(t *testing.T) {
	out := buildInstructions("test-pid", "commit", agentConfig{})
	if !strings.Contains(out, "Close after commit") {
		t.Error("should use 'commit' as closeOn gate")
	}
	if strings.Contains(out, "Close after push") {
		t.Error("should not mention push when closeOn is commit")
	}
}

func TestBuildInstructionsPreamblePostamble(t *testing.T) {
	out := buildInstructions("test-pid", "push", agentConfig{
		Preamble:  "PREAMBLE_MARKER",
		Postamble: "POSTAMBLE_MARKER",
	})
	preambleIdx := strings.Index(out, "PREAMBLE_MARKER")
	coreIdx := strings.Index(out, "The Workflow")
	postambleIdx := strings.Index(out, "POSTAMBLE_MARKER")

	if preambleIdx == -1 || coreIdx == -1 {
		t.Fatal("output should contain preamble and core")
	}
	if preambleIdx >= coreIdx {
		t.Error("preamble should appear before core")
	}
	// postamble is optional (can be empty), so check if present it comes after core
	if postambleIdx != -1 && coreIdx >= postambleIdx {
		t.Error("core should appear before postamble")
	}
}

func TestAllTemplatesShareCore(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(origDir) //nolint:errcheck

	_ = setupClaude("test-pid", "push", false)
	_ = setupCodex("test-pid", "push", false)
	_ = setupCopilot("test-pid", "push", false)

	claude, _ := os.ReadFile("CLAUDE.md")
	codex, _ := os.ReadFile("AGENTS.md")
	copilot, _ := os.ReadFile(".github/copilot-instructions.md")

	// All three must contain the same core content
	coreStrings := []string{
		"The Workflow",
		"Issue-first",
		"Capture reflex",
		"devdash create --title",
		"devdash update <id> --status=in_progress",
		"Never run git and devdash close in parallel",
		"Close after push",
		"devdash help cli",
		"Quick Reference",
	}
	for _, s := range coreStrings {
		if !strings.Contains(string(claude), s) {
			t.Errorf("Claude template missing core string: %s", s)
		}
		if !strings.Contains(string(codex), s) {
			t.Errorf("Codex template missing core string: %s", s)
		}
		if !strings.Contains(string(copilot), s) {
			t.Errorf("Copilot template missing core string: %s", s)
		}
	}
}

func TestSetupClaudeWrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(origDir) //nolint:errcheck

	err := setupClaude("test-pid", "push", false)
	if err != nil {
		t.Fatalf("setupClaude failed: %v", err)
	}

	data, err := os.ReadFile("CLAUDE.md")
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "devdash:agent-instructions") {
		t.Error("should contain agent-instructions markers")
	}
	if !strings.Contains(content, "DevDash — Task Tracking") {
		t.Error("Claude template should contain task tracking header")
	}
	if !strings.Contains(content, "test-pid") {
		t.Error("should contain project ID")
	}
	if !strings.Contains(content, "Quick Reference") {
		t.Error("Claude template should contain Quick Reference section")
	}
}

func TestSetupCopilotWrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(origDir) //nolint:errcheck

	err := setupCopilot("test-pid", "push", false)
	if err != nil {
		t.Fatalf("setupCopilot failed: %v", err)
	}

	data, err := os.ReadFile(".github/copilot-instructions.md")
	if err != nil {
		t.Fatalf("copilot-instructions.md not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "devdash") {
		t.Error("should contain devdash instructions")
	}
	if !strings.Contains(content, "test-pid") {
		t.Error("should contain project ID")
	}
}

func TestHelpTopicCLIDescriptions(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("help", "cli")
	if err != nil {
		t.Fatalf("help cli failed: %v", err)
	}
	if !strings.Contains(out, "choose what to work on next") {
		t.Errorf("ready description should mention 'choose what to work on next', got: %s", out)
	}
	if !strings.Contains(out, "Start here when the user already named the issue") {
		t.Errorf("show description should mention starting point, got: %s", out)
	}
}

func TestHelpTopicUnknown(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("help", "nonexistent")
	if err != nil {
		t.Fatalf("help nonexistent failed: %v", err)
	}
	if !strings.Contains(out, "Unknown help topic") {
		t.Errorf("should show unknown topic message, got: %s", out)
	}
}
