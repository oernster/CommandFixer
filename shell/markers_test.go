package shell

// The hook markers and the profile paths exist twice, in Go and in PowerShell,
// and they have to.
//
// The binary writes the hook block into a user's profile and normally removes
// it again. But an uninstall has to work when the binary is already gone, so
// uninstall.ps1 carries a fallback that strips the block itself, and that
// fallback needs the same markers and the same paths. Two languages cannot
// share one literal.
//
// So the scripts define theirs once, in profile-hook.ps1, and this reads that
// file and fails if the Go side has drifted from it. Without this, changing a
// marker on one side only leaves a hook line in someone's profile that nothing
// can find to remove, running on every prompt they type, and nothing reports
// it: the failure lands on a user's machine, not on a developer's.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sharedDefinitions is the one place the PowerShell scripts state any of this.
const sharedDefinitions = "profile-hook.ps1"

// assignment matches a single-quoted PowerShell scalar assignment, so the value
// is read from the file rather than restated here (which would defeat the
// point of the test).
func assignment(t *testing.T, script, name string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?m)^\$` + name + `\s*=\s*'([^']*)'`)
	match := pattern.FindStringSubmatch(script)
	if match == nil {
		t.Fatalf("%s does not assign $%s with a single-quoted value", sharedDefinitions, name)
	}
	return match[1]
}

func readSharedDefinitions(t *testing.T) string {
	t.Helper()
	// Tests run with the package directory as the working directory.
	path := filepath.Join("..", sharedDefinitions)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(content)
}

func TestSnippetMarkersMatchTheScripts(t *testing.T) {
	t.Parallel()
	script := readSharedDefinitions(t)

	if got := assignment(t, script, "CommandFixerSnippetStart"); got != snippetStart {
		t.Errorf(
			"start marker has drifted: Go has %q, %s has %q."+
				" A profile written with one and stripped with the other keeps"+
				" a dead hook line forever",
			snippetStart, sharedDefinitions, got,
		)
	}
	if got := assignment(t, script, "CommandFixerSnippetEnd"); got != snippetEnd {
		t.Errorf(
			"end marker has drifted: Go has %q, %s has %q",
			snippetEnd, sharedDefinitions, got,
		)
	}
}

func TestProfilePathsMatchTheScripts(t *testing.T) {
	t.Parallel()
	script := readSharedDefinitions(t)

	paths, err := AllProfilePaths()
	if err != nil {
		t.Fatalf("AllProfilePaths: %v", err)
	}

	// The script joins $HOME with a relative path, so compare the tail rather
	// than the whole thing: the home directory is the same by construction and
	// is not what can drift.
	for _, path := range paths {
		tail := filepath.Join(filepath.Base(filepath.Dir(filepath.Dir(path))),
			filepath.Base(filepath.Dir(path)),
			filepath.Base(path))
		if !strings.Contains(script, tail) {
			t.Errorf(
				"%s does not mention %q, so its fallback would not clean the"+
					" profile the binary installs into",
				sharedDefinitions, tail,
			)
		}
	}
}

func TestTheScriptsListEveryProfileGoInstallsInto(t *testing.T) {
	t.Parallel()
	script := readSharedDefinitions(t)

	paths, err := AllProfilePaths()
	if err != nil {
		t.Fatalf("AllProfilePaths: %v", err)
	}

	// Guards the test above from passing vacuously: if Go ever returns fewer
	// profiles than the script sweeps, the script is cleaning something the
	// binary no longer writes, which is the same drift in the other direction.
	inScript := strings.Count(script, "profile.ps1")
	if inScript != len(paths) {
		t.Errorf(
			"Go installs into %d profiles but %s names %d",
			len(paths), sharedDefinitions, inScript,
		)
	}
}
