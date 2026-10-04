package commands

import (
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
)

func TestListOrdersBySortOrderThenAge(t *testing.T) {
	two, one := 2, 1
	beads := []apiPkg.Bead{
		{ID: "a0000000-0000-0000-0000-000000000001", Subject: "Unordered old", Status: "pending", Priority: 2, CreatedAt: at(1)},
		{ID: "a0000000-0000-0000-0000-000000000002", Subject: "Ordered two", Status: "pending", Priority: 2, CreatedAt: at(2), SortOrder: &two},
		{ID: "a0000000-0000-0000-0000-000000000003", Subject: "Ordered one", Status: "pending", Priority: 2, CreatedAt: at(3), SortOrder: &one},
		{ID: "a0000000-0000-0000-0000-000000000004", Subject: "Urgent", Status: "pending", Priority: 0, CreatedAt: at(4)},
	}
	run := newTestEnv(t, beads)
	out, err := run("list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	last := -1
	for _, w := range []string{"Urgent", "Ordered one", "Ordered two", "Unordered old"} {
		i := strings.Index(out, w)
		if i < 0 || i < last {
			t.Fatalf("want priority, then sort order, then age; got:\n%s", out)
		}
		last = i
	}
}

func TestListParentUnknownIDFails(t *testing.T) {
	run := newTestEnv(t, hierarchyBeads())
	if _, err := run("list", "--parent=zzzz"); err == nil || !strings.Contains(err.Error(), "--parent") {
		t.Errorf("unknown --parent should fail and name the flag, got: %v", err)
	}
}

func TestShowPrettyWithoutChildren(t *testing.T) {
	run := newTestEnv(t, hierarchyBeads())
	out, err := run("show", "s000", "--pretty")
	if err != nil || strings.Contains(out, "Children") {
		t.Errorf("issue without children should have no Children section, got %q (%v)", out, err)
	}
}

func TestProjectCreateHintOutsideRepo(t *testing.T) {
	t.Chdir(t.TempDir()) // not a git repo
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("project", "create", "--name=project-one")
	if err != nil || !strings.Contains(out, "in your repo, or pass --project=proj-000") {
		t.Errorf("outside a repo, project create should suggest link or --project, got %q (%v)", out, err)
	}
}

func TestUpdateParentToCompletedParentWarns(t *testing.T) {
	beads := hierarchyBeads()
	beads[0].Status = "completed"
	run := newTestEnv(t, beads)
	out, err := run("update", "s000", "--parent=pa11")
	if err != nil || !strings.Contains(out, "is already completed") {
		t.Errorf("moving an issue under a completed parent should warn, got %q (%v)", out, err)
	}
}
