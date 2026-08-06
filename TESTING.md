# Testing Guide

Testing strategy, coverage requirements and how to run tests for CommandFixer.

See [DEVELOPMENT.md](DEVELOPMENT.md) for build setup.
See [ARCHITECTURE.md](ARCHITECTURE.md) for module design.

---

## Coverage Target

**A floor of 83%, enforced by `.\build.ps1 -Coverage`, which exits non-zero below it.**

The floor is the level the suite already holds, not an aspiration. A number
picked from ambition gets lowered the first time it blocks someone, which
teaches everyone that the gate is advisory. The current position:

| Package | Coverage |
|---------|----------|
| `corrector` | 100% |
| `config` | 92.1% |
| `logger` | 91.4% |
| `shell` | 87.7% |
| `main` | 65.3% |
| **total** | **83.5%** |

`corrector` is at 100% because it is pure computation over strings with nothing
to arrange. `main` is lowest because `cmdInstall` and `cmdUninstall` write to a
real user's PowerShell profile; the parts of them that are exercised are the
parts that can be pointed at a temporary file.

`main()` itself is excluded: it calls `os.Exit(1)`, which would terminate the
test process. It is a three-line wrapper around `run()`, which is tested.

Raise the floor when the suite earns it. Never lower it to make a run pass.

---

## Running Tests

### All tests

```powershell
go test ./...
```

### Single package

```powershell
go test ./config/...
go test ./corrector/...
go test ./shell/...
go test ./logger/...
go test .              # main package
```

### Verbose output

```powershell
go test -v ./...
```

### Single test

```powershell
go test -v -run TestSuggest_GitStatus_Typo ./corrector/
go test -v -run TestInstall ./shell/
```

### Race detector

```powershell
.\build.ps1 -Race
```

The Logger uses `sync.Mutex`. The race detector verifies no concurrent map or
slice access escapes the mutex.

**This needs a C toolchain and will not run without one.** Go's race detector
requires cgo and this project is deliberately CGO-free, so on a machine with no
gcc the command exits with `-race requires cgo` rather than running. That is an
environment limitation and not a fault in the suite: everything else, including
the coverage gate, runs without a C compiler.

---

## Coverage Measurement

### Generate and print summary

