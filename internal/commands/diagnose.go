package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/output"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newDiagnoseCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "diagnose <id>",
		Short: "Investigate bead: status, job history, failure details",
		Long: `Investigate a bead by showing its current state and associated job history.

Prints a summary line (ID, subject, status, priority, type), then lists all
jobs tied to that bead with their status and creation time. For the first
failed job found, it displays the error message, failure analysis (if any),
and the last 30 lines of the output log.

Useful as a single command to answer "what happened?" when a bead's jobs
aren't completing as expected.`,
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

			beadData, err := d.Client.Get("/beads/" + uuid + "?projectId=" + pid)
			if err != nil {
				return fmt.Errorf("failed to fetch bead: %w", err)
			}

			var bead api.Bead
			_ = json.Unmarshal(beadData, &bead)

			fmt.Println("── Bead ──")
			fmt.Printf("%s  %s  [%s] [P%d] [%s]\n",
				shortID(bead.ID), bead.Subject, bead.Status, bead.Priority, bead.BeadType)

			jobsData, err := d.Client.Get("/jobs?projectId=" + pid)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not fetch jobs: %v\n", err)
				return nil
			}

			var jobs []api.Job
			_ = json.Unmarshal(jobsData, &jobs)

			var beadJobs []api.Job
			for _, j := range jobs {
				if j.BeadID == uuid {
					beadJobs = append(beadJobs, j)
				}
			}

			fmt.Printf("\n── Jobs (%d) ──\n", len(beadJobs))
			for _, j := range beadJobs {
				fmt.Printf("%s %s  [%s]  %s\n",
					output.JobStatusIcon(j.Status), shortID(j.ID), j.Status, j.CreatedAt)
			}

			for _, j := range beadJobs {
				if j.Status != "failed" {
					continue
				}
				fmt.Printf("\n── Latest Failure: %s ──\n", shortID(j.ID))
				if j.Error != "" {
					fmt.Printf("Error: %s\n", j.Error)
				}
				if j.FailureAnalysis != nil && j.FailureAnalysis.Summary != "" {
					fmt.Printf("Analysis: %s\n", j.FailureAnalysis.Summary)
				}

				fullData, err := d.Client.Get("/jobs/" + j.ID)
				if err == nil {
					var fullJob api.Job
					if json.Unmarshal(fullData, &fullJob) == nil && fullJob.OutputLog != "" {
						lines := strings.Split(fullJob.OutputLog, "\n")
						if len(lines) > 30 {
							lines = lines[len(lines)-30:]
						}
						fmt.Println("\nLog (last 30 lines):")
						for _, l := range lines {
							fmt.Printf("  %s\n", l)
						}
					}
				}
				break
			}

			return nil
		},
	}
}
