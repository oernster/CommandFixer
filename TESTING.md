# Testing Guide

Testing strategy, coverage requirements, and how to run tests for CommandFixer.

See [DEVELOPMENT.md](DEVELOPMENT.md) for build setup.
See [ARCHITECTURE.md](ARCHITECTURE.md) for module design.

---

## Coverage Target

**100% of all reachable branches in every package.**

The only legitimate exclusion is the `main()` function itself: it calls `os.Exit(1)` on error, which cannot be tested within the same process without reflection tricks. `main()` is a 3-line wrapper around `run()`, which is fully tested. All other functions are covered.

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
go test -v -run TestCorrect_LiteralMatch ./corrector/
go test -v -run TestInstall ./shell/
```

### Race detector

```powershell
go test -race ./...
```

The Logger uses `sync.Mutex`. The race detector verifies no concurrent map/slice access escapes the mutex.

---

## Coverage Measurement

### Generate and print summary

```powershell
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

Expected output (all lines should show 100.0%):

```
github.com/oernster/commandfixer/config/loader.go:DefaultConfigDir     100.0%
github.com/oernster/commandfixer/config/loader.go:DefaultConfigPath    100.0%
...
total:                                                                  100.0%
```

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
| `corrector/engine_test.go` | `corrector` package |
| `shell/powershell_test.go` | `shell` package (including unexported helpers) |
| `logger/stats_test.go` | `logger` package |
| `main_test.go` | `main` package CLI dispatch |

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

**Known untestable branch:** `json.MarshalIndent` on a plain `Config` struct cannot return an error. The error check exists as defensive code. Coverage tools will flag it as covered because the function executes, but the error path is unreachable in practice.

---

### corrector

**Approach:** Pure logic, no file I/O. All tests use in-memory `config.Config` values.

**Branches covered:**

| Function | Branch |
|----------|--------|
| `New` | Empty config, literal rules only, valid regex, invalid regex (compile error) |
| `Correct` | No rules, no match, literal substring match, literal full match, multiple rules both fire, regex match, regex no match, regex with capture group, mixed literal + regex |
| Rule recording | Last-fired rule stored in `RuleFrom`/`RuleTo` |
| Edge cases | Empty command, original preserved |

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
| `dispatch` | No args, `help`, `--help`, `-h`, `version`, `--version`, `-v`, unknown command |
| `cmdCorrect` | No args (error), no match (unchanged), match with log write, multi-word input joined, missing config (LoadOrDefault default), bad JSON config (error), invalid regex in config (error) |
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
- **Regex compilation**: Tested with both valid and invalid patterns; no mocking of `regexp.Compile`.

---

## Test Fixtures

No external fixture files. All test data is defined inline:

```go
// Inline config for corrector tests
cfg := &config.Config{
    Typos: []config.TypoEntry{
        {From: "git sattus", To: "git status"},
    },
}

// Inline JSON for config tests
content := `{"typos":[{"from":"git sattus","to":"git status"}]}`
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
