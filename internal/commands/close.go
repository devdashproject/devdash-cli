package commands

import (
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newCloseCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <id> [<id>...]",
		Short: "Close one or more issues",
		Long: `Close one or more issues, marking them as completed.

Accepts one or multiple issue IDs (short prefixes work). Optionally attach
a completion summary, the git commit SHA, and a pull request URL. When
closing a single issue, uses a direct PATCH; multiple IDs are sent as a
bulk close request.

Best practice: close after "git push" succeeds, and always include
--summary with context for future readers.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			pr, _ := cmd.Flags().GetString("pr")
			commit, _ := cmd.Flags().GetString("commit")
			summary, _ := cmd.Flags().GetString("summary")

			var cr *api.CompletionResult
			if pr != "" || commit != "" || summary != "" {
				cr = &api.CompletionResult{Summary: summary, PR: pr, Commit: commit}
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			var uuids []string
			for _, arg := range args {
				uuid, err := resolve.ID(arg, beads)
				if err != nil {
					return fmt.Errorf("failed to resolve %q: %w", arg, err)
				}
				uuids = append(uuids, uuid)
			}

			if len(uuids) == 1 {
				req := api.CloseBeadRequest{ProjectID: pid, Status: "completed", CompletionResult: cr}
				_, err := d.Client.Patch("/beads/"+uuids[0], req)
				if err != nil {
					return err
				}
				fmt.Printf("Closed: %s\n", uuids[0])
				return nil
			}

			items := make([]api.BulkCloseItem, len(uuids))
			for i, uuid := range uuids {
				items[i] = api.BulkCloseItem{ID: uuid, CompletionResult: cr}
			}

			_, err = d.Client.Post("/beads/bulk/close", api.BulkCloseRequest{ProjectID: pid, Beads: items})
			if err != nil {
				return err
			}

			for _, uuid := range uuids {
				fmt.Printf("Closed: %s\n", uuid)
			}
			return nil
		},
	}
	cmd.Flags().String("pr", "", "Pull request URL")
	cmd.Flags().String("commit", "", "Git commit SHA")
	cmd.Flags().String("summary", "", "Completion summary")
	return cmd
}
