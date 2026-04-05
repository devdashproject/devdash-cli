package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type reconcileFinding struct {
	Type          string `json:"type"`
	Severity      string `json:"severity"`
	Subject       string `json:"subject"`
	Reason        string `json:"reason"`
	RelatedBeadID string `json:"relatedBeadId,omitempty"`
}

func newReconcileCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reconcile-tasks",
		Short: "Audit and fix backlog inconsistencies",
		Long: `Audit the project backlog for inconsistencies such as orphaned
dependencies, status mismatches, and unreachable beads.

By default the command runs in --dry-run mode, listing what it found
without changing anything. Pass --auto-fix to apply corrections.
Use --json to get the raw findings as JSON.

Findings are grouped by type and include severity and related bead IDs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			autoFix, _ := cmd.Flags().GetBool("auto-fix")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			data, err := d.Client.Post("/jobs/reconcile-tasks", map[string]interface{}{
				"projectId": pid, "autoFix": autoFix,
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				var raw json.RawMessage
				_ = json.Unmarshal(data, &raw)
				out, _ := json.MarshalIndent(raw, "", "  ")
				fmt.Println(string(out))
				return nil
			}

			var result struct {
				Findings []reconcileFinding `json:"findings"`
				Fixed    int                `json:"fixed"`
			}
			if json.Unmarshal(data, &result) != nil {
				fmt.Println(string(data))
				return nil
			}

			fmt.Println("Auditing backlog...")
			fmt.Println()

			typeCounts := make(map[string]int)
			for _, f := range result.Findings {
				typeCounts[f.Type]++
			}
			for t, c := range typeCounts {
				fmt.Printf("  %s: %d\n", t, c)
			}
			fmt.Println()

			for _, f := range result.Findings {
				related := ""
				if f.RelatedBeadID != "" {
					related = fmt.Sprintf(" [related: %s]", shortID(f.RelatedBeadID))
				}
				fmt.Printf("[%s] %s\n  → %s%s\n\n", f.Severity, f.Subject, f.Reason, related)
			}

			if !autoFix && len(result.Findings) > 0 {
				fmt.Println("Run with --auto-fix to fix dependency issues automatically")
			}
			if autoFix {
				fmt.Printf("Fixed %d issues.\n", result.Fixed)
			}
			return nil
		},
	}
	cmd.Flags().Bool("dry-run", true, "Preview findings only (default)")
	cmd.Flags().Bool("auto-fix", false, "Apply fixes automatically")
	cmd.Flags().Bool("json", false, "Output raw JSON")
	return cmd
}
