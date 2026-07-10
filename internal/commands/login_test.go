package commands

import (
	"strings"
	"testing"
)

func TestBuildCLITokenURL(t *testing.T) {
	const base = "https://api.example.com"

	t.Run("no provider omits the param", func(t *testing.T) {
		got := buildCLITokenURL(base, 8080, "abc123", "")
		want := "https://api.example.com/api/auth/cli-token?port=8080&nonce=abc123"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("github provider is appended", func(t *testing.T) {
		got := buildCLITokenURL(base, 8080, "abc123", "github")
		want := "https://api.example.com/api/auth/cli-token?port=8080&nonce=abc123&provider=github"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("provider value is url-escaped", func(t *testing.T) {
		// A value needing escaping must not break the query string.
		got := buildCLITokenURL(base, 8080, "abc123", "goo gle&x")
		if !strings.HasSuffix(got, "&provider=goo+gle%26x") {
			t.Fatalf("expected escaped provider suffix, got %q", got)
		}
	})
}
