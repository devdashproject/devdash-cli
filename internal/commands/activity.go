package commands

import (
	"net/url"
	"strconv"

	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newActivityCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity [<id>]",
		Short: "View activity log",
		Long: `View the activity log for the current project or a specific issue.

Without arguments, shows recent activity across the project. When an
issue ID is provided, shows only activity for that issue, across its whole
history. Use --limit to set the page size (server default 50, max 500);
when there is more, the response has "hasMore": true and a "nextCursor"
to pass back with --cursor. JSON by default; --pretty prints one line per
event (no emails or avatar URLs).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			pretty, _ := cmd.Flags().GetBool("pretty")

			q := url.Values{}
			if limit > 0 {
				q.Set("limit", strconv.Itoa(limit))
			}
			if cursor != "" {
				q.Set("cursor", cursor)
			}
			if len(args) > 0 {
				uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
				if err != nil {
					return err
				}
				// The server filters by targetId (and paginates the filtered feed)
				q.Set("targetId", uuid)
			}

			path := "/projects/" + pid + "/activity"
			if enc := q.Encode(); enc != "" {
				path += "?" + enc
			}

			data, err := d.Client.Get(path)
			if err != nil {
				return err
			}

			printOutput(data, pretty, prettyActivity)
			return nil
		},
	}
	cmd.Flags().Int("limit", 0, "Maximum number of results (server default 50, max 500)")
	cmd.Flags().String("cursor", "", "Fetch the next page: pass nextCursor from the previous response")
	cmd.Flags().Bool("pretty", false, "One line per event instead of JSON")
	return cmd
}
