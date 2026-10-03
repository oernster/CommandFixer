# Testing Guide

Testing strategy, coverage requirements and how to run tests for CommandFixer.

See [DEVELOPMENT.md](DEVELOPMENT.md) for build setup.
See [ARCHITECTURE.md](ARCHITECTURE.md) for module design.

---

## Coverage Target

**A floor of 83%, enforced by `.\build.ps1 -Coverage`, which exits non-zero below it.**

The floor is the level the suite already holds, not an aspiration. A number
picked from ambition gets lowered the first time it blocks someone, which
teaches everyone that the gate is advisory. The per-package figures below are a
snapshot from one run, not a gate; only the total floor above is enforced:

| Package | Coverage |
|---------|----------|
| `corrector` | 100% |
| `config` | 92.1% |
| `logger` | 91.4% |
| `shell` | 89.5% |
| `main` | 65.5% |
| **total** | **84.7%** |

`corrector` reached 100% in that run because it is pure computation over strings
with nothing to arrange. `main` is lowest because `cmdInstall` and `cmdUninstall` write to a
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

### On the race detector

There is none, deliberately. Go's race detector requires cgo, this project is
deliberately CGO-free and the Windows machines it is developed on have no C
toolchain, so `go test -race` can only exit with `-race requires cgo`. A command
that cannot run is worse than an absent one: it reads as a check being
performed.

`Logger` still guards its state with a `sync.Mutex` and that is still the rule
to follow when adding concurrent state. It is held by review rather than by the
detector.

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
total:                                                                  84.7%
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
| `corrector/lookup_test.go` | That a first token the command lookup knows is never renamed |
| `corrector/distance_test.go` | The string metric alone |
| `shell/powershell_test.go` | The hook snippet, the profile paths and the read-only operations |
| `shell/documents_test.go` | Where the profiles go, with the Documents folder handed in, a moved one included |
| `shell/documents_windows_test.go` | The real Documents known-folder lookup (Windows only) |
| `shell/install_test.go` | The operations that change a user's profile |
| `shell/markers_test.go` | That the Go markers and paths still match `profile-hook.ps1` |
| `shell/architecture_test.go` | That the snippet shown in `ARCHITECTURE.md` is exactly what `ProfileSnippet` generates |
| `logger/stats_test.go` | `logger` package |
| `main_test.go` | CLI routing and the shared helpers |
| `commands_test.go` | The suggest, correct, log and stats commands |
| `install_test.go` | The two commands that write to a PowerShell profile |
| `path_windows_test.go` | The PATH lookup against a real directory holding a `code.cmd` shim (Windows only) |
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
| `applyDefaults` | `LogFile` empty (set default), `LogFile` non-empty (preserve), `MaxLogLines` zero (set 10000), `MaxLogLines` non-zero (preserve), `SimilarityThreshold` out of range (set 0.6), valid (preserve) |

**Known untestable branch:** `json.MarshalIndent` on a plain `Config` struct cannot return an error. The error check exists as defensive code and its `return` is never reached, so the coverage report shows it as uncovered: `Save` measures 87.5% for that one statement.

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
| Subcommand correction | Typos across git, docker, kubectl and the trailing arguments preserved; `docker imagse` becomes `docker images`, not `docker image` |
| Tool-name correction | Mistyped tool with a valid subcommand, mistyped tool plus mistyped subcommand |
| Command aliases | `gti` to `git` regardless of the threshold, with the subcommand then corrected; a `gti` the lookup knows is left alone |
| Windows subcommand tools | winget, choco, scoop, net, sc, reg, netsh |
| Windows standalone commands | dir, mkdir, copy, ipconfig, tasklist, arguments preserved, below threshold left alone |
| PowerShell aliases | `ls` never becomes `cls`; the alias set is never corrected |
| Command lookup | A known command is not renamed (`code` stays, never `mode`); without the lookup it is; a known tool's subcommand is still corrected; an unknown typo still is; `WithCommandLookup` copies, keeps the threshold and ignores nil |
| `bestMatch` | A tie on similarity goes to the candidate closest in length, in either list order |
| `similarity` | Equal strings, empty strings, wholly different, either side of the default threshold |
| `damerauLevenshtein` | Both empty, one empty, equal, single deletion, known distance, adjacent transposition |

