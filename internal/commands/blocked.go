package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/output"
	"github.com/spf13/cobra"
)

func newBlockedCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "blocked",
		Short: "Pending issues with unsatisfied dependencies",
		Long: `Show pending issues that are waiting on unfinished dependencies.

An issue is considered blocked when it has at least one dependency that
has not yet been completed. Results are sorted by priority. Use this to
identify bottlenecks — the dependencies shown are what need to be
completed before these issues can move forward.`,
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
			for _, b := range beads {
				if b.Status == "completed" {
					completedIDs[b.ID] = true
				}
			}

			var blocked []api.Bead
			for _, b := range beads {
				if b.Status != "pending" || len(b.BlockedBy) == 0 {
					continue
				}
				if isBlocked(b, completedIDs) {
					blocked = append(blocked, b)
				}
			}

			sort.Slice(blocked, func(i, j int) bool {
				return blocked[i].Priority < blocked[j].Priority
			})

			if len(blocked) == 0 {
				fmt.Fprintln(os.Stderr, "No blocked issues.")
				return nil
			}

			for _, b := range blocked {
				fmt.Println(output.FormatBlockedLine(b))
			}
			return nil
		},
	}
}
