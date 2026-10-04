package commands

import (
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
)

func TestValidateStatus(t *testing.T) {
	for _, ok := range beadStatuses {
		if err := validateStatus(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for in, want := range map[string]string{"in-progress": `did you mean "in_progress"`, "InProgress": `did you mean "in_progress"`, "Completed": `did you mean "completed"`, "done": "valid statuses: pending"} {
		if err := validateStatus(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("validateStatus(%q) = %v, want %q", in, err, want)
		}
	}
	if err := validateStatus("open", "open"); err != nil {
		t.Errorf("extra statuses should be allowed: %v", err)
	}
}

func TestUpdateAndListRejectBadStatusBeforeCallingServer(t *testing.T) {
	run := newTestEnv(t, apiPkg.SampleBeads())
	if _, err := run("update", "aaaa0000", "--status=in-progress"); err == nil || !strings.Contains(err.Error(), "in_progress") {
		t.Errorf("update should suggest in_progress, got: %v", err)
	}
	if _, err := run("list", "--status=bogus"); err == nil || !strings.Contains(err.Error(), "open") {
		t.Errorf("list should list valid statuses incl. open, got: %v", err)
	}
	if _, err := run("list", "--status=open"); err != nil {
		t.Errorf("list --status=open should work: %v", err)
	}
}
