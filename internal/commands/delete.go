package commands

import (
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newDeleteCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id> [<id>...]",
		Short: "Delete one or more issues",
		Long: `Permanently delete one or more issues.

Accepts one or multiple issue IDs. Use --cascade to also delete all
child issues. Use --force to skip the confirmation prompt.

This action is irreversible. If you want to preserve history, consider
closing the issue instead.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			cascade, _ := cmd.Flags().GetBool("cascade")

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			for _, arg := range args {
				uuid, err := resolve.ID(arg, beads)
				if err != nil {
					return fmt.Errorf("failed to resolve %q: %w", arg, err)
				}

				path := "/beads/" + uuid + "?projectId=" + pid
				if cascade {
					path += "&cascade=true"
				}

				_, err = d.Client.Delete(path)
				if err != nil {
					return fmt.Errorf("failed to delete %s: %w", uuid, err)
				}

				fmt.Printf("Deleted: %s\n", uuid)
			}
			return nil
		},
	}
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	cmd.Flags().Bool("cascade", false, "Delete children too")
	return cmd
}
