package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/output"
	"github.com/spf13/cobra"
)

func newReadyCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ready",
		Short: "Pending, unblocked issues sorted by priority",
		Long: `Show pending, unblocked issues sorted by automability score then priority.

This is the "what should I work on next?" command. It filters out completed,
in-progress, and blocked issues, as well as "thought" type beads, leaving
only actionable work. Results are ranked so the most automatable, highest
priority issues appear first.

Use --since to narrow results to issues created within a time window
(e.g. --since=7d, --since=2h, or --since=2025-01-01).`,
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
			for _, b := range beads {
				if b.Status == "completed" {
					completedIDs[b.ID] = true
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
				if isBlocked(b, completedIDs) {
					continue
				}
				ready = append(ready, b)
			}

			sort.Slice(ready, func(i, j int) bool {
				si, sj := automabilityScore(ready[i]), automabilityScore(ready[j])
				if si != sj {
					return si > sj
				}
				return ready[i].Priority < ready[j].Priority
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
