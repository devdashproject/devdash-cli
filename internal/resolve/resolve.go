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
