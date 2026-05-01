package resolve

import (
	"testing"

	"github.com/devdashproject/devdash-cli/internal/api"
)

var testBeads = []api.Bead{
	{ID: "aaaa0000-1111-2222-3333-444444444444", LocalBeadID: "proj-1"},
	{ID: "bbbb0000-1111-2222-3333-444444444444", LocalBeadID: "proj-2"},
	{ID: "bbbb1111-1111-2222-3333-444444444444", LocalBeadID: "proj-3"},
	{ID: "cccc0000-1111-2222-3333-444444444444", LocalBeadID: "PROJ-4"},
}

func TestResolveFullUUID(t *testing.T) {
	uuid := "aaaa0000-1111-2222-3333-444444444444"
	result, err := ID(uuid, testBeads)
	if err != nil {
		t.Fatalf("ID(%q) failed: %v", uuid, err)
	}
	if result != uuid {
		t.Errorf("ID(%q) = %q, want %q", uuid, result, uuid)
	}
}

func TestResolveFullUUIDNotInList(t *testing.T) {
	// Full UUIDs should be returned as-is, no list lookup needed
	uuid := "dead0000-1111-2222-3333-444444444444"
	result, err := ID(uuid, testBeads)
	if err != nil {
		t.Fatalf("ID(%q) failed: %v", uuid, err)
	}
	if result != uuid {
		t.Errorf("ID(%q) = %q, want %q", uuid, result, uuid)
	}
}

func TestResolvePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"aaaa", "aaaa0000-1111-2222-3333-444444444444"},
		{"aaaa0000", "aaaa0000-1111-2222-3333-444444444444"},
		{"cccc", "cccc0000-1111-2222-3333-444444444444"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ID(tc.input, testBeads)
			if err != nil {
				t.Fatalf("ID(%q) failed: %v", tc.input, err)
			}
			if result != tc.want {
				t.Errorf("ID(%q) = %q, want %q", tc.input, result, tc.want)
			}
		})
	}
}

func TestResolvePrefixAmbiguous(t *testing.T) {
	// "bbbb" matches two beads
	_, err := ID("bbbb", testBeads)
	if err == nil {
		t.Fatal("ID(\"bbbb\") should fail with ambiguous prefix")
	}
}

func TestResolveLocalID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"proj-1", "aaaa0000-1111-2222-3333-444444444444"},
		{"proj-2", "bbbb0000-1111-2222-3333-444444444444"},
		{"PROJ-4", "cccc0000-1111-2222-3333-444444444444"}, // case insensitive
		{"proj-4", "cccc0000-1111-2222-3333-444444444444"}, // case insensitive
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ID(tc.input, testBeads)
			if err != nil {
				t.Fatalf("ID(%q) failed: %v", tc.input, err)
			}
			if result != tc.want {
				t.Errorf("ID(%q) = %q, want %q", tc.input, result, tc.want)
			}
		})
	}
}

func TestResolveLocalIDNotFound(t *testing.T) {
	_, err := ID("proj-999", testBeads)
	if err == nil {
		t.Fatal("ID(\"proj-999\") should fail")
	}
}

func TestResolveEmpty(t *testing.T) {
	_, err := ID("", testBeads)
	if err == nil {
		t.Fatal("ID(\"\") should fail")
	}
}

func TestResolveWhitespace(t *testing.T) {
	result, err := ID("  aaaa  ", testBeads)
	if err != nil {
		t.Fatalf("ID(\"  aaaa  \") failed: %v", err)
	}
	if result != "aaaa0000-1111-2222-3333-444444444444" {
		t.Errorf("got %q", result)
	}
}

var testProjects = []api.Project{
	{ID: "proj-aaaa-0000-0000-0000-0000000000aa", Name: "Project Alpha"},
	{ID: "proj-bbbb-0000-0000-0000-0000000000bb", Name: "Project Beta"},
	{ID: "proj-bbbb-1111-0000-0000-0000000000cc", Name: "Project Gamma"},
	{ID: "test-project-id", Name: "Test Project"},
}

func TestResolveProjectIDFullUUID(t *testing.T) {
	uuid := "proj-aaaa-0000-0000-0000-0000000000aa"
	result, err := resolveProjectPrefix(uuid, testProjects)
	if err != nil {
		t.Fatalf("resolveProjectPrefix(%q) failed: %v", uuid, err)
	}
	if result != uuid {
		t.Errorf("resolveProjectPrefix(%q) = %q, want %q", uuid, result, uuid)
	}
}

func TestResolveProjectIDPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"proj-aaaa", "proj-aaaa-0000-0000-0000-0000000000aa"},
		{"proj-a", "proj-aaaa-0000-0000-0000-0000000000aa"},
		{"test", "test-project-id"},
		{"TEST", "test-project-id"}, // case insensitive
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := resolveProjectPrefix(tc.input, testProjects)
			if err != nil {
				t.Fatalf("resolveProjectPrefix(%q) failed: %v", tc.input, err)
			}
			if result != tc.want {
				t.Errorf("resolveProjectPrefix(%q) = %q, want %q", tc.input, result, tc.want)
			}
		})
	}
}

func TestResolveProjectIDAmbiguous(t *testing.T) {
	_, err := resolveProjectPrefix("proj-bbbb", testProjects)
	if err == nil {
		t.Fatal("resolveProjectPrefix(\"proj-bbbb\") should fail with ambiguous prefix")
	}
	if !contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
}

func TestResolveProjectIDNotFound(t *testing.T) {
	_, err := resolveProjectPrefix("zzzzzzzz", testProjects)
	if err == nil {
		t.Fatal("resolveProjectPrefix(\"zzzzzzzz\") should fail")
	}
	if !contains(err.Error(), "no project found") {
		t.Errorf("expected 'no project found' in error, got: %v", err)
	}
}

func TestResolveProjectIDEmpty(t *testing.T) {
	_, err := resolveProjectPrefix("", testProjects)
	if err == nil {
		t.Fatal("resolveProjectPrefix(\"\") should fail")
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
