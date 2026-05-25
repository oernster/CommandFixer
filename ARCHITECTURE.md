# Architecture

System design, module breakdown, and data flow for CommandFixer.

See [DEVELOPMENT.md](DEVELOPMENT.md) for build and local dev steps.
See [TESTING.md](TESTING.md) for testing strategy.

---

## Overview

CommandFixer is a single Go binary. When invoked by the PowerShell profile hook, it:

1. Reads the user's typo dictionary from `config.json`.
2. Applies correction rules to the command string.
3. Prints the (possibly corrected) command to stdout.
4. Logs the correction to a JSONL file if a change was made.

The PowerShell profile hook reads the printed output and replaces the input buffer before execution.

---

## Data Flow

```
User types command + Enter
         |
         v
 PSReadLine intercepts Enter key
         |
         v
 Calls: commandfixer.exe correct "<buffer>"
         |
         v
 +--------------------------+
 |  config.LoadOrDefault()  |  reads config.json (falls back to defaults if missing)
 +--------------------------+
         |
         v
 +--------------------------+
 |  corrector.New(cfg)      |  compiles rules (pre-compiles regex patterns)
 +--------------------------+
         |
         v
 +--------------------------+
 |  engine.Correct(cmd)     |  applies rules in order, returns Result
 +--------------------------+
         |         |
         |         v (if Changed)
         |  logger.Log(original, corrected, rule)  -->  corrections.log
         |
         v
 Prints result.Corrected to stdout (exit 0)
         |
         v
 PSReadLine reads stdout
         |
    changed?
    /       \
  yes        no
   |          |
   v          v
 Shows       Accepts
 "Corrected: original command
 X -> Y"     directly
   |
   v
 Replaces buffer with corrected command
   |
   v
 Executes corrected command
```

---

## Module Breakdown

### `main` (main.go)

Entry point and CLI dispatcher.

**Responsibilities:**
- Parse `os.Args`
- Resolve the default config path via `config.DefaultConfigPath()`
- Route to the correct command handler: `correct`, `install`, `uninstall`, `stats`, `version`, `help`
- Print usage

**Key design decision:** `dispatch(args, cfgPath)` is separate from `main()` so tests can inject a temporary config path without touching the real filesystem.

**Exported API:** none (package main)

**Internal functions:**

| Function | Role |
|----------|------|
| `run(args)` | Resolves default config path, calls `dispatch` |
| `dispatch(args, cfgPath)` | Routes commands; injectable for tests |
| `cmdCorrect(args, cfgPath)` | Load config, correct, log |
| `cmdInstall(args)` | Write PS profile hook |
| `cmdUninstall(args)` | Remove PS profile hook |
| `cmdStats(cfgPath)` | Aggregate and print log stats |
| `printUsage()` | Print help text |

---

### `config` (config/loader.go)

Loads, validates, and saves the JSON typo dictionary.

**Data structures:**

```go
type TypoEntry struct {
    From  string // literal string or regex pattern to match
    To    string // replacement string
    Regex bool   // if true, From is a regexp.Compile pattern
}

type Settings struct {
    LogFile         string // path to JSONL corrections log
    ShowCorrections bool   // whether the PS hook announces changes
    MaxLogLines     int    // soft cap for log size (future rotation)
}

type Config struct {
    Typos    []TypoEntry
    Settings Settings
}
```

**Key design decisions:**

- `LoadOrDefault` returns a zero-typo config (not an error) when the file is missing. This lets the binary work on a fresh machine without the user creating config first.
- `applyDefaults()` is called unconditionally after every load, so partial configs (missing `log_file`, etc.) always have safe values.
- `Save` creates parent directories: the user never needs to `mkdir` manually.

**Exported functions:**

| Function | Description |
|----------|-------------|
| `DefaultConfigDir()` | `$HOME/.typo-fixer` |
| `DefaultConfigPath()` | `$HOME/.typo-fixer/config.json` |
| `Load(path)` | Read and unmarshal; error if missing or invalid JSON |
| `LoadOrDefault(path)` | Like Load but returns defaults if file is absent |
| `Save(path, cfg)` | Marshal to indented JSON and write |

---

### `corrector` (corrector/engine.go)

Compiles typo rules and applies them to command strings.

**Data structures:**

```go
type Result struct {
    Original  string // command before any correction
    Corrected string // command after all rules applied
    Changed   bool   // true if at least one rule fired
    RuleFrom  string // last rule that fired (From field)
    RuleTo    string // last rule that fired (To field)
}

type Engine struct {
    rules []compiledRule // pre-compiled rules
}

type compiledRule struct {
    entry config.TypoEntry
    regex *regexp.Regexp // non-nil for regex rules only
}
```

**Key design decisions:**

- **All rules apply in sequence.** Each rule's output feeds the next. Multiple typos in one command are corrected in one pass.
- **Literal rules use `strings.ReplaceAll`**, not equality. So `"git sattus"` in the dictionary corrects `"git sattus -v"` (substring match).
- **Regex rules use `regexp.ReplaceAllString`**, supporting capture groups (`$1`, etc.).
- **Regex compiled eagerly** in `New()`. Invalid patterns fail fast at startup rather than silently at correction time.
- **Result records the last firing rule** (not all rules). This is sufficient for log attribution; full rule chain logging is a future enhancement.

**Exported functions:**

| Function | Description |
|----------|-------------|
| `New(cfg)` | Compile all rules; error if any regex is invalid |
| `engine.Correct(cmd)` | Apply rules, return Result |

