package diagnostics

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	githubRepo          = "AlexBelWeb/flibustahub"
	githubNewIssueLimit = 8000

	IssueKindBug  = "bug"
	IssueKindIdea = "idea"
)

// IssueReport is the preview payload. Body is the full markdown; it is never silently trimmed.
type IssueReport struct {
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	GitHubURL string `json:"githubUrl"`
	URLFits   bool   `json:"urlFits"`
}

// BuildReport assembles a GitHub issue preview from the user's text and a snapshot.
func BuildReport(kind, description string, snap Snapshot) IssueReport {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind != IssueKindIdea {
		kind = IssueKindBug
	}
	desc := strings.TrimSpace(description)
	title := issueTitle(kind, desc)
	body := issueBody(kind, desc, snap)
	gh, fits := githubURL(title, body)
	return IssueReport{
		Kind:      kind,
		Title:     title,
		Body:      body,
		GitHubURL: gh,
		URLFits:   fits,
	}
}

func (s *Service) BuildReport(kind, description string, snap Snapshot) IssueReport {
	return BuildReport(kind, description, snap)
}

func issueTitle(kind, desc string) string {
	prefix := "Bug"
	if kind == IssueKindIdea {
		prefix = "Idea"
	}
	first := desc
	if i := strings.IndexAny(first, "\r\n"); i >= 0 {
		first = first[:i]
	}
	first = strings.TrimSpace(first)
	if first == "" {
		return prefix
	}
	runes := []rune(first)
	if len(runes) > 80 {
		first = string(runes[:80])
	}
	return prefix + ": " + first
}

func issueBody(kind, desc string, snap Snapshot) string {
	var b strings.Builder
	b.WriteString("## Description\n\n")
	if desc == "" {
		b.WriteString("(empty)\n")
	} else {
		b.WriteString(desc)
		b.WriteByte('\n')
	}
	b.WriteString("\n## Environment\n\n")
	fmt.Fprintf(&b, "- App: %s (%s)\n", orDash(snap.Version), orDash(snap.Commit))
	fmt.Fprintf(&b, "- Built: %s\n", orDash(snap.BuildDate))
	fmt.Fprintf(&b, "- OS: %s/%s\n", orDash(snap.OS), orDash(snap.Arch))
	fmt.Fprintf(&b, "- WebView: %s\n", orDash(snap.WebView))
	fmt.Fprintf(&b, "- Dump: %s\n", orDash(snap.INPXVersion))
	fmt.Fprintf(&b, "- Catalog: %d works, %d authors, imported %s\n", snap.Works, snap.Authors, orDash(snap.ImportedAt))
	fmt.Fprintf(&b, "- Unnamed genres: %d\n", snap.UnnamedGenres)
	fmt.Fprintf(&b, "- AI provider: %s\n", orDash(snap.AIProvider))
	fmt.Fprintf(&b, "- OPDS enabled: %v\n", snap.OPDSEnabled)
	if snap.Paths.DBPath != "" {
		fmt.Fprintf(&b, "- Database: `%s`\n", snap.Paths.DBPath)
	}
	if snap.Paths.CoversDir != "" {
		fmt.Fprintf(&b, "- Covers: `%s`\n", snap.Paths.CoversDir)
	}
	if len(snap.RecentLogLines) > 0 {
		b.WriteString("\n## Recent warnings and errors\n\n```\n")
		for _, line := range snap.RecentLogLines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		b.WriteString("```\n")
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func githubURL(title, body string) (string, bool) {
	base := "https://github.com/" + githubRepo + "/issues/new"
	full := base + "?" + url.Values{"title": {title}, "body": {body}}.Encode()
	if len(full) <= githubNewIssueLimit {
		return full, true
	}
	short := base + "?" + url.Values{"title": {title}}.Encode()
	return short, false
}
