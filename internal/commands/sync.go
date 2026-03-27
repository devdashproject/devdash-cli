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
