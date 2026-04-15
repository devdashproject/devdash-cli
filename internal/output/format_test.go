package output

import (
	"strings"
	"testing"

	"github.com/devdashproject/devdash-cli/internal/api"
)

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"pending", "○"},
		{"in_progress", "●"},
		{"completed", "✓"},
		{"failed", "✗"},
		{"unknown", "○"},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			got := StatusIcon(tc.status)
			if got != tc.want {
				t.Errorf("StatusIcon(%q) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestJobStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"queued", "○"},
		{"running", "●"},
		{"completed", "✓"},
		{"failed", "✗"},
		{"skipped", "⊘"},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			got := JobStatusIcon(tc.status)
			if got != tc.want {
				t.Errorf("JobStatusIcon(%q) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestFormatReadyLine(t *testing.T) {
	bead := api.Bead{
		ID: "aaaa0000-1111-2222-3333-444444444444", LocalBeadID: "test-1",
		Subject: "My task", Priority: 1, BeadType: "task",
	}

	line := FormatReadyLine(bead)

	if !strings.HasPrefix(line, "○ test-1") {
		t.Errorf("should start with pending icon and local ID, got: %s", line)
	}
	if !strings.Contains(line, "[P1]") {
		t.Errorf("should contain priority, got: %s", line)
	}
	if !strings.Contains(line, "[task]") {
		t.Errorf("should contain type, got: %s", line)
	}
	if !strings.Contains(line, "- My task") {
		t.Errorf("should contain subject, got: %s", line)
	}
}

func TestFormatReadyLineWithScore(t *testing.T) {
	bead := api.Bead{
		ID: "aaaa", LocalBeadID: "test-1",
		Subject: "Scored task", Priority: 2, BeadType: "feature",
		BurnIntelligence: &api.BurnIntelligence{
			AutomabilityGrade: "A",
			AutomabilityScore: 85,
		},
	}

	line := FormatReadyLine(bead)
	if !strings.Contains(line, "[A]") {
		t.Errorf("should contain automability grade, got: %s", line)
	}
}

func TestFormatListLine(t *testing.T) {
	tests := []struct {
		name   string
		bead   api.Bead
		expect string
	}{
		{
			"pending",
			api.Bead{ID: "aaaa", LocalBeadID: "t-1", Subject: "Task", Status: "pending", Priority: 0, BeadType: "task"},
			"○ t-1 [P0] [task] - Task",
		},
		{
			"in_progress",
			api.Bead{ID: "bbbb", LocalBeadID: "t-2", Subject: "Feature", Status: "in_progress", Priority: 1, BeadType: "feature"},
			"● t-2 [P1] [feature] - Feature",
		},
		{
			"completed",
			api.Bead{ID: "cccc", LocalBeadID: "t-3", Subject: "Bug", Status: "completed", Priority: 2, BeadType: "bug"},
			"✓ t-3 [P2] [bug] - Bug",
		},
		{
			"blocked",
			api.Bead{ID: "dddd", LocalBeadID: "t-4", Subject: "Blocked", Status: "pending", Priority: 1, BeadType: "task", BlockedBy: []string{"x"}},
			"○ t-4 [P1] [task] - Blocked(blocked)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatListLine(tc.bead)
			if got != tc.expect {
				t.Errorf("FormatListLine() = %q, want %q", got, tc.expect)
			}
		})
	}
}

func TestFormatBlockedLine(t *testing.T) {
	bead := api.Bead{
		ID: "aaaa0000-1111-2222-3333-444444444444", LocalBeadID: "test-1",
		Subject: "Task", Priority: 2, BlockedBy: []string{"bbbb0000-1111-2222-3333-444444444444"},
	}

	line := FormatBlockedLine(bead)
	if !strings.Contains(line, "blocked by:") {
		t.Errorf("should contain 'blocked by:', got: %s", line)
	}
	if !strings.Contains(line, "bbbb0000") {
		t.Errorf("should contain truncated blocker ID, got: %s", line)
	}
}

func TestFormatStats(t *testing.T) {
	out := FormatStats(10, 5, 2, 3, 1, 4)
	if !strings.Contains(out, "Total:       10") {
		t.Errorf("should contain Total, got: %s", out)
	}
	if !strings.Contains(out, "Blocked:     1") {
		t.Errorf("should contain Blocked, got: %s", out)
	}
	if !strings.Contains(out, "Ready:       4") {
		t.Errorf("should contain Ready, got: %s", out)
	}
}

func TestFormatStaleLine(t *testing.T) {
	bead := api.Bead{
		ID: "aaaa", LocalBeadID: "test-1",
		Subject: "Stale task", StaleMinutes: 45, StaleSince: "2026-03-27T10:00:00Z",
	}

	line := FormatStaleLine(bead)
	if !strings.Contains(line, "⚠") {
		t.Errorf("should contain stale icon, got: %s", line)
	}
	if !strings.Contains(line, "45m") {
		t.Errorf("should contain stale minutes, got: %s", line)
	}
}

func TestParseSince(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"24h", false},
		{"7d", false},
		{"2w", false},
		{"2026-03-27", false},
		{"", true},
		{"abc", true},
		{"x", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			_, err := ParseSince(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseSince(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestFormatSinceISO(t *testing.T) {
	iso, err := FormatSinceISO("7d")
	if err != nil {
		t.Fatalf("FormatSinceISO(\"7d\") failed: %v", err)
	}
	if !strings.Contains(iso, "T") || !strings.HasSuffix(iso, "Z") {
		t.Errorf("should be ISO 8601 format, got: %s", iso)
	}
}

func TestFormatJobLine(t *testing.T) {
	job := api.Job{
		ID:        "job-0001-0000-0000-000000000001",
		Status:    "completed",
		Prompt:    "Implement the feature",
		CreatedAt: "2026-03-27T12:00:00Z",
	}

	line := FormatJobLine(job)
	if !strings.Contains(line, "✓") {
		t.Errorf("should contain completed icon, got: %s", line)
	}
	if !strings.Contains(line, "job-0001") {
		t.Errorf("should contain truncated job ID, got: %s", line)
	}
}

func TestFormatJobFailureLine(t *testing.T) {
	job := api.Job{
		ID:        "job-fail-0000-0000-000000000001",
		Status:    "failed",
		Error:     "Connection timeout",
		CreatedAt: "2026-03-27T12:00:00Z",
	}

	line := FormatJobFailureLine(job)
	if !strings.Contains(line, "✗") {
		t.Errorf("should contain failed icon, got: %s", line)
	}
	if !strings.Contains(line, "Connection timeout") {
		t.Errorf("should contain error message, got: %s", line)
	}
}
