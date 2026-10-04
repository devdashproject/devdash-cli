package commands

import (
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
)

func TestPrettyBead(t *testing.T) {
	out, err := prettyBead([]byte(`{"id":"aaaa0000-1","subject":"Fix it","status":"completed","priority":1,"beadType":"bug",
		"parentBeadId":"pppp0000-0000","blockedBy":["bbbb0000-0000"],"blocks":[],"assigneeName":"Ada","description":"line1\nline2",
		"owner":null,"analysisResult":null,"createdAt":"2026-10-04T07:06:26.790Z",
		"completionResult":{"summary":"Done well","commitSha":"abc123","prUrl":"https://x/pr/1"}}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"aaaa0000-1  Fix it", "Status: completed   Priority: P1   Type: bug", "Parent:      pppp0000",
		"Blocked by:  bbbb0000", "Assignee:    Ada", "Created:     2026-10-04 07:06", "  line2", "Done well", "Commit:      abc123", "PR:          https://x/pr/1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "null") || strings.Contains(out, "Blocks:") {
		t.Errorf("should skip null/empty fields:\n%s", out)
	}
}

func TestPrettyCommentsAndActivityHideEmails(t *testing.T) {
	comments := []byte(`{"data":[{"authorType":"system","author":null,"content":"Assigned to a team member.","createdAt":"2026-10-04T07:06:26Z",
		"metadata":{"assigneeEmail":"someone@example.com"}},{"authorType":"user","author":{"displayName":"Ada","avatarUrl":"https://img"},"content":"hi","createdAt":"2026-10-04T08:00:00Z"}]}`)
	out, err := prettyComments(comments)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[2026-10-04 07:06] system: Assigned to a team member.") || !strings.Contains(out, "Ada: hi") {
		t.Errorf("unexpected comments view:\n%s", out)
	}
	activity := []byte(`{"data":[{"action":"bead.status_changed","actorType":"user","actor":{"displayName":"Ada","avatarUrl":"https://img"},
		"createdAt":"2026-10-04T07:06:26Z","metadata":{"fromStatus":"pending","toStatus":"in_progress","subject":"Fix it","assigneeEmail":"someone@example.com"}}]}`)
	out, err = prettyActivity(activity)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "bead.status_changed by Ada: pending → in_progress — Fix it") {
		t.Errorf("unexpected activity view:\n%s", out)
	}
	if strings.Contains(out, "@") || strings.Contains(out, "https://img") {
		t.Errorf("pretty views must not show emails or avatar URLs:\n%s", out)
	}
}

func TestCommentConfirmsAndUpdateEchoesChanges(t *testing.T) {
	run := newTestEnv(t, apiPkgSampleBeadsForPretty())
	out, err := run("comment", "aaaa0000", "--body=note")
	if err != nil || !strings.Contains(out, "Commented on aaaa0000") {
		t.Errorf("comment should confirm, got %q (%v)", out, err)
	}
	out, err = run("update", "aaaa0000", "--status=in_progress", "--priority=1")
	if err != nil || !strings.Contains(out, "priority=1") || !strings.Contains(out, "status=in_progress") {
		t.Errorf("update should echo what changed, got %q (%v)", out, err)
	}
}

func apiPkgSampleBeadsForPretty() []apiPkg.Bead { return apiPkg.SampleBeads() }
