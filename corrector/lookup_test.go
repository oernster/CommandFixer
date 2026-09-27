package corrector

import "testing"

// Tests for the command lookup: a first token that runs as a real command is
// never renamed, however close it sits to something in the database.
//
// The case that raised it: VS Code's "code" is one edit from the Windows
// "mode", so "code --install-extension x" was offered as "mode ..." even though
// code.cmd was on PATH.

const codeInstall = "code --install-extension rust-lang.rust-analyzer"

// lookupOf answers true for exactly the names given.
func lookupOf(names ...string) CommandLookup {
	known := make(map[string]bool, len(names))
	for _, name := range names {
		known[name] = true
	}
	return func(name string) bool { return known[name] }
}

func TestSuggest_RealCommand_NotRenamed(t *testing.T) {
	t.Parallel()
	e := New(0).WithCommandLookup(lookupOf("code"))
	result, found := e.Suggest(codeInstall)
	if found {
		t.Errorf("a command on PATH was corrected to %q", result)
	}
	if result != codeInstall {
		t.Errorf("expected input unchanged, got %q", result)
	}
}

// Without the lookup the database alone does rename it. This pins the lookup
// as the thing that stops it, rather than some other rule that happens to.
func TestSuggest_NoLookup_DatabaseAloneRenames(t *testing.T) {
	t.Parallel()
	result, found := New(0).Suggest(codeInstall)
	if !found || result != "mode --install-extension rust-lang.rust-analyzer" {
		t.Errorf("expected the database to offer mode, got %q (found=%v)", result, found)
	}
}

// The lookup guards the tool name only. A known tool's mistyped subcommand is
// still corrected even though the tool itself is, of course, on PATH.
func TestSuggest_RealCommand_SubcommandStillCorrected(t *testing.T) {
	t.Parallel()
	e := New(0).WithCommandLookup(lookupOf("git"))
	result, found := e.Suggest("git sattus")
	if !found || result != "git status" {
		t.Errorf("expected %q, got %q (found=%v)", "git status", result, found)
	}
}

// A mistyped tool the lookup does not know is still corrected.
func TestSuggest_UnknownToLookup_StillCorrected(t *testing.T) {
	t.Parallel()
	e := New(0).WithCommandLookup(lookupOf("code"))
	result, found := e.Suggest("dokcer ps")
	if !found || result != "docker ps" {
		t.Errorf("expected %q, got %q (found=%v)", "docker ps", result, found)
	}
}

// An alias is a habitual typo only while nothing of that name is installed. A
// real gti on PATH is a command in its own right and is left alone, subcommand
// and all.
func TestSuggest_AliasOnPath_NotRewritten(t *testing.T) {
	t.Parallel()
	e := New(0).WithCommandLookup(lookupOf("gti"))
	for _, input := range []string{"gti status", "gti sattus"} {
		if result, found := e.Suggest(input); found || result != input {
			t.Errorf("%q: a gti on PATH was rewritten to %q (found=%v)", input, result, found)
		}
	}
}

// With no gti installed the alias still applies, even under a lookup that
// knows the tool it points at.
func TestSuggest_AliasNotOnPath_StillRewritten(t *testing.T) {
	t.Parallel()
	e := New(0).WithCommandLookup(lookupOf("git"))
	if result, found := e.Suggest("gti sattus"); !found || result != "git status" {
		t.Errorf("expected %q, got %q (found=%v)", "git status", result, found)
	}
}

func TestWithCommandLookup_LeavesOriginalUntouched(t *testing.T) {
	t.Parallel()
	original := New(0)
	_ = original.WithCommandLookup(lookupOf("code"))
	if _, found := original.Suggest(codeInstall); !found {
		t.Error("WithCommandLookup changed the engine it was called on")
	}
}

func TestWithCommandLookup_Nil_KeepsCurrentLookup(t *testing.T) {
	t.Parallel()
	e := New(0).WithCommandLookup(lookupOf("code")).WithCommandLookup(nil)
	if _, found := e.Suggest(codeInstall); found {
		t.Error("a nil lookup replaced the one already in use")
	}
}

func TestWithCommandLookup_KeepsThreshold(t *testing.T) {
	t.Parallel()
	const custom = 0.9
	if got := New(custom).WithCommandLookup(lookupOf()).Threshold(); got != custom {
		t.Errorf("expected threshold %v to survive the copy, got %v", custom, got)
	}
}
