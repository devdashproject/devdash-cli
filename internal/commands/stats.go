package commands

import (
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/output"
	"github.com/spf13/cobra"
)

func newStatsCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Project health: open/closed/blocked counts",
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
			var pending, inProgress, completed int
			for _, b := range beads {
				switch b.Status {
				case "pending":
					pending++
				case "in_progress":
					inProgress++
				case "completed":
					completed++
					completedIDs[b.ID] = true
				}
			}

			var blocked int
			for _, b := range beads {
				if b.Status == "pending" && len(b.BlockedBy) > 0 && isBlocked(b, completedIDs) {
					blocked++
				}
			}

			ready := pending - blocked
			total := len(beads)

			fmt.Println(output.FormatStats(total, pending, inProgress, completed, blocked, ready))
			return nil
		},
	}
}
