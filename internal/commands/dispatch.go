package commands

import (
	"encoding/json"
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newDispatchCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispatch <id>",
		Short: "Dispatch a bead for execution",
		Long: `Queue a bead for execution as a job.

The prompt sent to the worker is chosen from the first non-empty field in this
order: PreInstructions, Description, Subject. Use --priority (0-4) to control
scheduling urgency and --worker to select an execution environment
(docker, e2b, or railway).

Returns the created job ID on success.`,
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

			beadData, _ := d.Client.Get("/beads/" + uuid + "?projectId=" + pid)
			var bead api.Bead
			_ = json.Unmarshal(beadData, &bead)

			prompt := bead.PreInstructions
			if prompt == "" {
				prompt = bead.Description
			}
			if prompt == "" {
				prompt = bead.Subject
			}

			body := map[string]interface{}{"beadId": uuid, "projectId": pid, "prompt": prompt}
			if cmd.Flags().Changed("priority") {
				p, _ := cmd.Flags().GetInt("priority")
				body["priority"] = p
			}
			if worker, _ := cmd.Flags().GetString("worker"); worker != "" {
				body["workerType"] = worker
			}

			data, err := d.Client.Post("/jobs", body)
			if err != nil {
				return err
			}

			var job api.Job
			_ = json.Unmarshal(data, &job)

			fmt.Printf("Job queued: %s\n", job.ID)
			fmt.Printf("  Bead:   %s — %s\n", shortID(uuid), bead.Subject)
			fmt.Printf("  Status: %s\n", job.Status)
			if job.WorkerType != "" {
				fmt.Printf("  Worker: %s\n", job.WorkerType)
			}
			return nil
		},
	}
	cmd.Flags().Int("priority", 2, "Job priority: 0-4")
	cmd.Flags().String("worker", "", "Execution backend: docker, e2b, railway")
	return cmd
}
