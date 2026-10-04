package resolve

import (
	"fmt"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
)

// ID resolves a user-provided ID (full UUID, UUID prefix, or local ID) to a full UUID.
// It uses the provided beads cache to avoid extra API calls.
func ID(input string, beads []api.Bead) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty ID")
	}

	// Full UUID (36 chars with dashes)
	if len(input) == 36 && strings.Count(input, "-") == 4 {
		return input, nil
	}

	// Local ID (contains a dash but isn't a UUID prefix)
	if strings.Contains(input, "-") && !isHexString(strings.ReplaceAll(input, "-", "")) {
		return resolveLocalID(input, beads)
	}

	// UUID prefix
	return resolvePrefix(input, beads)
}

// IDWithFetch resolves an ID, fetching beads from the API if needed.
func IDWithFetch(input string, client *api.Client, projectID string) (string, error) {
	input = strings.TrimSpace(input)

	// Full UUID doesn't need resolution
	if len(input) == 36 && strings.Count(input, "-") == 4 {
		return input, nil
	}

	// Need beads for prefix/local resolution
	beads, err := api.FetchAll[api.Bead](client, "/beads?projectId="+projectID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch beads for ID resolution: %w", err)
	}

	return ID(input, beads)
}

func resolveLocalID(localID string, beads []api.Bead) (string, error) {
	lower := strings.ToLower(localID)
	var matches []api.Bead

	for _, b := range beads {
		if strings.ToLower(b.LocalBeadID) == lower {
			matches = append(matches, b)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no bead found with local ID %q", localID)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("ambiguous local ID %q matches %d beads", localID, len(matches))
	}
}

func resolvePrefix(prefix string, beads []api.Bead) (string, error) {
	lower := strings.ToLower(prefix)
	var matches []api.Bead

	for _, b := range beads {
		if strings.HasPrefix(strings.ToLower(b.ID), lower) {
			matches = append(matches, b)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no bead found with prefix %q", prefix)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("ambiguous prefix %q matches %d beads — use a longer prefix", prefix, len(matches))
	}
}

// ProjectID resolves a project ID prefix to a full UUID.
// Full UUIDs are returned as-is (no API call). Shorter inputs
// are resolved against GET /projects.
func ProjectID(input string, client *api.Client) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty project ID")
	}
	if len(input) == 36 && strings.Count(input, "-") == 4 {
		return input, nil
	}
	projects, err := api.FetchAll[api.Project](client, "/projects")
	if err != nil {
		return "", fmt.Errorf("failed to fetch projects for ID resolution: %w", err)
	}
	return resolveProjectPrefix(input, projects)
}

func resolveProjectPrefix(prefix string, projects []api.Project) (string, error) {
	lower := strings.ToLower(prefix)
	var matches []api.Project
	for _, p := range projects {
		if strings.HasPrefix(strings.ToLower(p.ID), lower) {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no project found with prefix %q — run 'devdash project list' to see available projects", prefix)
	case 1:
		return matches[0].ID, nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = fmt.Sprintf("%s (%s)", m.ID[:8], m.Name)
		}
		return "", fmt.Errorf("ambiguous project prefix %q matches %d projects: %s — use a longer prefix",
			prefix, len(matches), strings.Join(names, ", "))
	}
}

func isHexString(s string) bool {
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return len(s) > 0
}

// ProjectInList finds a project by full ID, exact name (case-insensitive),
// or unambiguous ID prefix, in that order.
func ProjectInList(input string, projects []api.Project) (api.Project, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return api.Project{}, fmt.Errorf("empty project")
	}
	for _, p := range projects {
		if strings.EqualFold(p.ID, input) {
			return p, nil
		}
	}
	var byName []api.Project
	for _, p := range projects {
		if strings.EqualFold(p.Name, input) {
			byName = append(byName, p)
		}
	}
	if len(byName) == 1 {
		return byName[0], nil
	}
	if len(byName) > 1 {
		return api.Project{}, fmt.Errorf("%d projects are named %q — use the project ID instead (see 'devdash project list')", len(byName), input)
	}
	id, err := resolveProjectPrefix(input, projects)
	if err != nil {
		return api.Project{}, err
	}
	for _, p := range projects {
		if p.ID == id {
			return p, nil
		}
	}
	return api.Project{}, fmt.Errorf("no project found matching %q", input)
}
