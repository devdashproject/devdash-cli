package commands

import (
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newCommentCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment <id>",
		Short: "Add a comment to an issue",
		Long: `Add a comment to an issue.

Attaches a text comment to the specified issue. The --body flag is
required. Use this to record decisions, progress notes, or context
that doesn't belong in the issue title or description.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			body, _ := cmd.Flags().GetString("body")
			if body == "" {
				return fmt.Errorf("--body is required")
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
			if err != nil {
				return err
			}

			if _, err = d.Client.Post("/beads/"+uuid+"/comments", api.CommentRequest{ProjectID: pid, Content: body}); err != nil {
				return err
			}
			fmt.Printf("Commented on %s\n", uuid)
			return nil
		},
	}
	cmd.Flags().String("body", "", "Comment body (required)")
	return cmd
}

func newCommentsCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comments <id>",
		Short: "List comments on an issue",
		Long: `List all comments on an issue.

Fetches and displays every comment attached to the specified issue
in JSON format. Add --pretty for one line per comment. Use this to
review the discussion history and any decisions recorded on an issue.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
			if err != nil {
				return err
			}

			data, err := d.Client.Get("/beads/" + uuid + "/comments?projectId=" + pid)
			if err != nil {
				return err
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			printOutput(data, pretty, prettyComments)
			return nil
		},
	}
	cmd.Flags().Bool("pretty", false, "One line per comment instead of JSON")
	return cmd
}
