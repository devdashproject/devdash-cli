package commands

import (
	"errors"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
)

// foreignBlocker is what we learn about a blocker that is not in the current
// project's issue list (it lives in another project).
type foreignBlocker struct {
	Status      string
	ProjectName string
	NoAccess    bool
}

// label is the short suffix shown after the blocker's ID.
func (f foreignBlocker) label() string {
	switch {
	case f.NoAccess:
		return "(no access)"
	case f.ProjectName != "":
		return "(" + f.ProjectName + ")"
	}
	return ""
}

// lookupForeignBlockers fetches each distinct blocker ID of the candidate
// beads that is missing from beads (so it lives in another project). Errors
// other than 403/404 leave the blocker out of the result; the caller then
// treats it as unfinished, as before.
func lookupForeignBlockers(d *Deps, beads []api.Bead, candidate func(api.Bead) bool) map[string]foreignBlocker {
	known := make(map[string]bool, len(beads))
	for _, b := range beads {
		known[b.ID] = true
	}
	out := map[string]foreignBlocker{}
	tried := map[string]bool{}
	for _, b := range beads {
		if !candidate(b) {
			continue
		}
		for _, id := range b.BlockedBy {
			if known[id] || tried[id] {
				continue
			}
			tried[id] = true
			detail, err := api.JSON[api.Bead](d.Client.Get("/beads/" + id))
			if err != nil {
				var apiErr *api.APIError
				if errors.As(err, &apiErr) && (apiErr.StatusCode == 403 || apiErr.StatusCode == 404) {
					out[id] = foreignBlocker{NoAccess: true}
				}
				continue
			}
			name := detail.ProjectName
			if detail.Project != nil && detail.Project.Name != "" {
				name = detail.Project.Name
			}
			out[id] = foreignBlocker{Status: detail.Status, ProjectName: name}
		}
	}
	return out
}

func foreignLabels(f map[string]foreignBlocker) map[string]string {
	labels := make(map[string]string, len(f))
	for id, fb := range f {
		if l := fb.label(); l != "" {
			labels[id] = l
		}
	}
	return labels
}

// markForeignCompleted records completed foreign blockers so they stop
// blocking. Blockers with no access or an unknown status keep blocking.
func markForeignCompleted(completed map[string]bool, f map[string]foreignBlocker) {
	for id, fb := range f {
		if strings.EqualFold(fb.Status, "completed") {
			completed[id] = true
		}
	}
}
