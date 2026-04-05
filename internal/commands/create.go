package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newCreateCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long: `Create a new issue in the current project.

Requires --title. Optionally set the type (task, bug, feature,
enhancement, thought), priority (0=critical through 4=backlog),
description, parent issue, due date, and time estimate.

The new issue is created in "pending" status. Use "devdash update"
to mark it in_progress when you begin work.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			title, _ := cmd.Flags().GetString("title")
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if strings.HasPrefix(title, "-") {
				return fmt.Errorf("title cannot start with '-': %s\nUse --title=\"...\" for titles that might look like flags", title)
			}

			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			description, _ := cmd.Flags().GetString("description")
			beadType, _ := cmd.Flags().GetString("type")
			priority, _ := cmd.Flags().GetInt("priority")
			parent, _ := cmd.Flags().GetString("parent")
			due, _ := cmd.Flags().GetString("due")
			estimate, _ := cmd.Flags().GetInt("estimate")

			req := api.CreateBeadRequest{
				ProjectID:   pid,
				Subject:     title,
				Description: description,
				BeadType:    beadType,
				Priority:    &priority,
			}

			if parent != "" {
				parentUUID, err := resolve.IDWithFetch(parent, d.Client, pid)
				if err != nil {
					return fmt.Errorf("failed to resolve parent ID: %w", err)
				}
				req.ParentBeadID = parentUUID
			}

			if due != "" {
				req.DueDate = due
			}

			if cmd.Flags().Changed("estimate") {
				req.EstimatedMinutes = &estimate
			}

			data, err := d.Client.Post("/beads", req)
			if err != nil {
				return err
			}

			var bead api.Bead
			if err := json.Unmarshal(data, &bead); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Created: %s - %s\n", bead.ID, bead.Subject)
			return nil
		},
	}
	cmd.Flags().String("title", "", "Issue title (required)")
	cmd.Flags().String("description", "", "Issue description")
	cmd.Flags().String("type", "task", "Issue type: task, bug, feature, enhancement, thought")
	cmd.Flags().Int("priority", 2, "Priority: 0=critical, 1=high, 2=medium, 3=low, 4=backlog")
	cmd.Flags().String("parent", "", "Parent bead ID")
	cmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().Int("estimate", 0, "Estimated minutes")
	return cmd
}
