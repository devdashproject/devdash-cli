package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/output"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newDiagnoseCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "diagnose <id>",
		Short: "Investigate bead: status, job history, failure details",
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

			beadData, err := d.Client.Get("/beads/" + uuid + "?projectId=" + pid)
			if err != nil {
				return fmt.Errorf("failed to fetch bead: %w", err)
			}

			var bead api.Bead
			json.Unmarshal(beadData, &bead)

			fmt.Println("── Bead ──")
			fmt.Printf("%s  %s  [%s] [P%d] [%s]\n",
				shortID(bead.ID), bead.Subject, bead.Status, bead.Priority, bead.BeadType)

			jobsData, err := d.Client.Get("/jobs?projectId=" + pid)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not fetch jobs: %v\n", err)
				return nil
			}

			var jobs []api.Job
			json.Unmarshal(jobsData, &jobs)

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
