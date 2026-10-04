package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/output"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newListCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long: `List all issues for the current project, sorted by priority.

Results can be narrowed with --status (pending, in_progress, blocked,
completed, ready, failed, archived, or the shorthand open), --since (accepts relative durations
like 2h, 3d, 1w or an absolute YYYY-MM-DD date filtering on updatedAt),
--parent (show only children of a specific issue), and --mine (show
only issues assigned to you). --tree indents children under their parents.

Issues are sorted by priority, then sort order (see 'update --sort-order'),
then oldest first.

The "open" shorthand enumerates the whole active backlog in one call —
pending, in_progress, and blocked beads together — so a whole-backlog
sweep cannot miss one of those buckets.

When no issues match the filters, a message is printed to stderr.

Icons: ○ pending  ● in progress  ⊘ blocked  ✓ completed  ✗ failed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			if v, _ := cmd.Flags().GetString("status"); v != "" {
				if err := validateStatus(v, "open"); err != nil {
					return err
				}
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

			if parent != "" {
				// Accept short IDs and local IDs, like every other command
				if parent, err = resolve.ID(parent, beads); err != nil {
					return fmt.Errorf("--parent: %w", err)
				}
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

			sort.SliceStable(filtered, func(i, j int) bool {
				return listLess(filtered[i], filtered[j])
			})

			if len(filtered) == 0 {
				fmt.Fprintln(os.Stderr, "No issues found.")
				return nil
			}

			if tree, _ := cmd.Flags().GetBool("tree"); tree {
				printTree(filtered)
				return nil
			}
			for _, b := range filtered {
				fmt.Println(output.FormatListLine(b))
			}
			return nil
		},
	}
	cmd.Flags().String("status", "", "Filter by status: pending, ready, in_progress, completed, blocked, failed, archived, or open (pending+in_progress+blocked)")
	cmd.Flags().String("since", "", "Filter by updatedAt (Nh, Nd, Nw, or YYYY-MM-DD)")
	cmd.Flags().String("parent", "", "Filter by parent bead ID")
	cmd.Flags().Bool("mine", false, "Show only issues assigned to you")
	cmd.Flags().Bool("tree", false, "Show children indented under their parents")
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

// listLess orders issues by priority, then explicit sort order (unset last),
// then creation time (oldest first).
func listLess(a, b api.Bead) bool {
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

// printTree prints issues with children indented under their parents. Issues
// whose parent isn't in the list are shown at the top level.
func printTree(beads []api.Bead) {
	inList := make(map[string]bool, len(beads))
	for _, b := range beads {
		inList[b.ID] = true
	}
	children := map[string][]api.Bead{}
	var roots []api.Bead
	for _, b := range beads {
		if b.ParentBeadID != "" && inList[b.ParentBeadID] {
			children[b.ParentBeadID] = append(children[b.ParentBeadID], b)
		} else {
			roots = append(roots, b)
		}
	}
	var walk func(b api.Bead, depth int)
	walk = func(b api.Bead, depth int) {
		fmt.Println(strings.Repeat("  ", depth) + output.FormatListLine(b))
		for _, c := range children[b.ID] {
			walk(c, depth+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}
}
