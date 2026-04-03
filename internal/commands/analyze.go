package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newAnalyzeCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "analyze <id>",
		Short: "Trigger sandbox analysis for an issue",
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

			beadData, _ := d.Client.Get("/beads/" + uuid + "?projectId=" + pid)
			var bead api.Bead
			_ = json.Unmarshal(beadData, &bead)
			fmt.Fprintf(os.Stderr, "Analyzing: %s\n\n", bead.Subject)

			data, err := d.Client.Post("/jobs/analyze", map[string]string{"beadId": uuid, "projectId": pid})
			if err != nil {
				return fmt.Errorf("failed to start analysis: %w", err)
			}

			var job api.Job
			if err := json.Unmarshal(data, &job); err != nil {
				return fmt.Errorf("failed to parse job response: %w", err)
			}

			short := shortID(job.ID)
			fmt.Fprintf(os.Stderr, "Analysis queued: %s\n", short)
			fmt.Fprintf(os.Stderr, "Check status: devdash jobs show %s\n", short)
			fmt.Println(job.ID)

			return nil
		},
	}
}
