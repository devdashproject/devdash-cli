package commands

import (
	"regexp"
	"sort"

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/output"
	"strings"
)

// printJSON pretty-prints any JSON response, falling back to the raw bytes.
func printJSON(data []byte) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(buf.String())
}

// printOutput prints JSON by default, or the --pretty human view.
func printOutput(data []byte, pretty bool, format func([]byte) (string, error)) {
	if pretty {
		if text, err := format(data); err == nil {
			fmt.Print(text)
			return
		}
	}
	printJSON(data)
}

// listItems accepts either a bare JSON array or a {"data": [...]} envelope.
func listItems(data []byte) ([]map[string]interface{}, error) {
	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}
	var page struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}
	return page.Data, nil
}

func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func nested(m map[string]interface{}, key string) map[string]interface{} {
	v, _ := m[key].(map[string]interface{})
	return v
}

// shortTime trims an ISO timestamp to "YYYY-MM-DD HH:MM".
func shortTime(ts string) string {
	ts = strings.Replace(ts, "T", " ", 1)
	if len(ts) > 16 {
		return ts[:16]
	}
	return ts
}

func idList(v interface{}) string {
	items, _ := v.([]interface{})
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, shortID(fmt.Sprint(it)))
	}
	return strings.Join(ids, ", ")
}

var (
	mdBold    = regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	mdCode    = regexp.MustCompile("`([^`\n]+)`")
	mdHeading = regexp.MustCompile(`(?m)^#{1,6}\s+`)
)

// stripMarkdown removes the markup that reads badly in a terminal: bold,
// inline code and heading markers. Text and line breaks are kept.
func stripMarkdown(s string) string {
	s = mdBold.ReplaceAllString(s, "$1$2")
	s = mdCode.ReplaceAllString(s, "$1")
	return mdHeading.ReplaceAllString(s, "")
}

// prettyBead renders an issue for humans, skipping empty fields.
func prettyBead(data []byte) (string, error) {
	var b map[string]interface{}
	if err := json.Unmarshal(data, &b); err != nil {
		return "", err
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s  %s\n", str(b, "id"), str(b, "subject"))
	priority := str(b, "priority")
	if priority != "" {
		priority = "P" + priority
	}
	fmt.Fprintf(&sb, "Status: %s   Priority: %s   Type: %s\n", str(b, "status"), priority, str(b, "beadType"))

	field := func(label, value string) {
		if value != "" {
			fmt.Fprintf(&sb, "%-12s %s\n", label+":", value)
		}
	}
	field("Parent", shortID(str(b, "parentBeadId")))
	field("Blocked by", idList(b["blockedBy"]))
	field("Blocks", idList(b["blocks"]))
	assignee := str(b, "assigneeName")
	if assignee == "" {
		assignee = str(b, "owner")
	}
	field("Assignee", assignee)
	field("Due", str(b, "dueDate"))
	field("Created", shortTime(str(b, "createdAt")))
	field("Updated", shortTime(str(b, "updatedAt")))

	block := func(title, text string) {
		text = stripMarkdown(text)
		if strings.TrimSpace(text) != "" {
			fmt.Fprintf(&sb, "\n%s:\n  %s\n", title, strings.ReplaceAll(strings.TrimSpace(text), "\n", "\n  "))
		}
	}
	block("Description", str(b, "description"))
	block("Pre-instructions", str(b, "preInstructions"))

	if cr := nested(b, "completionResult"); cr != nil {
		block("Completion summary", str(cr, "summary"))
		commit := firstNonEmptyStr(str(cr, "commitSha"), str(cr, "commit"))
		pr := firstNonEmptyStr(str(cr, "prUrl"), str(cr, "pr"))
		if commit != "" || pr != "" {
			sb.WriteString("\n")
			field("Commit", commit)
			field("PR", pr)
		}
	}
	return sb.String(), nil
}

// prettyComments renders one line per comment (author names only, no emails).
func prettyComments(data []byte) (string, error) {
	items, err := listItems(data)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "No comments.\n", nil
	}
	var sb strings.Builder
	for _, c := range items {
		author := str(nested(c, "author"), "displayName")
		if author == "" {
			author = str(c, "authorType")
		}
		fmt.Fprintf(&sb, "[%s] %s: %s\n", shortTime(str(c, "createdAt")), author, stripMarkdown(str(c, "content")))
	}
	return sb.String(), nil
}

// prettyActivity renders one line per event with the useful details only.
func prettyActivity(data []byte) (string, error) {
	items, err := listItems(data)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "No activity.\n", nil
	}
	var sb strings.Builder
	for _, a := range items {
		actor := str(nested(a, "actor"), "displayName")
		if actor == "" {
			actor = str(a, "actorType")
		}
		line := fmt.Sprintf("[%s] %s by %s", shortTime(str(a, "createdAt")), str(a, "action"), actor)
		meta := nested(a, "metadata")
		if from, to := str(meta, "fromStatus"), str(meta, "toStatus"); to != "" {
			line += fmt.Sprintf(": %s → %s", from, to)
		}
		if from, to := str(meta, "fromPriority"), str(meta, "toPriority"); to != "" {
			line += fmt.Sprintf(": P%s → P%s", from, to)
		}
		subject := firstNonEmptyStr(str(meta, "subject"), str(meta, "beadSubject"))
		subject = firstNonEmptyStr(subject, str(a, "artifactName"))
		if subject != "" {
			line += fmt.Sprintf(" — %s", subject)
		}
		sb.WriteString(line + "\n")
	}
	var page struct {
		NextCursor string `json:"nextCursor"`
		HasMore    bool   `json:"hasMore"`
	}
	if json.Unmarshal(data, &page) == nil && page.HasMore && page.NextCursor != "" {
		fmt.Fprintf(&sb, "More: add --cursor=%s\n", page.NextCursor)
	}
	return sb.String(), nil
}

func firstNonEmptyStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// prettyChildren lists a parent's children for show --pretty, in list order.
func prettyChildren(beads []api.Bead, parentID string) string {
	var kids []api.Bead
	for _, b := range beads {
		if b.ParentBeadID == parentID {
			kids = append(kids, b)
		}
	}
	if len(kids) == 0 {
		return ""
	}
	sort.SliceStable(kids, func(i, j int) bool { return listLess(kids[i], kids[j]) })
	var sb strings.Builder
	fmt.Fprintf(&sb, "\nChildren (%d):\n", len(kids))
	for _, k := range kids {
		sb.WriteString("  " + output.FormatListLine(k) + "\n")
	}
	return sb.String()
}