```powershell
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

The last line is the one the gate reads:

```
github.com/oernster/commandfixer/corrector/engine.go:Suggest           100.0%
github.com/oernster/commandfixer/main.go:cmdInstall                     36.0%
...
total:                                                                  83.5%
```

Note the quoting. Unquoted, PowerShell splits `-coverprofile=coverage.out` at
the dot and hands go `.out` as a package name, which fails and leaves a
truncated file called `coverage` behind. `build.ps1` quotes both flags.

### HTML report (clickable line-by-line)

```powershell
go tool cover -html=coverage.out -o coverage.html
Start-Process coverage.html   # opens in browser
```

Or with the build script:

```powershell
.\build.ps1 -Coverage
```

---

## Test Organisation

Each package has a co-located `_test.go` file in the **same package** (white-box testing). This gives direct access to unexported functions.

| File | Tests for |
|------|-----------|
| `config/loader_test.go` | `config` package |
| `corrector/engine_test.go` | Correction policy: what Suggest decides to do |
| `corrector/windows_test.go` | Correction over the Windows entries, both subcommand tools and standalone commands |
| `corrector/distance_test.go` | The string metric alone |
| `shell/powershell_test.go` | The hook snippet, the profile paths and the read-only operations |
| `shell/install_test.go` | The operations that change a user's profile |
| `shell/markers_test.go` | That the Go markers and paths still match `profile-hook.ps1` |
| `logger/stats_test.go` | `logger` package |
| `main_test.go` | CLI routing and the shared helpers |
| `commands_test.go` | The suggest, correct, log and stats commands |
| `install_test.go` | The two commands that write to a PowerShell profile |
| `structural_test.go` | Import boundaries and file size, over the repository itself |

The split is by concern rather than by file size, though size is what forced it:
four files had passed 400 lines with nothing measuring them. `structural_test.go`
now fails both above the cap and in the band just below it, so a file cannot be
shaved to 399 and break again on the next edit.

---

## Package-by-Package Strategy

### config

**Approach:** Use `t.TempDir()` for all file I/O. No mocking required.

**Branches covered:**

| Function | Branch |
|----------|--------|
| `Load` | Success, file not found (`os.IsNotExist`), invalid JSON |
| `LoadOrDefault` | Success, file not found (returns default), non-not-found error (directory as file path) |
| `Save` | Success, `MkdirAll` fails (regular file used as parent dir), `os.WriteFile` fails (directory at file path) |
| `applyDefaults` | `LogFile` empty (set default), `LogFile` non-empty (preserve), `MaxLogLines` zero (set 10000), `MaxLogLines` non-zero (preserve) |

**Known untestable branch:** `json.MarshalIndent` on a plain `Config` struct cannot return an error. The error check exists as defensive code. Coverage tools will flag it as covered because the function executes but the error path is unreachable in practice.

---

### corrector

**Approach:** Pure logic, no file I/O and no fixtures. Every test is a plain call
with strings in and strings out, which is the whole reason `structural_test.go`
forbids this package from importing anything that reaches outside the process.

**Branches covered:**

| Function | Branch |
|----------|--------|
| `New` | Zero threshold (default applied), negative, above one, valid, exactly one |
| `Suggest` | Empty input, single token, unknown tool, exact subcommand (no correction), too dissimilar, below a custom threshold |
| Subcommand correction | Typos across git, docker, kubectl and the trailing arguments preserved |
| Tool-name correction | Mistyped tool alone, mistyped tool plus mistyped subcommand |
| Command aliases | `gti` to `git`, unconditionally, with the subcommand then corrected |
| Windows subcommand tools | winget, choco, scoop, net, sc, reg, netsh |
| Windows standalone commands | dir, mkdir, copy, ipconfig, tasklist, arguments preserved, below threshold left alone |
| PowerShell aliases | `ls` never becomes `cls`; the alias set is never corrected |
| `similarity` | Equal strings, empty strings, wholly different, either side of the default threshold |
| `damerauLevenshtein` | Both empty, one empty, equal, single deletion, known distance, adjacent transposition |

---

### shell

**Approach:** `t.TempDir()` for all profile file operations. Unexported helpers (`removeSnippet`, `readProfileSafe`) tested directly.

**Branches covered:**

| Function | Branch |
|----------|--------|
| `ProfileSnippet` | Returns string with both markers and binary path |
| `Install` | Fresh profile (created from scratch), existing profile appended, existing profile without trailing newline, already installed (`ErrAlreadyInstalled`), parent dirs created |
| `Uninstall` | Snippet removed, existing content preserved, not installed (`ErrNotInstalled`), file not found (error) |
| `IsInstalled` | True (after install), false (no snippet), false (file missing - nil error) |
| `readProfileSafe` | File not found (returns `""`), file exists (returns content) |
| `removeSnippet` | No start marker (no-op), no end marker (truncate from start), snippet at content start, snippet at content end, empty before and after |

---

### logger

**Approach:** `t.TempDir()` for log file paths.

**Branches covered:**

| Function | Branch |
|----------|--------|
| `New` | Constructor correctness |
| `Log` | Single write, directory creation, multiple appended entries, timestamp range check |
| `ReadStats` | File not found (empty stats), empty file, valid entries (count + rule breakdown), malformed lines skipped |
| `splitLines` | Empty string, whitespace-only, normal lines, trailing newline |

---

### main

**Approach:** Test `dispatch()` directly with temp config files. `run()` covered by smoke test (uses real home dir to resolve config path, which is fine).

**Branches covered:**

| Function | Branch |
|----------|--------|
| `run` | Help command smoke (exercises `config.DefaultConfigPath` in real env) |
| `dispatch` | No args, `help`, `--help`, `-h`, `version`, `--version`, `-v`, `suggest`, `log`, unknown command |
| `cmdSuggest` | No args, known typo, exact command (no output), unknown tool (no output), multi-word input joined, bad config (error), missing config (default used) |
| `cmdCorrect` | No args (error), no match (unchanged), match with log write, multi-word input joined, missing config (LoadOrDefault default), bad JSON config (error) |
| `cmdLog` | No args (error), one arg (error), entry written, bad config (error), missing config (default used) |
| `cmdInstall` | Explicit profile path (success), already installed (error forwarded) |
| `cmdUninstall` | Explicit profile path (success), not installed (error forwarded) |
| `cmdStats` | Empty log (zero output), with entries (non-zero output), bad config (error) |
| `printUsage` | Smoke test (does not panic) |

**`main()` excluded:** calls `os.Exit(1)` which terminates the test process. The pattern `func main() { if err := run(...); err != nil { os.Exit(1) } }` is idiomatic Go and universally excluded from test coverage.

---

## Mocking Strategy

CommandFixer is designed to avoid mocking:

- **File I/O**: All file-dependent functions accept a `path string` parameter. Tests pass `t.TempDir()` paths. No mocking framework needed.
- **`os.Executable()`**: Returns the test binary path in test context. Fine for verifying profile content.
- **`os.UserHomeDir()`**: Called in `DefaultConfigDir/Path` and `DefaultProfilePath`. These are only tested for format (suffix/contains), not for exact value. No mocking needed.
- **Time**: `logger.Log` timestamps are checked via before/after bounds in tests, not exact values.
- **The correction engine**: nothing to mock. It takes a string and returns a string, so its tests are direct calls. `structural_test.go` keeps it that way.

---

## Test Fixtures

No external fixture files. All test data is defined inline:

```go
// Corrector tests need no config at all: the database is compiled in.
engine := corrector.New(0.6)
got, changed := engine.Suggest("git sattus")

// Inline JSON for config tests
content := `{"settings":{"similarity_threshold":0.6,"max_log_lines":10000}}`
os.WriteFile(path, []byte(content), 0644)

// Inline JSONL for logger tests
validJSON := `{"timestamp":"2024-01-01T00:00:00Z","original":"a","corrected":"b","rule":"r"}`
```

---

## CI Integration

Add this to your CI pipeline (e.g., GitHub Actions):

```yaml
- name: Test
  run: go test -race -coverprofile=coverage.out -covermode=atomic ./...

- name: Check coverage
  run: |
    go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | \
    grep -E '^(100\.0|9[5-9]\.[0-9])%$' || (echo "Coverage too low" && exit 1)
```

---

## Troubleshooting Tests

**Test leaves temp files:**
`t.TempDir()` is cleaned up automatically by the test runner. No manual cleanup needed.

**Race detector flags something:**
The `Logger` struct uses `sync.Mutex`. If you add new concurrent state, protect it with the existing mutex or a new one.

**Profile install test fails on CI (no home dir):**
`DefaultProfilePath()` and `DefaultConfigPath()` call `os.UserHomeDir()`. On headless CI, set `$HOME` before running tests:

```bash
HOME=/tmp go test ./...
```

**Test isolation:**
All tests call `t.Parallel()`. They do not share any global state. Each test gets its own `t.TempDir()`.
