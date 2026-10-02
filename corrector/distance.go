package corrector

// The string metric, kept apart from what the engine does with it.
//
// This is self-contained arithmetic over two strings with no notion of
// commands, tools or thresholds. It is also the part most likely to be
// swapped for something else (a different edit distance, a weighting for
// keyboard adjacency); separating it means that change touches nothing
// else in the package.

// bestMatch returns the candidate most similar to token together with its
// similarity score in [0, 1]. It returns an empty string and 0 when no
// candidate has any similarity (or candidates is empty).
//
// On a tie the candidate closest in length to token wins, so the answer does not
// depend on the order of the list. A transposition or a substitution keeps the
// typed length while a dropped or doubled character changes it, so the
// length-preserving reading is the likelier typo: "imagse" is one edit from both
// "image" and "images" yet means the second.
func bestMatch(token string, candidates []string) (string, float64) {
	best := ""
	bestSim := 0.0
	for _, candidate := range candidates {
		sim := similarity(token, candidate)
		closer := sim == bestSim && lengthGap(token, candidate) < lengthGap(token, best)
		if sim > bestSim || (best != "" && closer) {
			bestSim = sim
			best = candidate
		}
	}
	return best, bestSim
}

// lengthGap is how far apart two strings are in byte length.
func lengthGap(a, b string) int {
	if len(a) > len(b) {
		return len(a) - len(b)
	}
	return len(b) - len(a)
}

// similarity returns a value in [0, 1] representing how alike a and b are,
// based on normalised Damerau-Levenshtein distance over byte length.
func similarity(a, b string) float64 {
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1.0
	}
	dist := damerauLevenshtein(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// damerauLevenshtein computes the optimal string alignment distance between two
// ASCII strings. It is the Levenshtein distance extended so that a transposition
// of two adjacent characters counts as a single edit, which matches common
// typing mistakes such as "psuh" for "push" or "gti" for "git".
func damerauLevenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	d := make([][]int, la+1)
	for i := 0; i <= la; i++ {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, min(d[i][j-1]+1, d[i-1][j-1]+cost))
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[la][lb]
}
