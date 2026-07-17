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

Results can be narrowed with --status (pending, in_progress, blocked,
completed, or the shorthand open), --since (accepts relative durations
like 2h, 3d, 1w or an absolute YYYY-MM-DD date filtering on updatedAt),
--parent (show only children of a specific bead ID), and --mine (show
only beads assigned to you).

The "open" shorthand enumerates the whole active backlog in one call —
pending, in_progress, and blocked beads together — so a whole-backlog
sweep cannot miss one of those buckets.

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
			mine, _ := cmd.Flags().GetBool("mine")

			var sinceFilter string
			if since != "" {
				sinceFilter, err = output.FormatSinceISO(since)
				if err != nil {
					return err
				}
			}

			var currentUserID string
			if mine {
				user, err := api.JSON[api.CurrentUser](d.Client.Get("/auth/me"))
				if err != nil {
					return err
				}
				currentUserID = user.ID
			}

			var filtered []api.Bead
			for _, b := range beads {
				if !statusMatches(b, statusFilter) {
					continue
				}
				if sinceFilter != "" && b.UpdatedAt.Format("2006-01-02T15:04:05.000Z") < sinceFilter {
					continue
				}
				if parent != "" && b.ParentBeadID != parent {
					continue
				}
				if mine && b.AssignedTo != currentUserID {
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
	cmd.Flags().String("status", "", "Filter by status: pending, in_progress, blocked, completed, open")
	cmd.Flags().String("since", "", "Filter by updatedAt (Nh, Nd, Nw, or YYYY-MM-DD)")
	cmd.Flags().String("parent", "", "Filter by parent bead ID")
	cmd.Flags().Bool("mine", false, "Show only issues assigned to you")
	return cmd
}

// statusMatches reports whether bead b passes the --status filter. Alongside
// exact matches on a stored status (pending, in_progress, blocked, completed,
// failed) it understands the shorthand "open": the active backlog of pending,
// in_progress, and blocked beads, enumerable in a single call so a whole-backlog
// sweep cannot miss one of those buckets — blocked was the bucket missed during
// the 2026-07-16 cull. Terminal states (completed, failed) are excluded.
func statusMatches(b api.Bead, filter string) bool {
	switch filter {
	case "":
		return true
	case "open":
		return b.Status == "pending" || b.Status == "in_progress" || b.Status == "blocked"
	default:
		return b.Status == filter
	}
}
