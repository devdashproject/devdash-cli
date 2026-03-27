package commands

import (
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newUpdateCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
			if err != nil {
				return err
			}

			req := api.UpdateBeadRequest{}
			hasChanges := false

			if cmd.Flags().Changed("status") {
				v, _ := cmd.Flags().GetString("status")
				req.Status = &v
				hasChanges = true
			}
			if cmd.Flags().Changed("title") {
				v, _ := cmd.Flags().GetString("title")
				req.Subject = &v
				hasChanges = true
			}
			if cmd.Flags().Changed("description") {
				v, _ := cmd.Flags().GetString("description")
				req.Description = &v
				hasChanges = true
			}
			if cmd.Flags().Changed("priority") {
				v, _ := cmd.Flags().GetInt("priority")
				req.Priority = &v
				hasChanges = true
			}
			if cmd.Flags().Changed("owner") {
				v, _ := cmd.Flags().GetString("owner")
				req.AssignedTo = &v
				hasChanges = true
			}
			if cmd.Flags().Changed("parent") {
				v, _ := cmd.Flags().GetString("parent")
				parentUUID, err := resolve.IDWithFetch(v, d.Client, pid)
				if err != nil {
					return fmt.Errorf("failed to resolve parent ID: %w", err)
				}
				req.ParentBeadID = &parentUUID
				hasChanges = true
			}
			if cmd.Flags().Changed("pre-instructions") {
				v, _ := cmd.Flags().GetString("pre-instructions")
				req.PreInstructions = &v
				hasChanges = true
			}
			if cmd.Flags().Changed("due") {
				v, _ := cmd.Flags().GetString("due")
				req.DueDate = &v
				hasChanges = true
			}
			if cmd.Flags().Changed("estimate") {
				v, _ := cmd.Flags().GetInt("estimate")
				req.EstimatedMinutes = &v
				hasChanges = true
			}

			if !hasChanges {
				return fmt.Errorf("no changes specified")
			}

			_, err = d.Client.Patch("/beads/"+uuid, req)
			if err != nil {
				return err
			}

			fmt.Printf("Updated: %s\n", uuid)
			return nil
		},
	}
	cmd.Flags().String("status", "", "Status: pending, in_progress, completed")
	cmd.Flags().String("title", "", "New title")
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().Int("priority", -1, "Priority: 0-4")
	cmd.Flags().String("owner", "", "Assign to (email or name)")
	cmd.Flags().String("parent", "", "Parent bead ID")
	cmd.Flags().String("pre-instructions", "", "Agent-specific context")
	cmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().Int("estimate", 0, "Estimated minutes")
	return cmd
}
