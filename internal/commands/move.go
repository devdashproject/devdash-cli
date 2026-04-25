package commands

import (
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newMoveCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move an issue to a different project",
		Long: `Move an issue from the current project to a target project.

The issue's full history, comments, and metadata are preserved by the
server during the move. Use --to to specify the target project ID (full
UUID or unambiguous prefix accepted by the server).

The current project is determined by --project or the .devdash file,
as with all other commands.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcePID, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			targetPID, _ := cmd.Flags().GetString("to")
			if targetPID == "" {
				return fmt.Errorf("--to is required: specify the target project ID")
			}

			if sourcePID == targetPID {
				return fmt.Errorf("source and target project are the same (%s); nothing to do", sourcePID)
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, sourcePID)
			if err != nil {
				return err
			}

			req := api.MoveBeadRequest{
				ProjectID:       sourcePID,
				TargetProjectID: targetPID,
			}
			_, err = d.Client.Post("/beads/"+uuid+"/move", req)
			if err != nil {
				return err
			}

			fmt.Printf("Moved: %s → project %s\n", uuid, targetPID)
			return nil
		},
	}
	cmd.Flags().String("to", "", "Target project ID (required; full UUID or server-accepted prefix)")
	return cmd
}
