package commands

import (
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
)

// The server sets status "blocked" when a dependency is added.
func beadsWithServerBlocked() []apiPkg.Bead {
	beads := apiPkg.SampleBeads()
	beads = append(beads,
		apiPkg.Bead{
			ID: "9999aaaa-0000-0000-0000-000000000008", LocalBeadID: "test-8",
			Subject: "Server blocked", Status: "blocked", Priority: 1, BeadType: "task",
			BlockedBy: []string{"cccc0000-0000-0000-0000-000000000003"},
		},
		apiPkg.Bead{
			ID: "9999bbbb-0000-0000-0000-000000000009", LocalBeadID: "test-9",
			Subject: "Archived", Status: "archived", Priority: 3, BeadType: "task",
		},
	)
	return beads
}

func TestBlockedIncludesServerBlockedStatus(t *testing.T) {
	run := newTestEnv(t, beadsWithServerBlocked())
	out, err := run("blocked")
	if err != nil {
		t.Fatalf("blocked failed: %v", err)
	}
	if !strings.Contains(out, "Server blocked") {
		t.Errorf("should list status=blocked bead, got: %s", out)
	}
	if !strings.Contains(out, "Blocked task") {
		t.Errorf("should still list pending bead with open deps, got: %s", out)
	}
}

func TestStatsCountsServerBlockedStatus(t *testing.T) {
	run := newTestEnv(t, beadsWithServerBlocked())
	out, err := run("stats")
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	// pending: test-1, test-2 (blocked), test-5, test-7 → ready = 3
	for _, want := range []string{"Total:       9", "Blocked:     2", "Ready:       3", "Other:       1"} {
		if !strings.Contains(out, want) {
			t.Errorf("stats missing %q, got:\n%s", want, out)
		}
	}
}
