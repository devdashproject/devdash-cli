package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockRoute defines a mock API route.
type MockRoute struct {
	Method   string
	Path     string // prefix match
	Status   int
	Response interface{}
}

// MockServer creates an httptest server with the given routes.
// Returns the server and an API client pointed at it.
func MockServer(t *testing.T, routes []MockRoute) (*httptest.Server, *Client) {
	t.Helper()

	mux := http.NewServeMux()

	// Register a catch-all handler that matches routes by method and path prefix
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, route := range routes {
			if r.Method == route.Method && matchPath(r.URL.Path, route.Path) {
				w.Header().Set("Content-Type", "application/json")
				status := route.Status
				if status == 0 {
					status = 200
				}
				w.WriteHeader(status)
				if route.Response != nil {
					json.NewEncoder(w).Encode(route.Response)
				}
				return
			}
		}
		w.WriteHeader(404)
		w.Write([]byte(`{"error": "no matching route"}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Client expects BaseURL + "/api" + path, so strip /api from mock
	client := New(server.URL, "test-token", "test")
	// Override: the mock server doesn't have /api prefix, so adjust
	client.BaseURL = server.URL + "/api"

	return server, client
}

func matchPath(actual, pattern string) bool {
	// Remove /api prefix from actual path for matching
	actual = strings.TrimPrefix(actual, "/api")
	return strings.HasPrefix(actual, pattern)
}

// SampleBeads returns test bead data for use in tests.
func SampleBeads() []Bead {
	return []Bead{
		{
			ID: "aaaa0000-0000-0000-0000-000000000001", LocalBeadID: "test-1",
			Subject: "Ready task", Status: "pending", Priority: 1, BeadType: "task",
			AssignedTo: "test-user-id",
		},
		{
			ID: "bbbb0000-0000-0000-0000-000000000002", LocalBeadID: "test-2",
			Subject: "Blocked task", Status: "pending", Priority: 2, BeadType: "task",
			BlockedBy: []string{"cccc0000-0000-0000-0000-000000000003"},
			AssignedTo: "other-user-id",
		},
		{
			ID: "cccc0000-0000-0000-0000-000000000003", LocalBeadID: "test-3",
			Subject: "In progress", Status: "in_progress", Priority: 0, BeadType: "feature",
			AssignedTo: "test-user-id",
		},
		{
			ID: "dddd0000-0000-0000-0000-000000000004", LocalBeadID: "test-4",
			Subject: "Completed", Status: "completed", Priority: 1, BeadType: "bug",
		},
		{
			ID: "eeee0000-0000-0000-0000-000000000005", LocalBeadID: "test-5",
			Subject: "Thought item", Status: "pending", Priority: 3, BeadType: "thought",
			AssignedTo: "other-user-id",
		},
		{
			ID: "ffff0000-0000-0000-0000-000000000006", LocalBeadID: "test-6",
			Subject: "Stale task", Status: "in_progress", Priority: 2, BeadType: "task",
			StaleMinutes: 45, StaleSince: "2026-03-27T10:00:00Z",
			AssignedTo: "test-user-id",
		},
		{
			ID: "1111aaaa-0000-0000-0000-000000000007", LocalBeadID: "test-7",
			Subject: "Scored task", Status: "pending", Priority: 1, BeadType: "task",
			BurnIntelligence: &BurnIntelligence{
				AutomabilityScore: 85,
				AutomabilityGrade: "A",
				ComplexityScore:   30,
			},
		},
	}
}

// SampleProjects returns test project data.
func SampleProjects() []Project {
	return []Project{
		{ID: "proj-0001-0000-0000-000000000001", Name: "test-project", GithubRepo: "user/test-project"},
		{ID: "proj-0002-0000-0000-000000000002", Name: "another-project"},
	}
}
