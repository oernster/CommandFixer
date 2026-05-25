// Command commandfixer is a PowerShell typo auto-corrector.
// It intercepts shell commands, fuzzy-matches subcommands against a built-in
// database of popular CLI tools, and prompts for confirmation before correcting.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/oernster/commandfixer/config"
	"github.com/oernster/commandfixer/corrector"
	"github.com/oernster/commandfixer/logger"
	"github.com/oernster/commandfixer/shell"
)

const appVersion = "2.0.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run resolves the default config path and dispatches to the appropriate command.
// It is the testable entry point for the main package.
func run(args []string) error {
	cfgPath, err := config.DefaultConfigPath()
	if err != nil {
		return err
	}
	return dispatch(args, cfgPath)
}

// dispatch routes CLI args to the correct command handler.
// cfgPath is the resolved path to config.json (injected for testability).
func dispatch(args []string, cfgPath string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "suggest":
		return cmdSuggest(args[1:], cfgPath)
	case "correct":
		return cmdCorrect(args[1:], cfgPath)
	case "log":
		return cmdLog(args[1:], cfgPath)
	case "install":
		return cmdInstall(args[1:])
	case "uninstall":
		return cmdUninstall(args[1:])
	case "stats":
		return cmdStats(cfgPath)
	case "version", "--version", "-v":
		fmt.Printf("commandfixer v%s\n", appVersion)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (run 'commandfixer help')", args[0])
	}
}

// cmdSuggest is the machine-facing command used by the PSReadLine hook.
// It fuzzy-matches the input command and prints the corrected form to stdout
// if a suggestion is found. Prints nothing when no correction is needed.
func cmdSuggest(args []string, cfgPath string) error {
	if len(args) == 0 {
		return fmt.Errorf("suggest: provide a command to check")
	}
	input := strings.Join(args, " ")

	cfg, err := config.LoadOrDefault(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	eng := corrector.New(cfg.Settings.SimilarityThreshold)
	suggestion, found := eng.Suggest(input)
	if found {
		fmt.Println(suggestion)
	}
	return nil
}

// cmdCorrect is the human-facing diagnostic command.
// It prints the corrected form of the input command (or the input unchanged
// when no correction is found) and logs the correction if one occurred.
func cmdCorrect(args []string, cfgPath string) error {
	if len(args) == 0 {
		return fmt.Errorf("correct: provide a command to check")
	}
	input := strings.Join(args, " ")

	cfg, err := config.LoadOrDefault(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	eng := corrector.New(cfg.Settings.SimilarityThreshold)
	suggestion, found := eng.Suggest(input)
	if found {
		fmt.Println(suggestion)
		log := logger.New(cfg.Settings.LogFile)
		if logErr := log.Log(input, suggestion, "auto-fuzzy"); logErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write log: %v\n", logErr)
		}
	} else {
		fmt.Println(input)
	}
	return nil
}

// cmdLog records a correction event to the log file.
// Called by the PSReadLine hook when the user confirms a suggestion.
// args[0] = original command, args[1] = corrected command.
func cmdLog(args []string, cfgPath string) error {
	if len(args) < 2 {
		return fmt.Errorf("log: provide original and corrected commands")
	}
	original := args[0]
	corrected := args[1]

	cfg, err := config.LoadOrDefault(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	l := logger.New(cfg.Settings.LogFile)
	if err := l.Log(original, corrected, "auto-fuzzy"); err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	return nil
}

// cmdInstall adds the CommandFixer hook to the PowerShell profile.
// An optional first argument overrides the profile path (used in tests).
func cmdInstall(args []string) error {
	profilePath, err := shell.DefaultProfilePath()
	if err != nil {
		return err
	}
	if len(args) > 0 {
		profilePath = args[0]
	}
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	if err := shell.Install(profilePath, binaryPath); err != nil {
		return err
	}
	fmt.Printf("Installed CommandFixer into: %s\n", profilePath)
	fmt.Println("Restart PowerShell to activate.")
	return nil
}

// cmdUninstall removes the CommandFixer hook from the PowerShell profile.
// An optional first argument overrides the profile path (used in tests).
func cmdUninstall(args []string) error {
	profilePath, err := shell.DefaultProfilePath()
	if err != nil {
		return err
	}
	if len(args) > 0 {
		profilePath = args[0]
	}
	if err := shell.Uninstall(profilePath); err != nil {
		return err
	}
	fmt.Printf("Removed CommandFixer from: %s\n", profilePath)
	return nil
}

// cmdStats loads the log file and prints correction counts.
func cmdStats(cfgPath string) error {
	cfg, err := config.LoadOrDefault(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	stats, err := logger.ReadStats(cfg.Settings.LogFile)
	if err != nil {
		return fmt.Errorf("read stats: %w", err)
	}
	fmt.Printf("Total corrections: %d\n", stats.TotalCorrections)
	if len(stats.RuleCounts) > 0 {
		fmt.Println("By rule:")
		for rule, count := range stats.RuleCounts {
			fmt.Printf("  %s: %d\n", rule, count)
		}
	}
	return nil
}

// usageText is defined as a constant to avoid fmt.Print seeing %USERPROFILE%
// as a formatting directive (go vet false-positive on the %U sequence).
const usageText = `commandfixer - PowerShell typo auto-corrector

Commands:
  suggest <cmd>        Fuzzy-match <cmd> and print the corrected form (used by hook)
  correct <cmd>        Like suggest but also logs the correction
  log <orig> <fixed>   Record a confirmed correction to the log file
  install [profile]    Add the PSReadLine hook to your PowerShell profile
  uninstall [profile]  Remove the hook from your PowerShell profile
  stats                Show correction statistics from the log
  version              Print version
  help                 Show this help

Config file:  %USERPROFILE%\.typo-fixer\config.json
Log file:     %USERPROFILE%\.typo-fixer\corrections.log
`

func printUsage() {
	os.Stdout.WriteString(usageText)
}
