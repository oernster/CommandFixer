// Package corrector applies typo-correction rules to shell command strings.
package corrector

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/oernster/commandfixer/config"
)

// Result holds the outcome of a correction attempt.
type Result struct {
	// Original is the command before any correction.
	Original string
	// Corrected is the command after all rules have been applied.
	Corrected string
	// Changed is true when at least one rule altered the command.
	Changed bool
	// RuleFrom and RuleTo record the last rule that fired.
	RuleFrom string
	RuleTo   string
}

// Engine applies an ordered list of typo-correction rules to shell commands.
type Engine struct {
	rules []compiledRule
}

type compiledRule struct {
	entry config.TypoEntry
	// regex is non-nil only when entry.Regex is true.
	regex *regexp.Regexp
}

// New builds an Engine from cfg.
// Returns an error if any regex pattern in cfg.Typos fails to compile.
func New(cfg *config.Config) (*Engine, error) {
	e := &Engine{}
	for _, entry := range cfg.Typos {
		rule := compiledRule{entry: entry}
		if entry.Regex {
			rx, err := regexp.Compile(entry.From)
			if err != nil {
				return nil, fmt.Errorf("compile regex pattern %q: %w", entry.From, err)
			}
			rule.regex = rx
		}
		e.rules = append(e.rules, rule)
	}
	return e, nil
}

// Correct applies all rules to cmd in declaration order.
// Each rule's output feeds the next, so corrections compose.
// The returned Result records the original command, the final corrected form,
// whether any change occurred, and which rule last fired.
func (e *Engine) Correct(cmd string) Result {
	result := Result{Original: cmd, Corrected: cmd}
	for _, rule := range e.rules {
		var next string
		if rule.regex != nil {
			next = rule.regex.ReplaceAllString(result.Corrected, rule.entry.To)
		} else {
			next = strings.ReplaceAll(result.Corrected, rule.entry.From, rule.entry.To)
		}
		if next != result.Corrected {
			result.Corrected = next
			result.Changed = true
			result.RuleFrom = rule.entry.From
			result.RuleTo = rule.entry.To
		}
	}
	return result
}
