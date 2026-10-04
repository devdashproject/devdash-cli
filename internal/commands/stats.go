package commands

import (
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/output"
	"github.com/spf13/cobra"
)

func newStatsCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Project health: open/closed/blocked counts",
		Long: `Show project health metrics at a glance.

Prints total issue count along with breakdowns by status: pending,
in_progress, completed, blocked, and ready. "Blocked" counts issues the
server marked blocked plus pending issues whose dependencies have not
completed; "ready" counts pending issues that are unblocked. Issues in
other states (failed, archived) are shown as "Other".

Use this as a quick pulse-check on overall project state without
having to list and scan individual issues.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			completedIDs := make(map[string]bool)
			var pending, inProgress, completed, other int
			for _, b := range beads {
				switch b.Status {
				case "pending":
					pending++
				case "in_progress":
					inProgress++
				case "completed":
					completed++
					completedIDs[b.ID] = true
				case "blocked":
				default:
					other++
				}
			}

			var blocked, ready int
			for _, b := range beads {
				if isEffectivelyBlocked(b, completedIDs) {
					blocked++
				} else if b.Status == "pending" {
					ready++
				}
			}

			total := len(beads)

			fmt.Println(output.FormatStats(total, pending, inProgress, completed, blocked, ready, other))
			return nil
		},
	}
}
