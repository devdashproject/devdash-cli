package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newCloseCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:        "close <id> [<id>...]",
		Short:      "Close one or more issues",
		SuggestFor: []string{"done", "finish", "complete", "resolve"},
		Long: `Close one or more issues, marking them as completed.

Accepts one or multiple issue IDs. Full UUIDs work from any directory with no
project configured (the server uses each issue's own project); short prefixes
are resolved in the current project. Optionally attach
a completion summary, the git commit SHA, and a pull request URL; with
several IDs, the same summary, commit and PR apply to each.

Best practice: close after "git push" succeeds, and always include
--summary with context for future readers.

When the last open child of a parent closes, the server closes the
parent automatically, without a summary; close reports this so you can
add one. Closing an issue that is already closed replaces its summary,
commit and PR.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pr, _ := cmd.Flags().GetString("pr")
			commit, _ := cmd.Flags().GetString("commit")
			summary, _ := cmd.Flags().GetString("summary")

			var cr *api.CompletionResult
			if pr != "" || commit != "" || summary != "" {
				cr = &api.CompletionResult{Summary: summary, PRURL: pr, CommitSHA: commit}
			}

			// Full UUIDs need no project; prefixes/local IDs resolve in the
			// current project (fetched once, only when one is present).
			allUUIDs := true
			for _, arg := range args {
				if !isFullUUID(strings.TrimSpace(arg)) {
					allUUIDs = false
				}
			}

			var pid string
			var beads []api.Bead
			if allUUIDs {
				if err := d.requireAuth(); err != nil {
					return err
				}
			} else {
				var err error
				pid, err = d.requireProject(cmd)
				if err != nil {
					return err
				}
				beads, err = api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
				if err != nil {
					return err
				}
			}

			var uuids []string
			for _, arg := range args {
				uuid, err := resolve.ID(arg, beads)
				if err != nil {
					return fmt.Errorf("failed to resolve %q: %w", arg, err)
				}
				uuids = append(uuids, uuid)
			}

			// Issues outside the current project (or with no project at all)
			// are looked up by UUID so close can report their prior state.
			beads = withForeignBeads(d, beads, uuids)

			// A UUID that is not in the current project's list lives elsewhere:
			// send it without a project so the server uses the bead's own.
			inProject := make(map[string]bool, len(beads))
			for _, b := range beads {
				if b.ProjectID == "" || b.ProjectID == pid {
					inProject[b.ID] = true
				}
			}
			closeOne := func(uuid string) error {
				bpid := pid
				if pid == "" || !inProject[uuid] {
					bpid = ""
				}
				return d.withBeadProject(uuid, bpid, func(p string) error {
					req := api.CloseBeadRequest{ProjectID: p, Status: "completed", CompletionResult: cr}
					_, err := d.Client.Patch("/beads/"+uuid, req)
					return err
				})
			}

			mixed := false
			for _, u := range uuids {
				if !inProject[u] {
					mixed = true
				}
			}

			if len(uuids) == 1 || mixed {
				for _, uuid := range uuids {
					if err := closeOne(uuid); err != nil {
						return err
					}
				}
				printCloseResults(d, pid, uuids, beads, cr != nil)
				return nil
			}

			items := make([]api.BulkCloseItem, len(uuids))
			for i, uuid := range uuids {
				items[i] = api.BulkCloseItem{ID: uuid, Summary: summary, CommitSHA: commit, PRURL: pr}
			}

			_, err := d.Client.Post("/beads/bulk/close", api.BulkCloseRequest{ProjectID: pid, Beads: items})
			if needsProjectID(err) && pid == "" {
				// Older server: close one at a time, resolving each bead's project.
				for _, uuid := range uuids {
					if err = closeOne(uuid); err != nil {
						return err
					}
				}
			} else if err != nil {
				return err
			}

			printCloseResults(d, pid, uuids, beads, cr != nil)
			return nil
		},
	}
	cmd.Flags().String("pr", "", "Pull request URL")
	cmd.Flags().String("commit", "", "Git commit SHA")
	cmd.Flags().String("summary", "", "Completion summary")
	return cmd
}

// withForeignBeads appends a detail fetch (GET /beads/:id) for each UUID that
// is not in beads, so printCloseResults knows its status and parent. Fetch
// failures are ignored: the close itself reports any real error.
func withForeignBeads(d *Deps, beads []api.Bead, uuids []string) []api.Bead {
	known := make(map[string]bool, len(beads))
	for _, b := range beads {
		known[b.ID] = true
	}
	for _, u := range uuids {
		if known[u] {
			continue
		}
		known[u] = true
		if b, err := api.JSON[api.Bead](d.Client.Get("/beads/" + u)); err == nil {
			if b.ProjectID == "" && b.Project != nil {
				b.ProjectID = b.Project.ID
			}
			beads = append(beads, b)
		}
	}
	return beads
}

// printCloseResults reports each close, flags issues that were already
// closed, and points out parents the server auto-closed without a summary.
func printCloseResults(d *Deps, pid string, uuids []string, before []api.Bead, withDetails bool) {
	byID := make(map[string]api.Bead, len(before))
	for _, b := range before {
		byID[b.ID] = b
	}

	parents := []string{}
	seen := map[string]bool{}
	for _, uuid := range uuids {
		b := byID[uuid]
		if b.Status == "completed" {
			if withDetails {
				fmt.Printf("Updated: %s (already closed; its summary, commit and PR were replaced)\n", uuid)
			} else {
				fmt.Printf("Already closed: %s\n", uuid)
			}
		} else {
			fmt.Printf("Closed: %s\n", uuid)
		}
		if p := b.ParentBeadID; p != "" && !seen[p] && byID[p].Status != "completed" {
			seen[p] = true
			parents = append(parents, p)
		}
	}

	for _, p := range parents {
		parent, err := api.JSON[api.Bead](d.Client.Get("/beads/" + p + projectQuery(pid)))
		if err != nil || parent.Status != "completed" {
			continue
		}
		fmt.Printf("Parent %s %q closed automatically: all its children are done.\n", shortID(p), parent.Subject)
		fmt.Printf("  Add an overall summary: devdash close %s --summary=\"...\"\n", shortID(p))
	}
}

// warnIfParentCompleted tells the user when new or restarted work sits under
// a parent that is already completed; the server leaves that parent closed.
func warnIfParentCompleted(d *Deps, pid, parentID string) {
	if parentID == "" {
		return
	}
	parent, err := api.JSON[api.Bead](d.Client.Get("/beads/" + parentID + projectQuery(pid)))
	if err != nil || parent.Status != "completed" {
		return
	}
	fmt.Fprintf(os.Stderr, "Note: parent %s %q is already completed and stays completed.\n", shortID(parentID), parent.Subject)
	fmt.Fprintf(os.Stderr, "  To reopen it: devdash update %s --status=in_progress\n", shortID(parentID))
}