---

### `shell` (shell/powershell.go)

Generates and manages the PowerShell profile hook.

**Key design decisions:**

- The hook uses `Set-PSReadLineKeyHandler -Key Enter`. This is the standard PSReadLine API for intercepting keystrokes. It requires PowerShell 7 with PSReadLine 2.x (shipped by default).
- The snippet is delimited by exact start/end marker strings. This makes install idempotent (detects existing hook) and makes uninstall reliable (removes the exact block).
- `readProfileSafe` treats `os.IsNotExist` as an empty profile. Users who have never set up a PS profile are handled without error.
- `removeSnippet` handles edge cases: snippet at start (no content before it), snippet at end, missing end marker (truncates from start marker).

**Exported functions:**

| Function | Description |
|----------|-------------|
| `ProfileSnippet(binaryPath)` | Generate the PS block to inject |
| `DefaultProfilePath()` | `$HOME/Documents/PowerShell/profile.ps1` |
| `Install(profilePath, binaryPath)` | Append hook; ErrAlreadyInstalled if present |
| `Uninstall(profilePath)` | Remove hook; ErrNotInstalled if absent |
| `IsInstalled(profilePath)` | Check without modifying |

**Sentinel errors:**

| Error | When |
|-------|------|
| `ErrAlreadyInstalled` | Returned by `Install` if hook already present |
| `ErrNotInstalled` | Returned by `Uninstall` if hook not found |

---

### `logger` (logger/stats.go)

Writes correction events to a JSONL log and aggregates statistics.

**Data structures:**

```go
type CorrectionEntry struct {
    Timestamp time.Time // UTC
    Original  string
    Corrected string
    Rule      string    // "from -> to" label
}

type Stats struct {
    TotalCorrections int
    RuleCounts       map[string]int  // rule label -> count
    History          []CorrectionEntry
}

type Logger struct {
    mu      sync.Mutex
    logPath string
}
```

**Key design decisions:**

- **JSONL format** (one JSON object per line). Each entry is self-contained; the file can be parsed line by line without loading the whole thing. Tolerant of partial writes (malformed lines are skipped in `ReadStats`).
- **Append-only writes** via `os.O_APPEND`. No seek, no overwrite - safe for concurrent invocations (multiple PS windows).
- **`sync.Mutex`** inside Logger for safe concurrent use within one process.
- **`ReadStats` returns empty stats (not error) for missing file.** First run before any correction has occurred should not fail.

---

## External Dependencies

None. CommandFixer uses only the Go standard library:

| Package | Used for |
|---------|---------|
| `encoding/json` | Config file and log serialisation |
| `fmt` | Error formatting and output |
| `os` | File I/O, executable path, home directory |
| `path/filepath` | Cross-platform path construction |
| `regexp` | Regex rule compilation and matching |
| `strings` | Literal rule matching via `strings.ReplaceAll` |
| `sync` | Logger mutex |
| `time` | Log timestamps |
| `errors` | Sentinel error values |

---

## Config File Location

| Platform | Default path |
|----------|-------------|
| Windows | `%USERPROFILE%\.typo-fixer\config.json` |
| Linux/macOS | `$HOME/.typo-fixer/config.json` |

The binary resolves this via `os.UserHomeDir()` at runtime, so the exact path varies per user.

---

## PowerShell Hook Mechanics

The installed snippet:

```powershell
# CommandFixer Integration - DO NOT EDIT
Set-PSReadLineKeyHandler -Key Enter -ScriptBlock {
    $line = $null; $cursor = $null
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    if ($line.Trim() -ne '') {
        $corrected = & 'C:\path\to\commandfixer.exe' correct $line 2>$null
        if ($LASTEXITCODE -eq 0 -and $corrected -and $corrected -ne $line) {
            Write-Host "CommandFixer: '$line' -> '$corrected'" -ForegroundColor Yellow
            [Microsoft.PowerShell.PSConsoleReadLine]::RevertLine()
            [Microsoft.PowerShell.PSConsoleReadLine]::Insert($corrected)
        }
    }
    [Microsoft.PowerShell.PSConsoleReadLine]::AcceptLine()
}
# End CommandFixer Integration
```

**Flow:**
1. `GetBufferState` extracts the current input line.
2. Binary is invoked with `correct <line>`. stderr is suppressed (`2>$null`) to avoid noise on config errors.
3. `$LASTEXITCODE -eq 0` guards against binary crashes silently.
4. `RevertLine` + `Insert` replaces the buffer atomically.
5. `AcceptLine` submits the (possibly replaced) line for execution.

**Failure mode:** if the binary fails (bad config, missing binary), the original command runs unchanged. CommandFixer failures are never user-visible beyond a missing correction.

---

## Extending the Architecture

### Adding a service mode (HTTP)

Add a `service` command to `dispatch()`. Use `net/http` with a `/correct` endpoint that accepts a JSON body `{"cmd": "..."}` and returns `{"corrected": "..."}`. The PS hook would then `curl` or `Invoke-RestMethod` the local service. This avoids binary startup cost on each keystroke.

### Adding case-insensitive matching

Add `CaseInsensitive bool` to `TypoEntry`. In `corrector.New`, wrap literal rules with `(?i)` regex or use `strings.EqualFold` + manual offset matching.

### Adding log rotation

Read existing entries count in `Logger.Log()`. If it exceeds `MaxLogLines`, truncate the oldest half. Requires a read-truncate-write cycle with the lock held.
