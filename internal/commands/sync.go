package commands

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newSyncCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Trigger full GitHub reconciliation",
		Long: `Trigger a full GitHub reconciliation for the current project.

Syncs issue state, labels, and metadata between GitHub and devdash
so both systems reflect the same reality. The sync runs server-side
and returns a JSON summary of what changed.

Use this after making changes directly in GitHub (closing issues,
editing labels, updating milestones) to pull those changes back
into devdash.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}
			data, err := d.Client.Post("/sync/"+pid+"/sync-all", nil)
			if err != nil {
				return err
			}
			var raw json.RawMessage
			_ = json.Unmarshal(data, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
}

func newImportCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <issue-number> | --all",
		Short: "Import GitHub issues",
		Long: `Import GitHub issues into devdash as beads.

Pass a single issue number to import one issue, or use --all to bulk-import
every issue from the linked GitHub repository. With --all you can filter by
state (e.g. --state=open).

Returns the imported bead ID for single imports or a count of imported issues
for bulk imports. Requires a project with a linked GitHub repository.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			all, _ := cmd.Flags().GetBool("all")
			if all {
				state, _ := cmd.Flags().GetString("state")
				body := map[string]string{}
				if state != "" {
					body["state"] = state
				}
				data, err := d.Client.Post("/sync/"+pid+"/bulk-import", body)
				if err != nil {
					return err
				}
				var result struct{ Imported int }
				if json.Unmarshal(data, &result) == nil {
					fmt.Printf("Imported %d issue(s).\n", result.Imported)
				} else {
					fmt.Println(string(data))
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("provide an issue number or use --all")
			}
			num, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			data, err := d.Client.Post(fmt.Sprintf("/sync/%s/issues/%d/import", pid, num), nil)
			if err != nil {
				return err
			}
			var result struct {
				BeadID string `json:"beadId"`
			}
			if json.Unmarshal(data, &result) == nil && result.BeadID != "" {
				fmt.Printf("Imported as bead %s\n", result.BeadID)
			} else {
				fmt.Println(string(data))
			}
			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Import all issues")
	cmd.Flags().String("state", "open", "Issue state filter: open, all")
	return cmd
}
