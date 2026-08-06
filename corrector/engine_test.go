package corrector

import "testing"

// Tests for the correction policy: what Suggest decides to do.
//
// The Windows command coverage lives in windows_test.go and the string metric
// in distance_test.go, mirroring the split of the package itself.

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
	// "sattus" vs "status": a single adjacent transposition, similarity ~0.833,
	// which is below a strict 0.9 threshold.
	e := New(0.9)
	_, found := e.Suggest("git sattus")
	if found {
		t.Errorf("expected no suggestion when similarity is below custom threshold 0.9")
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
// Suggest - command-name alias (gti -> git) and tool-name correction
// ---------------------------------------------------------------------------

func TestSuggest_GitAlias_Push(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("gti push")
	if !found {
		t.Fatal("expected found=true for 'gti push'")
	}
	if result != "git push" {
		t.Errorf("expected %q, got %q", "git push", result)
	}
}

func TestSuggest_GitAlias_Status(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("gti status")
	if !found {
		t.Fatal("expected found=true for 'gti status'")
	}
	if result != "git status" {
		t.Errorf("expected %q, got %q", "git status", result)
	}
}

func TestSuggest_GitAlias_WithSubcommandTypo(t *testing.T) {
	t.Parallel()
	// Both the command name (gti) and the subcommand (psuh) are wrong.
	e := New(0)
	result, found := e.Suggest("gti psuh")
	if !found {
		t.Fatal("expected found=true for 'gti psuh'")
	}
	if result != "git push" {
		t.Errorf("expected %q, got %q", "git push", result)
	}
}

func TestSuggest_GitAlias_PreservesTrailingArgs(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("gti push origin main")
	if !found {
		t.Fatal("expected found=true for 'gti push origin main'")
	}
	if result != "git push origin main" {
		t.Errorf("expected %q, got %q", "git push origin main", result)
	}
}

func TestSuggest_GitPush_Variants(t *testing.T) {
	t.Parallel()
	e := New(0)
	cases := []struct {
		in   string
		want string
	}{
		{"git puh", "git push"},  // single deletion
		{"git pus", "git push"},  // single deletion
		{"git psuh", "git push"}, // adjacent transposition
		{"git puhs", "git push"}, // adjacent transposition
	}
	for _, tc := range cases {
		result, found := e.Suggest(tc.in)
		if !found {
			t.Fatalf("expected found=true for %q", tc.in)
		}
		if result != tc.want {
			t.Errorf("for %q: expected %q, got %q", tc.in, tc.want, result)
		}
	}
}

func TestSuggest_ToolNameTypo_Docker(t *testing.T) {
	t.Parallel()
	// "dokcer" is a transposition of "docker"; the tool name itself is fixed.
	e := New(0)
	result, found := e.Suggest("dokcer ps")
	if !found {
		t.Fatal("expected found=true for 'dokcer ps'")
	}
	if result != "docker ps" {
		t.Errorf("expected %q, got %q", "docker ps", result)
	}
}

func TestSuggest_ToolNameAndSubcommandTypo_Docker(t *testing.T) {
	t.Parallel()
	// Both the tool (dokcer) and the subcommand (pss) are wrong.
	e := New(0)
	result, found := e.Suggest("dokcer pss")
	if !found {
		t.Fatal("expected found=true for 'dokcer pss'")
	}
	if result != "docker ps" {
		t.Errorf("expected %q, got %q", "docker ps", result)
	}
}
