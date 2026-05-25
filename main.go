// Command commandfixer is a PowerShell typo auto-corrector.
// It loads a user-defined correction dictionary and fixes common typing mistakes
// before commands execute.
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

const appVersion = "1.0.0"

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
	case "correct":
		return cmdCorrect(args[1:], cfgPath)
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

// cmdCorrect loads config, corrects the provided command, prints the result,
// and logs the correction if one occurred.
func cmdCorrect(args []string, cfgPath string) error {
	if len(args) == 0 {
		return fmt.Errorf("correct: provide a command to check")
	}
	input := strings.Join(args, " ")

	cfg, err := config.LoadOrDefault(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	eng, err := corrector.New(cfg)
	if err != nil {
		return fmt.Errorf("build corrector: %w", err)
	}

	result := eng.Correct(input)
	fmt.Println(result.Corrected)

	if result.Changed {
		log := logger.New(cfg.Settings.LogFile)
		rule := result.RuleFrom + " -> " + result.RuleTo
		if logErr := log.Log(result.Original, result.Corrected, rule); logErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write log: %v\n", logErr)
		}
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

func printUsage() {
	fmt.Print(`commandfixer - PowerShell typo auto-corrector

Commands:
  correct <cmd>        Check and print the corrected form of <cmd>
  install [profile]    Add the PSReadLine hook to your PowerShell profile
  uninstall [profile]  Remove the hook from your PowerShell profile
  stats                Show correction statistics from the log
  version              Print version
  help                 Show this help

Config file:  %USERPROFILE%\.typo-fixer\config.json
Log file:     %USERPROFILE%\.typo-fixer\corrections.log
`)
}
