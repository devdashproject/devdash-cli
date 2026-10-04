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
issue ID is provided, filters to activity related to that issue only
(searching the project's most recent 500 events). Use --limit to cap the
number of results returned. JSON by default; --pretty prints one line per
event (no emails or avatar URLs).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			pretty, _ := cmd.Flags().GetBool("pretty")
			path := "/projects/" + pid + "/activity"

			if len(args) > 0 {
				uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
				if err != nil {
					return err
				}
				data, err := beadActivity(d, path, uuid, limit)
				if err != nil {
					return err
				}
				printOutput(data, pretty, prettyActivity)
				return nil
			}

			if limit > 0 {
				path += fmt.Sprintf("?limit=%d", limit)
			}

			data, err := d.Client.Get(path)
			if err != nil {
				return err
			}

			printOutput(data, pretty, prettyActivity)
			return nil
		},
	}
	cmd.Flags().Int("limit", 0, "Maximum number of results")
	cmd.Flags().Bool("pretty", false, "One line per event instead of JSON")
	return cmd
}

// activityWindow is the most events the server returns in one page.
const activityWindow = 500

// beadActivity returns {"data": [...]} activity for one bead. The server currently ignores
// the targetId filter (dev-dash 0dd43585), so fetch the largest window and
// filter here as well; events older than that window are not reachable.
func beadActivity(d *Deps, path, uuid string, limit int) ([]byte, error) {
	data, err := d.Client.Get(fmt.Sprintf("%s?limit=%d&targetId=%s", path, activityWindow, uuid))
	if err != nil {
		return nil, err
	}

	var page struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return data, nil
	}

	matched := []json.RawMessage{}
	for _, raw := range page.Data {
		var item struct {
			TargetID string `json:"targetId"`
		}
		if json.Unmarshal(raw, &item) == nil && item.TargetID == uuid {
			matched = append(matched, raw)
		}
	}
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	return json.Marshal(map[string]interface{}{"data": matched})
}
