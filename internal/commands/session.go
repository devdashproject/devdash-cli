package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newSessionCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Save and list session summaries (project memory)",
		Long: `Save and list session summaries for the current project.

Session summaries are short handoff notes — what was done, what's left,
what to know next — stored in project memory. They show up as "Recent
Sessions" in the DevDash Memory tab and are returned to agents in project
context, so the next session (human or agent) can pick up where this one
stopped.`,
	}
	cmd.AddCommand(newSessionSaveCmd(d), newSessionListCmd(d))
	return cmd
}

func newSessionSaveCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save a session summary to project memory",
		Long: `Save a session summary to project memory.

Requires --summary. Use at the end of a session to record context for the
next one. Requires the manage_memory permission on the project.

Example:
  devdash session save --summary="Finished pagination; remaining: cursor tests in api/list_test.go"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}
			summary, _ := cmd.Flags().GetString("summary")
			if summary == "" {
				return fmt.Errorf("--summary is required")
			}
			data, err := d.Client.Post("/memory/sessions/"+pid, map[string]interface{}{"summary": summary})
			if err != nil {
				return err
			}
			var res struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(data, &res) == nil && res.ID != "" {
				fmt.Printf("Saved session: %s\n", res.ID)
			} else {
				fmt.Println("Saved session")
			}
			return nil
		},
	}
	cmd.Flags().String("summary", "", "Session summary (required)")
	return cmd
}

func newSessionListCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent session summaries",
		Long: `List recent session summaries for the current project, newest first,
as JSON. Use --limit to control how many are returned (default 10).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			data, err := d.Client.Get(fmt.Sprintf("/memory/sessions/%s?limit=%d", pid, limit))
			if err != nil {
				return err
			}
			var sessions []json.RawMessage
			if err := json.Unmarshal(data, &sessions); err != nil {
				fmt.Println(string(data))
				return nil
			}
			out, _ := json.MarshalIndent(sessions, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
	cmd.Flags().Int("limit", 10, "Maximum sessions to return (1-100)")
	return cmd
}
