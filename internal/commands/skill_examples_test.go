package commands

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// The installed skill (.claude/skills/devdash-cli) is what agents copy
// commands from. This test keeps it honest: every `devdash`/`dd` command in a
// fenced code block must name a real command, and every --flag must exist on
// it. Heredoc bodies and trailing comments are ignored.

var (
	fenceRe   = regexp.MustCompile("(?s)```[a-z]*\n(.*?)```")
	heredocRe = regexp.MustCompile(`(?s)"\$\(cat <<'?EOF'?\n.*?\nEOF\n\)"`)
	contRe    = regexp.MustCompile(`\\\n\s*`)
)

// splitArgs is a minimal shell-ish tokenizer: handles single/double quotes and
// drops $(...) substitutions and everything after an unquoted #, |, && or >.
func splitArgs(line string) []string {
	var out []string
	var cur strings.Builder
	var quote rune
	depth := 0
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case depth > 0:
			if r == '(' {
				depth++
			} else if r == ')' {
				depth--
			}
		case r == '"' || r == '\'':
			quote = r
			cur.WriteString("") // an empty quoted string still counts as a token
		case r == '$' && i+1 < len(rs) && rs[i+1] == '(':
			depth = 1
			i++
			cur.WriteString("x")
		case r == ' ' || r == '\t':
			flush()
		case r == '#' && cur.Len() == 0, r == '|', r == '>', r == '&' && i+1 < len(rs) && rs[i+1] == '&':
			flush()
			return out
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

func TestSkillExamplesMatchCLI(t *testing.T) {
	root := filepath.Join("..", "..", ".claude", "skills", "devdash-cli")
	var files []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".md") {
			files = append(files, p)
		}
		return nil
	})
	if len(files) == 0 {
		t.Skip("skill directory not present")
	}

	deps := &Deps{
		Cfg:    &config.Config{ProjectID: "p", APIURL: "http://127.0.0.1:0", Token: "t", ConfigDir: t.TempDir()},
		Client: apiPkg.New("http://127.0.0.1:0", "t", Version),
	}
	rootCmd := NewRootCmd(deps)
	checked := 0

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range fenceRe.FindAllStringSubmatch(string(raw), -1) {
			block := contRe.ReplaceAllString(m[1], "")
			block = heredocRe.ReplaceAllString(block, `"x"`)
			for _, line := range strings.Split(block, "\n") {
				line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "$ "))
				if !strings.HasPrefix(line, "devdash ") && !strings.HasPrefix(line, "dd ") {
					continue
				}
				args := splitArgs(line)[1:]
				// Drop the global --project=... flag wherever it appears.
				var clean []string
				for _, a := range args {
					if strings.HasPrefix(a, "--project") {
						continue
					}
					clean = append(clean, a)
				}
				if len(clean) == 0 {
					continue
				}
				if clean[0] == "help" {
					if len(clean) > 1 {
						topic := clean[1]
						_, known := helpTopics[topic]
						if cmd, _, err := rootCmd.Find(clean[1:]); !known && topic != "cross-project" && (err != nil || cmd == rootCmd) {
							t.Errorf("%s: %q is not a help topic or command", f, line)
						}
					}
					checked++
					continue
				}
				cmd, rest, err := rootCmd.Find(clean)
				if err != nil || cmd == rootCmd {
					t.Errorf("%s: %q does not name a devdash command", f, line)
					continue
				}
				checked++
				for _, a := range rest {
					if !strings.HasPrefix(a, "--") || a == "--" {
						continue
					}
					name := strings.SplitN(strings.TrimPrefix(a, "--"), "=", 2)[0]
					if name == "help" || lookupFlag(cmd, name) {
						continue
					}
					t.Errorf("%s: flag --%s does not exist on `%s` (line: %s)", f, name, cmd.CommandPath(), line)
				}
			}
		}
	}
	if checked < 30 {
		t.Fatalf("only %d skill commands checked; parser likely broke", checked)
	}
}

func lookupFlag(cmd *cobra.Command, name string) bool {
	var found bool
	visit := func(fs *pflag.FlagSet) {
		if fs != nil && fs.Lookup(name) != nil {
			found = true
		}
	}
	visit(cmd.Flags())
	visit(cmd.InheritedFlags())
	visit(cmd.PersistentFlags())
	return found
}
