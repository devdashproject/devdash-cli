package commands

import (
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newReportCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report <id>",
		Short: "Report progress on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			status, _ := cmd.Flags().GetString("status")
			if status == "" {
				return fmt.Errorf("--status is required (code_complete, committed, pushed, error)")
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
			if err != nil {
				return err
			}

			req := api.ReportRequest{Status: status}

			if v, _ := cmd.Flags().GetString("summary"); v != "" {
				req.Summary = v
			}
			if cmd.Flags().Changed("files-changed") {
				v, _ := cmd.Flags().GetInt("files-changed")
				req.FilesChanged = &v
			}
			if v, _ := cmd.Flags().GetString("branch"); v != "" {
				req.Branch = v
			}
			if v, _ := cmd.Flags().GetString("commit"); v != "" {
				req.Commit = v
			}
			if v, _ := cmd.Flags().GetString("error"); v != "" {
				req.Error = v
			}

			_, err = d.Client.Post("/beads/"+uuid+"/report", req)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: report failed: %v\n", err)
				return nil
			}

			fmt.Printf("Report submitted: %s for %s\n", status, uuid)
			return nil
		},
	}
	cmd.Flags().String("status", "", "Status: code_complete, committed, pushed, error (required)")
	cmd.Flags().String("summary", "", "Progress summary")
	cmd.Flags().Int("files-changed", 0, "Number of files changed")
	cmd.Flags().String("branch", "", "Git branch name")
	cmd.Flags().String("commit", "", "Git commit SHA")
	cmd.Flags().String("error", "", "Error message (when status=error)")
	return cmd
}
