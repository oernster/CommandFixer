package corrector

import "testing"

// Tests for the string metric, matching distance.go.
//
// These know nothing about commands or tools: they are arithmetic over two
// strings, which is the whole reason the metric is its own file.

// ---------------------------------------------------------------------------
// damerauLevenshtein
// ---------------------------------------------------------------------------

func TestDamerauLevenshtein_EmptyStrings(t *testing.T) {
	t.Parallel()
	if got := damerauLevenshtein("", ""); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestDamerauLevenshtein_EmptyA(t *testing.T) {
	t.Parallel()
	if got := damerauLevenshtein("", "abc"); got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestDamerauLevenshtein_EmptyB(t *testing.T) {
	t.Parallel()
	if got := damerauLevenshtein("abc", ""); got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestDamerauLevenshtein_EqualStrings(t *testing.T) {
	t.Parallel()
	if got := damerauLevenshtein("status", "status"); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestDamerauLevenshtein_SingleDeletion(t *testing.T) {
	t.Parallel()
	// "pss" -> "ps": delete one char.
	if got := damerauLevenshtein("pss", "ps"); got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestDamerauLevenshtein_KnownDistance(t *testing.T) {
	t.Parallel()
	// "comit" -> "commit": insert one 'm'.
	if got := damerauLevenshtein("comit", "commit"); got != 1 {
		t.Errorf("expected distance 1 for comit/commit, got %d", got)
	}
}

func TestDamerauLevenshtein_AdjacentTransposition(t *testing.T) {
	t.Parallel()
	// A single adjacent swap counts as one edit, not two.
	for _, tc := range []struct {
		a, b string
	}{
		{"ba", "ab"},
		{"gti", "git"},
		{"psuh", "push"},
	} {
		if got := damerauLevenshtein(tc.a, tc.b); got != 1 {
			t.Errorf("expected distance 1 for %q/%q transposition, got %d", tc.a, tc.b, got)
		}
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
	// "sattus" vs "status": a single adjacent transposition, distance=1,
	// max=6, sim~0.833.
	s := similarity("sattus", "status")
	if s <= defaultThreshold {
		t.Errorf("expected similarity > %v for sattus/status, got %v", defaultThreshold, s)
	}
}
