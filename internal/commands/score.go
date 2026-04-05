package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

type scoreResult struct {
	BeadID            string `json:"beadId"`
	ComplexityScore   int    `json:"complexityScore"`
	AutomabilityScore int    `json:"automabilityScore"`
	AutomabilityGrade string `json:"automabilityGrade"`
}

func newScoreCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "score [<id>]",
		Short: "Score beads for automability",
		Long: `Score beads to evaluate how suitable they are for automation.

Without an ID, scores every bead in the current project and prints a
table with the automability grade, automability score, and complexity
for each. With an ID, scores a single bead and returns the full JSON
detail including the breakdown factors.

Useful for triaging a backlog to find the best candidates for
automated execution.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			if len(args) == 0 {
				data, err := d.Client.Post("/projects/"+pid+"/beads/score", nil)
				if err != nil {
					return err
				}
				var result struct{ Scored []scoreResult }
				if json.Unmarshal(data, &result) != nil {
					var scored []scoreResult
					if json.Unmarshal(data, &scored) == nil {
						result.Scored = scored
					} else {
						fmt.Println(string(data))
						return nil
					}
				}
				fmt.Printf("Scored %d beads:\n", len(result.Scored))
				for _, s := range result.Scored {
					fmt.Printf("  %s  %s (%d/100)  complexity=%d\n",
						shortID(s.BeadID), s.AutomabilityGrade, s.AutomabilityScore, s.ComplexityScore)
				}
				return nil
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
			if err != nil {
				return err
			}
			data, err := d.Client.Post("/projects/"+pid+"/beads/"+uuid+"/score", nil)
			if err != nil {
				return err
			}
			var raw json.RawMessage
			_ = json.Unmarshal(data, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
}
