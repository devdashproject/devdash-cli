package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/output"
	"github.com/spf13/cobra"
)

func newListCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long: `List all issues for the current project, sorted by priority.

Results can be narrowed with --status (pending, in_progress, completed),
--since (accepts relative durations like 2h, 3d, 1w or an absolute
YYYY-MM-DD date filtering on updatedAt), and --parent (show only children
of a specific bead ID).

When no issues match the filters, a message is printed to stderr.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			statusFilter, _ := cmd.Flags().GetString("status")
			since, _ := cmd.Flags().GetString("since")
			parent, _ := cmd.Flags().GetString("parent")

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

			var filtered []api.Bead
			for _, b := range beads {
				if statusFilter != "" && b.Status != statusFilter {
					continue
				}
				if sinceFilter != "" && b.UpdatedAt.Format("2006-01-02T15:04:05.000Z") < sinceFilter {
					continue
				}
				if parent != "" && b.ParentBeadID != parent {
					continue
				}
				filtered = append(filtered, b)
			}

			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].Priority < filtered[j].Priority
			})

			if len(filtered) == 0 {
				fmt.Fprintln(os.Stderr, "No issues found.")
				return nil
			}

			for _, b := range filtered {
				fmt.Println(output.FormatListLine(b))
			}
			return nil
		},
	}
	cmd.Flags().String("status", "", "Filter by status: pending, in_progress, completed")
	cmd.Flags().String("since", "", "Filter by updatedAt (Nh, Nd, Nw, or YYYY-MM-DD)")
	cmd.Flags().String("parent", "", "Filter by parent bead ID")
	return cmd
}
