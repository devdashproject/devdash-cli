package commands

import (
	"strings"
	"testing"
	"time"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
)

func at(min int) apiPkg.FlexTime {
	return apiPkg.FlexTime{Time: time.Date(2026, 10, 4, 10, min, 0, 0, time.UTC)}
}

func TestReadyHidesParentWithOpenChildren(t *testing.T) {
	parent := "pppp0000-0000-0000-0000-000000000001"
	beads := []apiPkg.Bead{
		{ID: parent, LocalBeadID: "p-1", Subject: "Parent feature", Status: "pending", Priority: 1},
		{ID: "cccc0000-0000-0000-0000-000000000002", LocalBeadID: "c-1", Subject: "Child step", Status: "pending", Priority: 2, ParentBeadID: parent},
		{ID: "dddd0000-0000-0000-0000-000000000003", LocalBeadID: "p-2", Subject: "Finished parent", Status: "pending", Priority: 2},
		{ID: "eeee0000-0000-0000-0000-000000000004", LocalBeadID: "c-2", Subject: "Done child", Status: "completed", ParentBeadID: "dddd0000-0000-0000-0000-000000000003"},
	}
	run := newTestEnv(t, beads)
	out, err := run("ready")
	if err != nil {
		t.Fatalf("ready failed: %v", err)
	}
	if strings.Contains(out, "Parent feature") {
		t.Errorf("parent with open children should not be ready:\n%s", out)
	}
	if !strings.Contains(out, "Child step") || !strings.Contains(out, "Finished parent") {
		t.Errorf("child and parent-without-open-children should be ready:\n%s", out)
	}
}

func TestReadyTieBreakSortOrderThenOldest(t *testing.T) {
	two, one := 2, 1
	beads := []apiPkg.Bead{
		{ID: "a0000000-0000-0000-0000-000000000001", Subject: "Newest", Status: "pending", Priority: 2, CreatedAt: at(30)},
		{ID: "a0000000-0000-0000-0000-000000000002", Subject: "Oldest", Status: "pending", Priority: 2, CreatedAt: at(10)},
		{ID: "a0000000-0000-0000-0000-000000000003", Subject: "Middle", Status: "pending", Priority: 2, CreatedAt: at(20)},
		{ID: "a0000000-0000-0000-0000-000000000004", Subject: "Ordered2", Status: "pending", Priority: 2, CreatedAt: at(40), SortOrder: &two},
		{ID: "a0000000-0000-0000-0000-000000000005", Subject: "Ordered1", Status: "pending", Priority: 2, CreatedAt: at(50), SortOrder: &one},
	}
	run := newTestEnv(t, beads)
	out, err := run("ready")
	if err != nil {
		t.Fatalf("ready failed: %v", err)
	}
	want := []string{"Ordered1", "Ordered2", "Oldest", "Middle", "Newest"}
	last := -1
	for _, w := range want {
		i := strings.Index(out, w)
		if i < 0 || i < last {
			t.Fatalf("expected order %v, got:\n%s", want, out)
		}
		last = i
	}
}
