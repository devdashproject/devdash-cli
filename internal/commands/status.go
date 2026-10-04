package commands

import (
	"fmt"
	"strings"
)

// beadStatuses are the statuses the server accepts.
var beadStatuses = []string{"pending", "ready", "in_progress", "completed", "blocked", "failed", "archived"}

// validateStatus checks a --status value before it reaches the server and
// suggests the right spelling for near misses like "in-progress".
func validateStatus(s string, extra ...string) error {
	allowed := append(append([]string{}, beadStatuses...), extra...)
	for _, a := range allowed {
		if s == a {
			return nil
		}
	}
	norm := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(s)))
	if norm == "inprogress" {
		norm = "in_progress"
	}
	for _, a := range allowed {
		if norm == a {
			return fmt.Errorf("unknown status %q — did you mean %q?", s, a)
		}
	}
	return fmt.Errorf("unknown status %q — valid statuses: %s", s, strings.Join(allowed, ", "))
}
