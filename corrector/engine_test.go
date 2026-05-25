package corrector

import (
	"testing"

	"github.com/oernster/commandfixer/config"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildEngine(t *testing.T, typos []config.TypoEntry) *Engine {
	t.Helper()
	cfg := &config.Config{Typos: typos}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned unexpected error: %v", err)
	}
	return e
}

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew_EmptyConfig(t *testing.T) {
	t.Parallel()
	e, err := New(&config.Config{})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil Engine")
	}
	if len(e.rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(e.rules))
	}
}

func TestNew_LiteralRules(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Typos: []config.TypoEntry{
			{From: "git sattus", To: "git status"},
			{From: "ls --afl", To: "ls -afl"},
		},
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(e.rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(e.rules))
	}
	// Literal rules must have a nil regex.
	for i, r := range e.rules {
		if r.regex != nil {
			t.Errorf("rule[%d]: expected nil regex for literal rule", i)
		}
	}
}

func TestNew_RegexRule_Valid(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Typos: []config.TypoEntry{
			{From: `git\s+sattus`, To: "git status", Regex: true},
		},
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.rules[0].regex == nil {
		t.Error("expected non-nil regex for regex rule")
	}
}

func TestNew_RegexRule_InvalidPattern(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Typos: []config.TypoEntry{
			{From: `[invalid(`, To: "something", Regex: true},
		},
	}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for invalid regex pattern, got nil")
	}
}

// ---------------------------------------------------------------------------
// Correct - no match
// ---------------------------------------------------------------------------

func TestCorrect_NoRules(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, nil)
	result := e.Correct("git status")
	if result.Changed {
		t.Error("expected Changed=false with no rules")
	}
	if result.Corrected != "git status" {
		t.Errorf("expected unchanged command, got %q", result.Corrected)
	}
	if result.Original != "git status" {
		t.Errorf("expected Original preserved, got %q", result.Original)
	}
}

func TestCorrect_NoMatch(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: "git sattus", To: "git status"},
	})
	result := e.Correct("git status")
	if result.Changed {
		t.Error("expected no change for already-correct command")
	}
	if result.Corrected != "git status" {
		t.Errorf("expected %q, got %q", "git status", result.Corrected)
	}
}

// ---------------------------------------------------------------------------
// Correct - literal rules
// ---------------------------------------------------------------------------

func TestCorrect_LiteralMatch_FullCommand(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: "git sattus", To: "git status"},
	})
	result := e.Correct("git sattus")
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if result.Corrected != "git status" {
		t.Errorf("expected %q, got %q", "git status", result.Corrected)
	}
	if result.Original != "git sattus" {
		t.Errorf("expected Original %q, got %q", "git sattus", result.Original)
	}
	if result.RuleFrom != "git sattus" {
		t.Errorf("expected RuleFrom %q, got %q", "git sattus", result.RuleFrom)
	}
	if result.RuleTo != "git status" {
		t.Errorf("expected RuleTo %q, got %q", "git status", result.RuleTo)
	}
}

func TestCorrect_LiteralMatch_Substring(t *testing.T) {
	t.Parallel()
	// "sattus" appears as a substring; strings.ReplaceAll should catch it.
	e := buildEngine(t, []config.TypoEntry{
		{From: "sattus", To: "status"},
	})
	result := e.Correct("git sattus -v")
	if !result.Changed {
		t.Fatal("expected Changed=true for substring match")
	}
	if result.Corrected != "git status -v" {
		t.Errorf("expected %q, got %q", "git status -v", result.Corrected)
	}
}

func TestCorrect_MultipleRulesApply(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: "gti", To: "git"},
		{From: "sattus", To: "status"},
	})
	// Both typos present; both rules should fire.
	result := e.Correct("gti sattus")
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if result.Corrected != "git status" {
		t.Errorf("expected %q, got %q", "git status", result.Corrected)
	}
}

func TestCorrect_LastFiredRuleRecorded(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: "aaa", To: "AAA"},
		{From: "bbb", To: "BBB"},
	})
	result := e.Correct("aaa bbb")
	if result.RuleFrom != "bbb" {
		t.Errorf("expected last rule From %q, got %q", "bbb", result.RuleFrom)
	}
	if result.RuleTo != "BBB" {
		t.Errorf("expected last rule To %q, got %q", "BBB", result.RuleTo)
	}
}

// ---------------------------------------------------------------------------
// Correct - regex rules
// ---------------------------------------------------------------------------

func TestCorrect_RegexMatch(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: `git\s+sattus`, To: "git status", Regex: true},
	})
	result := e.Correct("git  sattus") // two spaces
	if !result.Changed {
		t.Fatal("expected Changed=true for regex rule")
	}
	if result.Corrected != "git status" {
		t.Errorf("expected %q, got %q", "git status", result.Corrected)
	}
}

func TestCorrect_RegexNoMatch(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: `^docker\s+pss$`, To: "docker ps", Regex: true},
	})
	result := e.Correct("git status")
	if result.Changed {
		t.Error("expected no change when regex does not match")
	}
}

func TestCorrect_RegexCapturingGroup(t *testing.T) {
	t.Parallel()
	// Replace any occurrence of "git commit -a" with "git commit -am"
	e := buildEngine(t, []config.TypoEntry{
		{From: `(git commit) -a\b`, To: "$1 -am", Regex: true},
	})
	result := e.Correct("git commit -a")
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if result.Corrected != "git commit -am" {
		t.Errorf("expected %q, got %q", "git commit -am", result.Corrected)
	}
}

// ---------------------------------------------------------------------------
// Correct - mixed literal and regex rules
// ---------------------------------------------------------------------------

func TestCorrect_MixedRules(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: "git sattus", To: "git status"},               // literal
		{From: `docker\s+pss`, To: "docker ps", Regex: true}, // regex
	})
	// Only first rule matches.
	result := e.Correct("git sattus")
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if result.Corrected != "git status" {
		t.Errorf("expected %q, got %q", "git status", result.Corrected)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestCorrect_EmptyCommand(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: "git sattus", To: "git status"},
	})
	result := e.Correct("")
	if result.Changed {
		t.Error("expected no change for empty command")
	}
	if result.Corrected != "" {
		t.Errorf("expected empty corrected, got %q", result.Corrected)
	}
}

func TestCorrect_OriginalPreserved(t *testing.T) {
	t.Parallel()
	e := buildEngine(t, []config.TypoEntry{
		{From: "git sattus", To: "git status"},
	})
	result := e.Correct("git sattus")
	if result.Original != "git sattus" {
		t.Errorf("Original must not be mutated; got %q", result.Original)
	}
}
