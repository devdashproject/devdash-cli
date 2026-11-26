package commands

import (
	"fmt"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

func newDepCmd(d *Deps) *cobra.Command {
	depCmd := &cobra.Command{
		Use:   "dep",
		Short: "Manage dependencies between issues",
		Long: `Manage dependencies between issues.

Use "dep add" to declare that one issue is blocked until another
completes, and "dep remove" to clear that relationship. Blocked
issues appear in "devdash blocked" and are excluded from
"devdash ready".`,
	}

	depCmd.AddCommand(&cobra.Command{
		Use:   "add <issue> <depends-on>",
		Short: "Add a dependency (issue is blocked until depends-on completes)",
		Long: `Declare that <issue> is blocked until <depends-on> completes.

Either ID may be a full UUID, which needs no project configured and may live
in a different project than the other issue (cross-project dependency; you need
access to both projects, and cycles across projects are rejected). Prefixes and
local IDs are resolved in the current project.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueUUID, depUUID, pid, err := d.resolveDepPair(cmd, args)
			if err != nil {
				return err
			}

			err = d.withBeadProject(issueUUID, pid, func(p string) error {
				_, err := d.Client.Post("/beads/"+issueUUID+"/dependencies", api.AddDependencyRequest{ProjectID: p, BlockedBy: depUUID})
				return err
			})
			if err != nil {
				return err
			}

			fmt.Printf("Added dependency: %s depends on %s%s\n", shortID(issueUUID), shortID(depUUID), d.foreignBlockerNote(issueUUID, depUUID))
			return nil
		},
	})

	depCmd.AddCommand(&cobra.Command{
		Use:   "remove <issue> <depends-on>",
		Short: "Remove a dependency",
		Long: `Remove a dependency. Either ID may be a full UUID (no project needed, may be
in another project); prefixes and local IDs are resolved in the current project.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueUUID, depUUID, pid, err := d.resolveDepPair(cmd, args)
			if err != nil {
				return err
			}

			err = d.withBeadProject(issueUUID, pid, func(p string) error {
				_, err := d.Client.Delete("/beads/" + issueUUID + "/dependencies/" + depUUID + projectQuery(p))
				return err
			})
			if err != nil {
				return err
			}

			fmt.Printf("Removed dependency: %s no longer depends on %s\n", shortID(issueUUID), shortID(depUUID))
			return nil
		},
	})

	return depCmd
}

// resolveDepPair resolves both dep arguments. Full UUIDs need no project. If
// either is a prefix, the current project is required to resolve it and is
// sent with the request, unless the issue itself is a full UUID (then the
// server uses the issue's own project, which also covers a foreign issue).
func (d *Deps) resolveDepPair(cmd *cobra.Command, args []string) (issueUUID, depUUID, pid string, err error) {
	var beads []api.Bead
	if !isFullUUID(strings.TrimSpace(args[0])) || !isFullUUID(strings.TrimSpace(args[1])) {
		pid, err = d.requireProject(cmd)
		if err != nil {
			return "", "", "", err
		}
		beads, err = api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
		if err != nil {
			return "", "", "", err
		}
	} else if err = d.requireAuth(); err != nil {
		return "", "", "", err
	}

	issueUUID, err = resolve.ID(args[0], beads)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to resolve issue %q: %w", args[0], err)
	}
	depUUID, err = resolve.ID(args[1], beads)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to resolve dependency %q: %w", args[1], err)
	}
	if isFullUUID(strings.TrimSpace(args[0])) {
		pid = ""
	}
	return issueUUID, depUUID, pid, nil
}

// foreignBlockerNote returns " (<blocker> is in project <name>)" when the
// blocker lives in a different project than the issue, else "". It is
// best-effort display only: any lookup failure (or an older server that omits
// project info) yields "".
func (d *Deps) foreignBlockerNote(issueUUID, depUUID string) string {
	projectOf := func(uuid string) (id, name string) {
		b, err := api.JSON[api.Bead](d.Client.Get("/beads/" + uuid))
		if err != nil {
			return "", ""
		}
		id, name = b.ProjectID, b.ProjectName
		if b.Project != nil {
			id, name = b.Project.ID, b.Project.Name
		}
		return id, name
	}
	issueProject, _ := projectOf(issueUUID)
	blockerProject, blockerName := projectOf(depUUID)
	if issueProject == "" || blockerProject == "" || issueProject == blockerProject {
		return ""
	}
	if blockerName == "" {
		blockerName = shortID(blockerProject)
	}
	return fmt.Sprintf(" (blocker is in project %s)", blockerName)
}
