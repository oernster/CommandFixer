package corrector

import "testing"

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew_ZeroThreshold_UsesDefault(t *testing.T) {
	t.Parallel()
	e := New(0)
	if e.Threshold() != defaultThreshold {
		t.Errorf("expected default threshold %v, got %v", defaultThreshold, e.Threshold())
	}
}

func TestNew_NegativeThreshold_UsesDefault(t *testing.T) {
	t.Parallel()
	e := New(-0.5)
	if e.Threshold() != defaultThreshold {
		t.Errorf("expected default threshold %v, got %v", defaultThreshold, e.Threshold())
	}
}

func TestNew_AboveOneThreshold_UsesDefault(t *testing.T) {
	t.Parallel()
	e := New(1.5)
	if e.Threshold() != defaultThreshold {
		t.Errorf("expected default threshold %v, got %v", defaultThreshold, e.Threshold())
	}
}

func TestNew_ValidThreshold_Stored(t *testing.T) {
	t.Parallel()
	e := New(0.8)
	if e.Threshold() != 0.8 {
		t.Errorf("expected threshold 0.8, got %v", e.Threshold())
	}
}

func TestNew_MaxThreshold_Valid(t *testing.T) {
	t.Parallel()
	e := New(1.0)
	if e.Threshold() != 1.0 {
		t.Errorf("expected threshold 1.0, got %v", e.Threshold())
	}
}

// ---------------------------------------------------------------------------
// Suggest - no suggestion cases
// ---------------------------------------------------------------------------

func TestSuggest_EmptyInput(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("")
	if found {
		t.Error("expected found=false for empty input")
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestSuggest_SingleToken(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("git")
	if found {
		t.Error("expected found=false for single-token input")
	}
	if result != "git" {
		t.Errorf("expected %q unchanged, got %q", "git", result)
	}
}

func TestSuggest_UnknownTool(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("foobarize baz")
	if found {
		t.Error("expected found=false for unknown tool")
	}
	if result != "foobarize baz" {
		t.Errorf("expected input unchanged, got %q", result)
	}
}

func TestSuggest_ExactSubcommand_NoCorrection(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("git status")
	if found {
		t.Error("expected found=false when subcommand is already exact")
	}
	if result != "git status" {
		t.Errorf("expected input unchanged, got %q", result)
	}
}

func TestSuggest_TooFarOff_NoCorrection(t *testing.T) {
	t.Parallel()
	// "abcdefghij" is far from any git subcommand.
	e := New(0)
	_, found := e.Suggest("git abcdefghij")
	if found {
		t.Error("expected found=false when subcommand is too dissimilar")
	}
}

func TestSuggest_BelowCustomThreshold(t *testing.T) {
	t.Parallel()
	// "sattus" vs "status": similarity ~0.667, below 0.8 threshold.
	e := New(0.8)
	_, found := e.Suggest("git sattus")
	if found {
		t.Errorf("expected no suggestion when similarity is below custom threshold 0.8")
	}
}

// ---------------------------------------------------------------------------
// Suggest - correction cases
// ---------------------------------------------------------------------------

func TestSuggest_GitStatus_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("git sattus")
	if !found {
		t.Fatal("expected found=true for 'git sattus'")
	}
	if result != "git status" {
		t.Errorf("expected %q, got %q", "git status", result)
	}
}

func TestSuggest_GitCommit_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("git comit")
	if !found {
		t.Fatal("expected found=true for 'git comit'")
	}
	if result != "git commit" {
		t.Errorf("expected %q, got %q", "git commit", result)
	}
}

func TestSuggest_GitBranch_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("git branhc")
	if !found {
		t.Fatal("expected found=true for 'git branhc'")
	}
	if result != "git branch" {
		t.Errorf("expected %q, got %q", "git branch", result)
	}
}

func TestSuggest_DockerPs_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("docker pss")
	if !found {
		t.Fatal("expected found=true for 'docker pss'")
	}
	if result != "docker ps" {
		t.Errorf("expected %q, got %q", "docker ps", result)
	}
}

func TestSuggest_KubectlGet_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("kubectl gt pods")
	if !found {
		t.Fatal("expected found=true for 'kubectl gt pods'")
	}
	if result != "kubectl get pods" {
		t.Errorf("expected %q, got %q", "kubectl get pods", result)
	}
}

func TestSuggest_PreservesTrailingArgs(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("git sattus -v --short origin")
	if !found {
		t.Fatal("expected found=true")
	}
	if result != "git status -v --short origin" {
		t.Errorf("expected %q, got %q", "git status -v --short origin", result)
	}
}

func TestSuggest_OriginalUnchangedOnNoMatch(t *testing.T) {
	t.Parallel()
	e := New(0)
	input := "unknowntool subcmd"
	result, found := e.Suggest(input)
	if found {
		t.Error("expected found=false")
	}
	if result != input {
		t.Errorf("expected input %q unchanged, got %q", input, result)
	}
}

// ---------------------------------------------------------------------------
// levenshtein
// ---------------------------------------------------------------------------

func TestLevenshtein_EmptyStrings(t *testing.T) {
	t.Parallel()
	if got := levenshtein("", ""); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestLevenshtein_EmptyA(t *testing.T) {
	t.Parallel()
	if got := levenshtein("", "abc"); got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestLevenshtein_EmptyB(t *testing.T) {
	t.Parallel()
	if got := levenshtein("abc", ""); got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestLevenshtein_EqualStrings(t *testing.T) {
	t.Parallel()
	if got := levenshtein("status", "status"); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestLevenshtein_SingleSubstitution(t *testing.T) {
	t.Parallel()
	// "ps" -> "pss": insert one char.
	if got := levenshtein("pss", "ps"); got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestLevenshtein_KnownDistance(t *testing.T) {
	t.Parallel()
	// "comit" -> "commit": insert one 'm'.
	if got := levenshtein("comit", "commit"); got != 1 {
		t.Errorf("expected distance 1 for comit/commit, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// similarity
// ---------------------------------------------------------------------------

func TestSimilarity_EqualStrings(t *testing.T) {
	t.Parallel()
	s := similarity("status", "status")
	if s != 1.0 {
		t.Errorf("expected 1.0 for equal strings, got %v", s)
	}
}

func TestSimilarity_EmptyStrings(t *testing.T) {
	t.Parallel()
	s := similarity("", "")
	if s != 1.0 {
		t.Errorf("expected 1.0 for both empty, got %v", s)
	}
}

func TestSimilarity_CompletelyDifferent(t *testing.T) {
	t.Parallel()
	// "abc" vs "xyz": all substitutions, distance=3, max=3, sim=0.
	s := similarity("abc", "xyz")
	if s != 0.0 {
		t.Errorf("expected 0.0 for fully different strings, got %v", s)
	}
}

func TestSimilarity_AboveDefaultThreshold(t *testing.T) {
	t.Parallel()
	// "comit" vs "commit": distance=1, max=6, sim~0.833.
	s := similarity("comit", "commit")
	if s <= defaultThreshold {
		t.Errorf("expected similarity > %v, got %v", defaultThreshold, s)
	}
}

func TestSimilarity_AboveDefaultThresholdForStatusTypo(t *testing.T) {
	t.Parallel()
	// "sattus" vs "status": distance=2, max=6, sim~0.667.
	s := similarity("sattus", "status")
	if s <= defaultThreshold {
		t.Errorf("expected similarity > %v for sattus/status, got %v", defaultThreshold, s)
	}
}
