package commands

import (
	"encoding/json"
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newActivityCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity [<id>]",
		Short: "View activity log",
		Long: `View the activity log for the current project or a specific issue.

Without arguments, shows all recent activity across the project. When an
issue ID is provided, filters to activity related to that issue only.
Use --limit to cap the number of results returned.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			path := "/projects/" + pid + "/activity"
			sep := "?"
			if len(args) > 0 {
				uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
				if err != nil {
					return err
				}
				path += sep + "targetId=" + uuid
				sep = "&"
			}

			if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
				path += fmt.Sprintf("%slimit=%d", sep, limit)
			}

			data, err := d.Client.Get(path)
			if err != nil {
				return err
			}

			var activity json.RawMessage
			if err := json.Unmarshal(data, &activity); err != nil {
				fmt.Println(string(data))
				return nil
			}
			out, _ := json.MarshalIndent(activity, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
	cmd.Flags().Int("limit", 0, "Maximum number of results")
	return cmd
}
