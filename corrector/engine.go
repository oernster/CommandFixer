// Package corrector applies fuzzy-matching typo correction to shell commands.
//
// It maintains a built-in database of popular CLI tools and their valid
// subcommands. When a typed command looks like a typo of a known subcommand
// (within a configurable similarity threshold), Suggest returns the corrected
// form. No user-maintained dictionary is required.
//
// For Windows standalone commands (dir, cd, copy, etc.), Suggest also
// fuzzy-matches the first token against a known standalone command list
// and corrects the command name itself when a close-enough match is found.
//
// The package is three files along one seam each: database.go is the data the
// engine matches against, distance.go is the string metric, and this file is
// the correction policy that decides what to do with them.
package corrector

import "strings"

// defaultThreshold is used when New receives a zero or out-of-range threshold.
const defaultThreshold = 0.6

// Engine performs fuzzy subcommand correction for known CLI tools.
type Engine struct {
	threshold float64
}

// New creates an Engine with the given similarity threshold (0.0, 1.0].
// A zero or out-of-range value applies the package default (0.6).
func New(threshold float64) *Engine {
	if threshold <= 0 || threshold > 1 {
		threshold = defaultThreshold
	}
	return &Engine{threshold: threshold}
}

// Threshold returns the similarity threshold in use.
func (e *Engine) Threshold() float64 {
	return e.threshold
}

// Suggest checks whether cmd contains a recognisable typo and returns the
// corrected form and true when a similar-enough match is found. It returns
// cmd unchanged and false when no correction can be made.
//
// Correction is applied in this order:
//
//  1. Command-name alias: when the first token is a known habitual typo
//     (commandAliases, for example "gti"), it is replaced unconditionally and
//     the subcommand is then corrected against the intended tool.
//
//  2. Subcommand correction: when the first token is an exact commandDB key,
//     the second token is fuzzy-matched against the tool's known subcommands.
//
//  3. Tool-name correction: when the first token matches neither of the above,
//     it is fuzzy-matched against the known CLI tools and the Windows standalone
//     commands; the closer of the two wins. A corrected CLI tool also has its
//     subcommand corrected.
//
// In every mode at least two tokens must be present, and a fuzzy match must
// meet or exceed the configured threshold. Tokens beyond the corrected ones are
// preserved verbatim.
func (e *Engine) Suggest(cmd string) (string, bool) {
	tokens := strings.Fields(cmd)
	if len(tokens) < 2 {
		return cmd, false
	}

	if canonical, aliased := commandAliases[tokens[0]]; aliased {
		tokens[0] = canonical
		corrected, _ := e.correctSubcommand(tokens)
		return corrected, true
	}

	if _, known := commandDB[tokens[0]]; known {
		return e.correctSubcommand(tokens)
	}

	return e.suggestToolName(cmd, tokens)
}

// correctSubcommand fuzzy-corrects tokens[1] against the known subcommands of
// the tool named in tokens[0], which must be a commandDB key. It returns the
// joined command and whether the subcommand was changed. An already-valid or
// too-dissimilar subcommand is left untouched.
func (e *Engine) correctSubcommand(tokens []string) (string, bool) {
	subcommands := commandDB[tokens[0]]
	subcommand := tokens[1]

	// Already a valid subcommand: nothing to correct.
	for _, sc := range subcommands {
		if sc == subcommand {
			return strings.Join(tokens, " "), false
		}
	}

	match, sim := bestMatch(subcommand, subcommands)
	if sim < e.threshold || match == "" {
		return strings.Join(tokens, " "), false
	}
	tokens[1] = match
	return strings.Join(tokens, " "), true
}

// suggestToolName attempts to correct a mistyped first token. It fuzzy-matches
// the token against both the known CLI tools (commandDB keys) and the Windows
// standalone commands, and applies the closer match. A corrected CLI tool also
// has its subcommand corrected; a corrected standalone command keeps all of its
// remaining arguments verbatim.
func (e *Engine) suggestToolName(cmd string, tokens []string) (string, bool) {
	tool := tokens[0]

	// Already an exact known standalone command: no correction needed.
	for _, sc := range windowsCommands {
		if sc == tool {
			return cmd, false
		}
	}

	toolMatch, toolSim := bestMatch(tool, commandDBTools)
	winMatch, winSim := bestMatch(tool, windowsCommands)

	// Prefer a CLI-tool correction on a tie, then also fix its subcommand.
	if toolSim >= winSim {
		if toolSim < e.threshold || toolMatch == "" {
			return cmd, false
		}
		tokens[0] = toolMatch
		corrected, _ := e.correctSubcommand(tokens)
		return corrected, true
	}

	if winSim < e.threshold || winMatch == "" {
		return cmd, false
	}
	tokens[0] = winMatch
	return strings.Join(tokens, " "), true
}
