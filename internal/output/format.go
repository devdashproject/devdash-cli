package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/devdashproject/devdash-cli/internal/api"
)

// Status icons matching the Bash version exactly.
const (
	IconPending    = "○"
	IconInProgress = "●"
	IconCompleted  = "✓"
	IconFailed     = "✗"
	IconStale      = "⚠"
	IconSkipped    = "⊘"
)

// StatusIcon returns the icon for a bead status.
func StatusIcon(status string) string {
	switch status {
	case "pending":
		return IconPending
	case "in_progress":
		return IconInProgress
	case "completed":
		return IconCompleted
	case "failed":
		return IconFailed
	default:
		return IconPending
	}
}

// JobStatusIcon returns the icon for a job status.
func JobStatusIcon(status string) string {
	switch status {
	case "queued":
		return IconPending
	case "running":
		return IconInProgress
	case "completed":
		return IconCompleted
	case "failed":
		return IconFailed
	case "skipped":
		return IconSkipped
	default:
		return IconPending
	}
}

// FormatReadyLine formats a bead for the `ready` command output.
func FormatReadyLine(b api.Bead) string {
	parts := []string{
		fmt.Sprintf("%s %s", IconPending, beadID(b)),
		fmt.Sprintf("[P%d]", b.Priority),
	}
	if b.BurnIntelligence != nil && b.BurnIntelligence.AutomabilityGrade != "" {
		parts = append(parts, fmt.Sprintf("[%s]", b.BurnIntelligence.AutomabilityGrade))
	}
	parts = append(parts, fmt.Sprintf("[%s]", b.BeadType))
	parts = append(parts, fmt.Sprintf("- %s", b.Subject))
	return strings.Join(parts, " ")
}

// FormatListLine formats a bead for the `list` command output.
func FormatListLine(b api.Bead) string {
	icon := StatusIcon(b.Status)
	suffix := ""
	if b.Status == "pending" && len(b.BlockedBy) > 0 {
		suffix = "(blocked)"
	}
	return fmt.Sprintf("%s %s [P%d] [%s] - %s%s",
		icon, beadID(b), b.Priority, b.BeadType, b.Subject, suffix)
}

// FormatBlockedLine formats a bead for the `blocked` command output.
func FormatBlockedLine(b api.Bead) string {
	blockers := strings.Join(shortIDs(b.BlockedBy), ", ")
	return fmt.Sprintf("%s %s [P%d] - %s  blocked by: %s",
		IconPending, beadID(b), b.Priority, b.Subject, blockers)
}

// FormatStaleLine formats a bead for the `stale` command output.
func FormatStaleLine(b api.Bead) string {
	return fmt.Sprintf("  %s %s — %s\n    Stale for %dm (since %s)",
		IconStale, beadID(b), b.Subject, b.StaleMinutes, b.StaleSince)
}

// FormatStats formats project statistics.
func FormatStats(total, pending, inProgress, completed, blocked, ready int) string {
	return fmt.Sprintf("Total:       %d\nPending:     %d\nIn Progress: %d\nCompleted:   %d\nBlocked:     %d\nReady:       %d",
		total, pending, inProgress, completed, blocked, ready)
}

// FormatJobLine formats a job for listing.
func FormatJobLine(j api.Job) string {
	icon := JobStatusIcon(j.Status)
	prompt := j.Prompt
	if len(prompt) > 70 {
		prompt = prompt[:70] + "..."
	}
	return fmt.Sprintf("%s %s [%s] %s  %s",
		icon, shortID(j.ID), j.Status, prompt, j.CreatedAt)
}

// FormatJobFailureLine formats a failed job for listing.
func FormatJobFailureLine(j api.Job) string {
	errMsg := j.Error
	if len(errMsg) > 80 {
		errMsg = errMsg[:80]
	}
	return fmt.Sprintf("%s %s %s  %s", IconFailed, shortID(j.ID), errMsg, j.CreatedAt)
}

// ParseSince converts a --since value to a time.Time.
// Supports: Nh (hours), Nd (days), Nw (weeks), YYYY-MM-DD.
func ParseSince(since string) (time.Time, error) {
	if since == "" {
		return time.Time{}, fmt.Errorf("empty --since value")
	}

	now := time.Now().UTC()

	// Try date format first
	if t, err := time.Parse("2006-01-02", since); err == nil {
		return t, nil
	}

	if len(since) < 2 {
		return time.Time{}, fmt.Errorf("invalid --since format: %q", since)
	}

	unit := since[len(since)-1]
	numStr := since[:len(since)-1]
	var n int
	if _, err := fmt.Sscanf(numStr, "%d", &n); err != nil {
		return time.Time{}, fmt.Errorf("invalid --since format: %q", since)
	}

	switch unit {
	case 'h':
		return now.Add(-time.Duration(n) * time.Hour), nil
	case 'd':
		return now.AddDate(0, 0, -n), nil
	case 'w':
		return now.AddDate(0, 0, -n*7), nil
	default:
		return time.Time{}, fmt.Errorf("invalid --since unit %q (use h, d, or w)", string(unit))
	}
}

// FormatSinceISO converts a --since value to an ISO 8601 string for API queries.
func FormatSinceISO(since string) (string, error) {
	t, err := ParseSince(since)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02T15:04:05.000Z"), nil
}

func beadID(b api.Bead) string {
	if b.LocalBeadID != "" {
		return b.LocalBeadID
	}
	return shortID(b.ID)
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

func shortIDs(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = shortID(id)
	}
	return out
}