---

### shell

**Approach:** `t.TempDir()` for all profile file operations. Unexported helpers (`removeSnippet`, `readProfileSafe`) tested directly.

**Branches covered:**

| Function | Branch |
|----------|--------|
| `ProfileSnippet` | Returns string with both markers and binary path; matches the block shown in `ARCHITECTURE.md` |
| `resolveDocumentsDir` | Known folder used (a moved, OneDrive-style folder), lookup failed (home fallback), lookup empty (home fallback), home not consulted when the lookup answers, both fail (error wrapped) |
| `profilePathUnder` | Both shells' profiles under a moved Documents folder |
| `knownDocumentsFolder` | Real lookup returns an existing absolute directory; `DefaultProfilePath` sits under it (Windows only) |
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
| `newEngine` / `onPath` | `code.cmd` on PATH (not corrected), not on PATH (corrected), only in the current directory (does not count) |
| `printUsage` | Smoke test (does not panic) |

**`main()` excluded:** calls `os.Exit(1)` which terminates the test process. The pattern `func main() { if err := run(...); err != nil { os.Exit(1) } }` is idiomatic Go and universally excluded from test coverage.

---

## Mocking Strategy

CommandFixer is designed to avoid mocking:

- **File I/O**: All file-dependent functions accept a `path string` parameter. Tests pass `t.TempDir()` paths. No mocking framework needed.
- **`os.Executable()`**: Returns the test binary path in test context. Fine for verifying profile content.
- **`os.UserHomeDir()`**: Called in `DefaultConfigDir/Path`; `DefaultProfilePath` calls it only as the fallback when Windows cannot say where Documents is. These are tested for format (suffix/contains), not for exact value. No mocking needed.
- **The Documents folder**: `resolveDocumentsDir` takes the known-folder lookup and the home lookup as plain functions, so `documents_test.go` hands it a moved folder such as `D:\OneDrive - Contoso\Dokumente`. That is how the moved-folder case is tested; no machine it has run on has a moved Documents folder.
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

There is no CI pipeline in this repository. If one is added, it should call the
same script a developer calls rather than restating the checks:

```yaml
- name: Lint
  run: pwsh ./build.ps1 -Lint

- name: Test and coverage
  run: pwsh ./build.ps1 -Coverage
```

That matters more than it looks. A pipeline that reimplements the coverage
threshold puts the floor in a second place, where it drifts from the first and
then quietly disagrees with it. One definition, called from both.

---

## Troubleshooting Tests

**Test leaves temp files:**
`t.TempDir()` is cleaned up automatically by the test runner. No manual cleanup needed.

**Adding concurrent state:**
The `Logger` struct uses `sync.Mutex`. Protect anything new with the existing mutex or a new one. There is no race detector here to catch you (see above), so this is held by review.

**Profile install test fails on CI (no home dir):**
`DefaultConfigPath()` calls `os.UserHomeDir()`, as does `DefaultProfilePath()` when the Documents lookup fails; it reads `USERPROFILE` on Windows and `HOME` elsewhere. On a headless runner without one, set it for the run:

```powershell
$env:USERPROFILE = (New-Item -ItemType Directory -Force "$env:TEMP\cf-home").FullName
go test ./...
```

**Test isolation:**
Every test calls `t.Parallel()` and gets its own `t.TempDir()`, with one set of exceptions: the tests in `path_windows_test.go` set PATH, PATHEXT and the working directory for the whole process, so they run alone and put everything back when they finish.

**The current-directory test passing for the wrong reason:**
`exec.LookPath` only searches the current directory while `NoDefaultCurrentDirectoryInExePath` is absent from the environment. Some machines set it; with it set the test would pass whatever `onPath` did. The test removes the variable for its own duration for that reason; do not take that line out.
