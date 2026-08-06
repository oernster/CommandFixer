# Architecture

System design, module breakdown and data flow for CommandFixer.

See [DEVELOPMENT.md](DEVELOPMENT.md) for build and local dev steps.
See [TESTING.md](TESTING.md) for testing strategy.

---

## Overview

CommandFixer is a single Go binary. When invoked by the PowerShell profile hook, it:

1. Reads its settings (the similarity threshold and the log path) from `config.json`.
2. Fuzzy-matches the command against a built-in database of CLI tools, their
   subcommands and the standard Windows commands.
3. Prints the corrected command to stdout when one is close enough.
4. Logs the correction to a JSONL file if the user accepted a change.

There is no typo dictionary to maintain. The database is compiled in, which is
what makes the tool useful before any configuration exists.

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
 Calls: commandfixer.exe suggest "<buffer>"
         |
         v
 +--------------------------+
 |  config.LoadOrDefault()  |  reads config.json (falls back to defaults if missing)
 +--------------------------+
         |
         v
 +----------------------------------+
 |  corrector.New(cfg.Settings      |  builds an engine at the configured
 |    .SimilarityThreshold)         |  similarity threshold
 +----------------------------------+
         |
         v
 +----------------------------------+
 |  engine.Suggest(cmd)             |  alias, then subcommand, then tool name;
 |                                  |  returns the command and whether it moved
 +----------------------------------+
         |
         v
 Prints the suggestion to stdout or nothing at all (exit 0)
         |
         v
 PSReadLine reads stdout
         |
   a suggestion?
    /          \
  yes           no
   |             |
   v             v
 Shows         Accepts the
 "did you      original command
  mean: X"     directly
   |
   v
 Waits for Y or n
   |
   v (only on Y)
 Replaces the buffer, then calls
 commandfixer.exe log "<from>" "<to>"  -->  corrections.log
   |
   v
 Executes the corrected command
```

---

## Module Breakdown

### `main` (main.go)

Entry point and CLI dispatcher.

**Responsibilities:**
- Parse `os.Args`
- Resolve the default config path via `config.DefaultConfigPath()`
- Route to the correct command handler: `suggest`, `correct`, `log`, `install`, `uninstall`, `stats`, `version`, `help`
- Print usage

**Key design decision:** `dispatch(args, cfgPath)` is separate from `main()` so tests can inject a temporary config path without touching the real filesystem.

**Exported API:** none (package main)

**Internal functions:**

| Function | Role |
|----------|------|
| `run(args)` | Resolves default config path, calls `dispatch` |
| `dispatch(args, cfgPath)` | Routes commands; injectable for tests |
| `cmdSuggest(args, cfgPath)` | The machine-facing command the prompt hook calls; prints a suggestion or nothing |
| `cmdCorrect(args, cfgPath)` | Load config, correct, log |
| `cmdLog(args, cfgPath)` | Record an accepted correction |
| `cmdInstall(args)` | Write PS profile hook |
| `cmdUninstall(args)` | Remove PS profile hook |
| `cmdStats(cfgPath)` | Aggregate and print log stats |
| `printUsage()` | Print help text |

---

### `config` (config/loader.go)

Loads, validates and saves the JSON settings file.

**Data structures:**

```go
type Settings struct {
    LogFile             string  // path to the JSONL corrections log
    MaxLogLines         int     // soft cap for log size (rotation not implemented)
    SimilarityThreshold float64 // (0.0, 1.0]; how close a match must be
}

type Config struct {
    Settings Settings
}
```

**Key design decisions:**

- **Settings only, no dictionary.** The command database is compiled in, so config exists to tune behaviour rather than to supply it. This is the change that made the tool useful on first run: there is nothing a user must write before it does anything.
- `LoadOrDefault` returns defaults (not an error) when the file is missing. This lets the binary work on a fresh machine without the user creating config first.
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

### `corrector` (corrector/engine.go, database.go, distance.go)

Fuzzy-matches a typed command against a built-in database of tools and their
subcommands and returns the corrected form when one is close enough.

Three files along one seam each, so that adding a tool never touches the
matching logic and changing the metric never touches either:

| File | Holds |
|------|-------|
| `database.go` | The data: known tools and their subcommands, the Windows standalone commands, the habitual-typo aliases |
| `engine.go` | The policy: what to correct, in what order and when to leave a command alone |
| `distance.go` | The metric: Damerau-Levenshtein distance and the similarity score derived from it |

**Data structures:**

```go
type Engine struct {
    threshold float64 // minimum similarity for a correction to apply
}
```

**Key design decisions:**

- **No user dictionary is required.** Correction comes from the built-in database rather than from rules a user has to write, which is what makes the tool useful on first run.
- **Correction is tried in a fixed order**: an unconditional alias (`gti` to `git`), then subcommand correction when the first token is a known tool, then tool-name correction against both the known tools and the Windows standalone commands, with the closer of the two winning and ties going to the CLI tool.
- **An exact match is never "corrected".** A token that already appears in the database is left alone, which is why the PowerShell POSIX-style aliases are listed explicitly: `ls` is one insertion from `cls` and would otherwise be rewritten to it.
- **Damerau-Levenshtein rather than plain Levenshtein**, so a transposition counts as one edit. Typing mistakes are mostly transpositions (`psuh`, `gti`); plain Levenshtein scores those as two.
- **The engine is pure computation over strings**: no filesystem, no environment, no clock. That is why its tests are a plain table with no fixture; `structural_test.go` enforces it rather than trusting it.

**Exported functions:**

| Function | Description |
|----------|-------------|
| `New(threshold)` | Build an engine; a zero or out-of-range threshold applies the default (0.6) |
| `engine.Threshold()` | The similarity threshold in use |
| `engine.Suggest(cmd)` | The corrected command and whether anything changed |

---

### `shell` (shell/powershell.go)

Generates and manages the PowerShell profile hook.

**Key design decisions:**

- The hook uses `Set-PSReadLineKeyHandler -Key Enter`. This is the standard PSReadLine API for intercepting keystrokes. It requires PowerShell 7 with PSReadLine 2.x (shipped by default).
- The snippet is delimited by exact start/end marker strings. This makes install idempotent (detects existing hook) and makes uninstall reliable (removes the exact block).
- Those markers exist in two languages and have to. The binary writes and removes the block but `uninstall.ps1` must still work when the binary is already gone, so it carries a fallback that strips the block itself. The scripts define their copy once in `profile-hook.ps1`; `shell/markers_test.go` reads that file and fails if the Go constants drift from it. A marker changed on one side only would leave a hook line nothing can find to remove, running on every prompt a user types.
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
| `strings` | Tokenising a command and rejoining it |
| `sort` | Deterministic ordering of the known tool names |
| `sync` | Logger mutex |
| `time` | Log timestamps |
| `errors` | Sentinel error values |

Test-only and not linked into the binary: `go/parser` and `go/token` for the
import-boundary scan, `bufio` and `io/fs` for the file-size scan; `regexp`
for reading a value out of `profile-hook.ps1`.

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

Fold both sides in `similarity` before measuring, in `distance.go`, so nothing
else in the package has to know. Note that this changes what "exact match" means
in `suggestToolName` and `correctSubcommand`, which currently compare with `==`:
those checks are what stop a valid command being rewritten, so they would need
folding in the same change or the two would disagree.

### Adding log rotation

Read existing entries count in `Logger.Log()`. If it exceeds `MaxLogLines`, truncate the oldest half. Requires a read-truncate-write cycle with the lock held.
