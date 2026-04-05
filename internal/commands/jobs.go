package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/output"
	"github.com/spf13/cobra"
)

func newJobsCmd(d *Deps) *cobra.Command {
	jobsCmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent jobs",
		Long: `List recent jobs for the current project, optionally filtered by bead.

By default, prints one summary line per job showing its ID, status, bead,
and creation time. Pass --bead to narrow results to a single issue.

Subcommands provide deeper inspection: "jobs show" dumps full JSON detail
for a single job, "jobs log" streams the output log (with optional --tail),
and "jobs failures" lists the last 10 failed jobs with optional --bead filter.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			path := "/jobs?projectId=" + pid
			if beadID, _ := cmd.Flags().GetString("bead"); beadID != "" {
				path += "&beadId=" + beadID
			}

			data, err := d.Client.Get(path)
			if err != nil {
				return err
			}

			jobs, err := api.JSON[[]api.Job](data, nil)
			if err != nil {
				return err
			}

			for _, j := range jobs {
				fmt.Println(output.FormatJobLine(j))
			}
			return nil
		},
	}
	jobsCmd.Flags().String("bead", "", "Filter by bead ID")

	jobsCmd.AddCommand(&cobra.Command{
		Use:   "show <id>",
		Short: "Job details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			data, err := d.Client.Get("/jobs/" + args[0])
			if err != nil {
				return err
			}
			var raw json.RawMessage
			_ = json.Unmarshal(data, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	})

	logCmd := &cobra.Command{
		Use:   "log <id>",
		Short: "Job output log",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			tail, _ := cmd.Flags().GetInt("tail")

			data, err := d.Client.Get("/jobs/" + args[0])
			if err != nil {
				return err
			}

			var job api.Job
			if err := json.Unmarshal(data, &job); err != nil {
				return err
			}

			log := job.OutputLog
			if tail > 0 && log != "" {
				lines := strings.Split(log, "\n")
				if len(lines) > tail {
					lines = lines[len(lines)-tail:]
				}
				log = strings.Join(lines, "\n")
			}

			fmt.Println(log)
			return nil
		},
	}
	logCmd.Flags().Int("tail", 0, "Last N lines")
	jobsCmd.AddCommand(logCmd)

	failuresCmd := &cobra.Command{
		Use:   "failures",
		Short: "Recent failed jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			path := "/jobs?projectId=" + pid
			if beadID, _ := cmd.Flags().GetString("bead"); beadID != "" {
				path += "&beadId=" + beadID
			}

			data, err := d.Client.Get(path)
			if err != nil {
				return err
			}

			jobs, err := api.JSON[[]api.Job](data, nil)
			if err != nil {
				return err
			}

			count := 0
			for _, j := range jobs {
				if j.Status != "failed" {
					continue
				}
				fmt.Println(output.FormatJobFailureLine(j))
				count++
				if count >= 10 {
					break
				}
			}
			return nil
		},
	}
	failuresCmd.Flags().String("bead", "", "Filter by bead ID")
	jobsCmd.AddCommand(failuresCmd)

	return jobsCmd
}
