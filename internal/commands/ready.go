package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/output"
	"github.com/spf13/cobra"
)

func newReadyCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ready",
		Short: "Pending, unblocked issues sorted by priority",
		Long: `Show pending, unblocked issues sorted by automability score then priority.

This is the "what should I work on next?" command. It filters out completed,
in-progress, and blocked issues, "thought" issues, and parent issues that
still have open children (work the children; the parent closes when they
do), leaving only actionable work. Results are ranked by automability,
then priority, then sort order (see 'update --sort-order'), then oldest
first, so a plan's steps come out in the order they were created.

Use --since to narrow results to issues created within a time window
(e.g. --since=7d, --since=2h, or --since=2025-01-01).

Icons: ○ pending  ● in progress  ⊘ blocked  ✓ completed  ✗ failed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			since, _ := cmd.Flags().GetString("since")
			var sinceFilter string
			if since != "" {
				sinceFilter, err = output.FormatSinceISO(since)
				if err != nil {
					return err
				}
			}

			completedIDs := make(map[string]bool)
			hasOpenChildren := make(map[string]bool)
			for _, b := range beads {
				if b.Status == "completed" {
					completedIDs[b.ID] = true
				}
				if b.ParentBeadID != "" && b.Status != "completed" && b.Status != "archived" {
					hasOpenChildren[b.ParentBeadID] = true
				}
			}

			var ready []api.Bead
			for _, b := range beads {
				if b.Status != "pending" {
					continue
				}
				if b.BeadType == "thought" {
					continue
				}
				if sinceFilter != "" && b.CreatedAt.Format("2006-01-02T15:04:05.000Z") < sinceFilter {
					continue
				}
				if isBlocked(b, completedIDs) || hasOpenChildren[b.ID] {
					continue
				}
				ready = append(ready, b)
			}

			sort.SliceStable(ready, func(i, j int) bool {
				return readyLess(ready[i], ready[j])
			})

			if len(ready) == 0 {
				fmt.Fprintln(os.Stderr, "No ready issues.")
				return nil
			}

			for _, b := range ready {
				fmt.Println(output.FormatReadyLine(b))
			}
			return nil
		},
	}
	cmd.Flags().String("since", "", "Filter by createdAt (Nh, Nd, Nw, or YYYY-MM-DD)")
	return cmd
}

// isEffectivelyBlocked reports whether a bead is waiting on something: the server
// marks beads "blocked" when a dependency is added, and pending beads can still
// carry unfinished dependencies.
func isEffectivelyBlocked(b api.Bead, completedIDs map[string]bool) bool {
	if b.Status == "blocked" {
		return true
	}
	return b.Status == "pending" && len(b.BlockedBy) > 0 && isBlocked(b, completedIDs)
}

// readyLess orders ready issues: automability (high first), priority,
// explicit sort order (unset last), then creation time (oldest first).
func readyLess(a, b api.Bead) bool {
	if sa, sb := automabilityScore(a), automabilityScore(b); sa != sb {
		return sa > sb
	}
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	if (a.SortOrder == nil) != (b.SortOrder == nil) {
		return a.SortOrder != nil
	}
	if a.SortOrder != nil && *a.SortOrder != *b.SortOrder {
		return *a.SortOrder < *b.SortOrder
	}
	return a.CreatedAt.Before(b.CreatedAt.Time)
}

func isBlocked(b api.Bead, completedIDs map[string]bool) bool {
	for _, dep := range b.BlockedBy {
		if !completedIDs[dep] {
			return true
		}
	}
	return false
}

func automabilityScore(b api.Bead) int {
	if b.BurnIntelligence != nil {
		return b.BurnIntelligence.AutomabilityScore
	}
	return 0
}
