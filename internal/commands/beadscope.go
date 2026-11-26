package commands

import (
	"errors"
	"net/url"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

// isFullUUID reports whether s has the shape of a full 36-character UUID.
func isFullUUID(s string) bool {
	return len(s) == 36 && strings.Count(s, "-") == 4
}

// projectQuery returns "?projectId=<pid>", or "" when pid is empty (the
// server then uses the bead's own project).
func projectQuery(pid string) string {
	if pid == "" {
		return ""
	}
	return "?projectId=" + url.QueryEscape(pid)
}

// resolveBead turns a user-supplied issue ID into a full UUID plus the project
// to send with the request.
//
// A full UUID needs no project: it is returned with an empty pid, so the
// request carries no projectId and the server checks permission against the
// bead's own project. This works from any directory, with no .devdash or
// DD_PROJECT_ID. A prefix or local ID is only unique within one project, so
// it still requires the current project and is resolved against it.
func (d *Deps) resolveBead(cmd *cobra.Command, arg string) (uuid, pid string, err error) {
	arg = strings.TrimSpace(arg)
	if isFullUUID(arg) {
		if err := d.requireAuth(); err != nil {
			return "", "", err
		}
		return arg, "", nil
	}
	pid, err = d.requireProject(cmd)
	if err != nil {
		return "", "", err
	}
	uuid, err = resolve.IDWithFetch(arg, d.Client, pid)
	if err != nil {
		return "", "", err
	}
	return uuid, pid, nil
}

// needsProjectID reports whether err looks like an older server rejecting a
// request because projectId was omitted (validation error naming projectId).
func needsProjectID(err error) bool {
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message+" "+apiErr.Body), "projectid")
}

// lookupProjectID asks the server which project a bead lives in
// (GET /beads/:id without projectId, which works on old and new servers).
func (d *Deps) lookupProjectID(uuid string) (string, error) {
	bead, err := api.JSON[api.Bead](d.Client.Get("/beads/" + uuid))
	if err != nil {
		return "", err
	}
	pid := bead.ProjectID
	if pid == "" && bead.Project != nil {
		pid = bead.Project.ID
	}
	return pid, nil
}

// withBeadProject runs fn(pid). When pid is empty and the server rejects the
// missing projectId (an older server), it resolves the bead's project with one
// GET and retries once with that project. Newer servers never hit the retry.
func (d *Deps) withBeadProject(uuid, pid string, fn func(pid string) error) error {
	err := fn(pid)
	if pid != "" || !needsProjectID(err) {
		return err
	}
	resolved, lerr := d.lookupProjectID(uuid)
	if lerr != nil || resolved == "" {
		return err
	}
	return fn(resolved)
}
