package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

type analyzeResult struct {
	EstimatedComplexity string          `json:"estimatedComplexity"`
	AffectedFiles       json.RawMessage `json:"affectedFiles"`
	AffectedModules     json.RawMessage `json:"affectedModules"`
	ShouldSubdivide     bool            `json:"shouldSubdivide"`
	Reasoning           string          `json:"reasoning"`
	AgentInstructions   string          `json:"agentInstructions"`
	Subtasks            []struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	} `json:"subtasks"`
}

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
			json.Unmarshal(beadData, &bead)
			fmt.Printf("Analyzing: %s\n\n", bead.Subject)

			data, err := d.Client.Post("/jobs/analyze", map[string]string{"beadId": uuid, "projectId": pid})
			if err != nil {
				return fmt.Errorf("failed to start analysis: %w", err)
			}

			var job api.Job
			json.Unmarshal(data, &job)

			timeout := time.After(300 * time.Second)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					return fmt.Errorf("analysis timed out after 5 minutes")
				case <-ticker.C:
					statusData, err := d.Client.Get("/jobs/" + job.ID)
					if err != nil {
						continue
					}
					var current api.Job
					json.Unmarshal(statusData, &current)

					switch current.Status {
					case "completed":
						return printAnalysis(current, d.Client, pid)
					case "failed":
						msg := current.Error
						if current.FailureAnalysis != nil {
							msg = current.FailureAnalysis.Summary
						}
						return fmt.Errorf("analysis failed: %s", msg)
					default:
						fmt.Fprintf(os.Stderr, "Status: %s...\n", current.Status)
					}
				}
			}
		},
	}
}

func printAnalysis(job api.Job, client *api.Client, pid string) error {
	if job.Result == nil {
		fmt.Println("Analysis complete (no result data)")
		return nil
	}

	resultBytes, _ := json.Marshal(job.Result)
	var result analyzeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		out, _ := json.MarshalIndent(job.Result, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("Complexity:  %s\n", result.EstimatedComplexity)
	fmt.Printf("Files:       %s\n", string(result.AffectedFiles))
	fmt.Printf("Modules:     %s\n", string(result.AffectedModules))
	fmt.Printf("Subdivide:   %v\n", result.ShouldSubdivide)

	if result.Reasoning != "" {
		fmt.Printf("\n## Reasoning\n%s\n", result.Reasoning)
	}
	if result.AgentInstructions != "" {
		fmt.Printf("\n## Agent Instructions\n%s\n", result.AgentInstructions)
	}

	if len(result.Subtasks) > 0 {
		fmt.Printf("\n## Decomposed into %d subtasks:\n", len(result.Subtasks))
		for _, st := range result.Subtasks {
			fmt.Printf("  - %s %s\n", shortID(st.ID), st.Subject)
		}
	}

	return nil
}
