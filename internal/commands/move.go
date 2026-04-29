package commands

import (
	"errors"
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newMoveCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move an issue to a different project",
		Long: `Move an issue from one project to another.

The issue's full history, comments, and metadata are preserved by the
server during the move.

Source project: determined by --from, then --project, then DD_PROJECT_ID,
then the .devdash file. Use --from explicitly when the issue lives in a
different project than the current directory.

Both --from and --to accept full UUIDs or unambiguous short prefixes.
Run 'devdash project list' to find project IDs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve source project: --from takes priority over global --project/env/.devdash
			var sourcePID string
			var err error
			if fromVal, _ := cmd.Flags().GetString("from"); fromVal != "" {
				sourcePID, err = resolve.ProjectID(fromVal, d.Client)
				if err != nil {
					return fmt.Errorf("source project: %w", err)
				}
			} else {
				sourcePID, err = d.requireProject(cmd)
				if err != nil {
					return err
				}
			}

			rawTo, _ := cmd.Flags().GetString("to")
			if rawTo == "" {
				return fmt.Errorf("--to is required: specify the target project ID or prefix")
			}
			targetPID, err := resolve.ProjectID(rawTo, d.Client)
			if err != nil {
				return err
			}

			if sourcePID == targetPID {
				return fmt.Errorf("source and target project are the same (%s); nothing to do", sourcePID)
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, sourcePID)
			if err != nil {
				return fmt.Errorf("%w\n\nHint: the issue may belong to a different project.\n"+
					"Set the source with --from=<project-id> or DD_PROJECT_ID=<project-id>.\n"+
					"Run 'devdash project list' to find project IDs.", err)
			}

			req := api.MoveBeadRequest{
				ProjectID:       sourcePID,
				TargetProjectID: targetPID,
			}
			_, err = d.Client.Post("/beads/"+uuid+"/move", req)
			if err != nil {
				var apiErr *api.APIError
				if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
					return fmt.Errorf("target project %s not found — verify with 'devdash project list'", shortID(targetPID))
				}
				return err
			}

			fmt.Printf("Moved: %s → project %s\n", uuid, targetPID)
			return nil
		},
	}
	cmd.Flags().String("to", "", "Target project ID (full UUID or short prefix)")
	cmd.Flags().String("from", "", "Source project ID (overrides --project / DD_PROJECT_ID)")
	return cmd
}
