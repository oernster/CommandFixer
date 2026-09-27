package shell

// ARCHITECTURE.md shows the hook snippet a user's profile receives, because a
// reader wants to see it without running the installer. That makes it a second
// copy of what ProfileSnippet generates; the first copy drifted until it
// described a hook that called a different command, never asked before
// correcting and had no completeness guard. Nothing noticed.
//
// So this reads the block from the document and fails when it is not exactly
// what ProfileSnippet produces. When it fails, the message carries the block
// to paste in.

import (
	"strings"
	"testing"
)

const (
	architectureDoc = "ARCHITECTURE.md"
	hookSection     = "## PowerShell Hook Mechanics"
	snippetFence    = "```powershell\n"
	fenceEnd        = "\n```"

	// docBinaryPath is the placeholder the document shows where a real install
	// has the full path to commandfixer.exe.
	docBinaryPath = `C:\path\to\commandfixer.exe`
)

// documentedSnippet returns the first powershell block in the hook section.
func documentedSnippet(t *testing.T) string {
	t.Helper()
	doc := readRepoFile(t, architectureDoc)

	start := strings.Index(doc, hookSection)
	if start < 0 {
		t.Fatalf("%s has no %q section", architectureDoc, hookSection)
	}
	section := doc[start+len(hookSection):]
	if next := strings.Index(section, "\n## "); next >= 0 {
		section = section[:next]
	}

	open := strings.Index(section, snippetFence)
	if open < 0 {
		t.Fatalf("%s: the %q section shows no powershell block", architectureDoc, hookSection)
	}
	body := section[open+len(snippetFence):]
	end := strings.Index(body, fenceEnd)
	if end < 0 {
		t.Fatalf("%s: the powershell block in %q is never closed", architectureDoc, hookSection)
	}
	return body[:end]
}

// firstDifference names the first line at which two texts part company.
func firstDifference(got, want string) (int, string, string) {
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			return i + 1, g, w
		}
	}
	return 0, "", ""
}

func TestArchitectureDocShowsTheInstalledSnippet(t *testing.T) {
	t.Parallel()
	got := documentedSnippet(t)
	want := strings.TrimSuffix(ProfileSnippet(docBinaryPath), "\n")
	if got == want {
		return
	}
	line, g, w := firstDifference(got, want)
	t.Errorf(
		"the snippet in %s has drifted from ProfileSnippet, first at line %d of the block:\n"+
			"  document:       %q\n  ProfileSnippet: %q\n\n"+
			"Replace the block with this:\n\n%s%s%s",
		architectureDoc, line, g, w, snippetFence, want, fenceEnd,
	)
}
