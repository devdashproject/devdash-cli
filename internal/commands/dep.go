package commands

import (
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

func newDepCmd(d *Deps) *cobra.Command {
	depCmd := &cobra.Command{
		Use:   "dep",
		Short: "Manage dependencies between issues",
	}

	depCmd.AddCommand(&cobra.Command{
		Use:   "add <issue> <depends-on>",
		Short: "Add a dependency (issue is blocked until depends-on completes)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			issueUUID, err := resolve.ID(args[0], beads)
			if err != nil {
				return fmt.Errorf("failed to resolve issue %q: %w", args[0], err)
			}

			depUUID, err := resolve.ID(args[1], beads)
			if err != nil {
				return fmt.Errorf("failed to resolve dependency %q: %w", args[1], err)
			}

			_, err = d.Client.Post("/beads/"+issueUUID+"/dependencies", api.AddDependencyRequest{ProjectID: pid, BlockedBy: depUUID})
			if err != nil {
				return err
			}

			fmt.Printf("Added dependency: %s depends on %s\n", shortID(issueUUID), shortID(depUUID))
			return nil
		},
	})

	depCmd.AddCommand(&cobra.Command{
		Use:   "remove <issue> <depends-on>",
		Short: "Remove a dependency",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			issueUUID, err := resolve.ID(args[0], beads)
			if err != nil {
				return fmt.Errorf("failed to resolve issue %q: %w", args[0], err)
			}

			depUUID, err := resolve.ID(args[1], beads)
			if err != nil {
				return fmt.Errorf("failed to resolve dependency %q: %w", args[1], err)
			}

			_, err = d.Client.Delete("/beads/" + issueUUID + "/dependencies/" + depUUID + "?projectId=" + pid)
			if err != nil {
				return err
			}

			fmt.Printf("Removed dependency: %s no longer depends on %s\n", shortID(issueUUID), shortID(depUUID))
			return nil
		},
	})

	return depCmd
}
