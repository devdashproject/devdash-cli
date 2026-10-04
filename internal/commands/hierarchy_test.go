package commands

import (
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
)

func hierarchyBeads() []apiPkg.Bead {
	parent := "pa110000-0000-0000-0000-000000000001"
	return []apiPkg.Bead{
		{ID: parent, Subject: "Parent feature", Status: "in_progress", Priority: 1, CreatedAt: at(0)},
		{ID: "c2220000-0000-0000-0000-000000000002", Subject: "Second step", Status: "pending", Priority: 2, ParentBeadID: parent, CreatedAt: at(2)},
		{ID: "c1110000-0000-0000-0000-000000000003", Subject: "First step", Status: "completed", Priority: 2, ParentBeadID: parent, CreatedAt: at(1)},
		{ID: "s0000000-0000-0000-0000-000000000004", Subject: "Standalone", Status: "pending", Priority: 3, CreatedAt: at(3)},
	}
}

func TestListTreeIndentsChildrenInOrder(t *testing.T) {
	run := newTestEnv(t, hierarchyBeads())
	out, err := run("list", "--tree")
	if err != nil {
		t.Fatalf("list --tree failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 || !strings.Contains(lines[0], "Parent feature") ||
		!strings.HasPrefix(lines[1], "  ") || !strings.Contains(lines[1], "First step") ||
		!strings.HasPrefix(lines[2], "  ") || !strings.Contains(lines[2], "Second step") ||
		strings.HasPrefix(lines[3], " ") || !strings.Contains(lines[3], "Standalone") {
		t.Errorf("unexpected tree:\n%s", out)
	}
}

func TestListParentAcceptsShortID(t *testing.T) {
	run := newTestEnv(t, hierarchyBeads())
	out, err := run("list", "--parent=pa11")
	if err != nil {
		t.Fatalf("list --parent=<prefix> failed: %v", err)
	}
	if !strings.Contains(out, "First step") || strings.Contains(out, "Standalone") {
		t.Errorf("--parent prefix should show only the children:\n%s", out)
	}
}

func TestShowPrettyListsChildren(t *testing.T) {
	run := newTestEnv(t, hierarchyBeads())
	out, err := run("show", "pa11", "--pretty")
	if err != nil {
		t.Fatalf("show --pretty failed: %v", err)
	}
	i, j := strings.Index(out, "First step"), strings.Index(out, "Second step")
	if !strings.Contains(out, "Children (2):") || i < 0 || j < i {
		t.Errorf("show --pretty should list children oldest first:\n%s", out)
	}
}
