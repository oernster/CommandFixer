# Development Guide

Local setup, build steps and debugging for CommandFixer.

See [ARCHITECTURE.md](ARCHITECTURE.md) for code structure and design decisions.
See [TESTING.md](TESTING.md) for the full testing strategy.

---

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.21+ | https://go.dev/dl |
| PowerShell | 7.x | For integration testing |
| Git | any | For version control |

No third-party Go dependencies. Standard library only.

---

## Clone and Verify

```powershell
git clone https://github.com/oernster/commandfixer
cd commandfixer
go vet ./...          # should produce no output
go test ./...         # all tests should pass
```

---

## Build

### Windows binary (recommended)

```powershell
.\build.ps1           # builds commandfixer.exe
.\build.ps1 -Test     # tests then builds
```

Or manually, reading the version from the `VERSION` file as the script does:

```powershell
$env:GOOS   = "windows"
$env:GOARCH = "amd64"
go build "-ldflags=-s -w -X main.appVersion=$(Get-Content VERSION)" -o commandfixer.exe .
```

A plain `go build` still works and produces a binary reporting `0.0.0-dev`, which
is how you can tell one was built without the version.

### Other switches

```powershell
.\build.ps1 -Lint        # gofmt, go vet and staticcheck
.\build.ps1 -Coverage    # coverage report, fails below the floor
.\build.ps1 -Clean       # remove build artefacts
```

`build.ps1` is the whole workflow. There is deliberately no Makefile: this tool
corrects PowerShell commands and its users are on Windows, so a second copy of
the workflow written for a platform its developer does not use was a checklist
that could not be run and went stale. One runner, runnable where the work is.

---

## Project Layout

```
CommandFixer/
├── main.go                  Entry point and CLI dispatch
├── main_test.go             CLI routing and shared test helpers
├── commands_test.go         suggest, correct, log and stats
├── install_test.go          The commands that write to a PowerShell profile
├── structural_test.go       Import-boundary and file-size rules
├── VERSION                  The single source of truth for the version
├── go.mod                   Module definition (no external deps)
├── config/
│   ├── loader.go            JSON config load/save, defaults
│   └── loader_test.go
├── corrector/
│   ├── engine.go            Correction policy: what to correct and when
│   ├── database.go          Known tools, subcommands and Windows commands
│   ├── distance.go          Damerau-Levenshtein distance and similarity
│   ├── engine_test.go
│   ├── windows_test.go
│   └── distance_test.go
├── shell/
│   ├── powershell.go        Profile snippet generation and install/uninstall
│   ├── powershell_test.go   The snippet, the paths, the read-only operations
│   ├── install_test.go      The operations that change a profile
│   └── markers_test.go      Go and PowerShell markers still agree
├── logger/
│   ├── stats.go             JSONL log writer and stats aggregator
│   └── stats_test.go
├── config.example.json      Starter settings file
├── profile-hook.ps1         Hook markers and profile paths, defined once
├── install.ps1              One-shot installer
├── uninstall.ps1            Uninstaller, with a binary-independent fallback
└── build.ps1                Build, test, lint and coverage runner
```

---

## Local Development Workflow

### 1. Edit source

Edit any `.go` file. No hot-reload; rebuild to test changes.

### 2. Run tests

```powershell
go test ./...                        # all packages
go test ./corrector/...              # single package
go test -run TestCorrect_Literal ./corrector/  # single test
go test -v ./...                     # verbose output
```

### 3. Quick integration check

Build the binary and call it directly:

```powershell
go build -o commandfixer.exe .
.\commandfixer.exe correct "git sattus"
# Should print: git status
```

### 4. Test with a live PS profile (optional)

```powershell
.\commandfixer.exe install
# Restart PowerShell or dot-source the profile:
. $PROFILE
```

Type `git sattus` and press Enter. You should see the correction message.

---

## Adding a Command to the Database

There is no user-facing typo dictionary. Corrections come from the built-in
database, so teaching CommandFixer about a new tool is a source change in
`corrector/database.go` and nothing else:

1. **A CLI tool with subcommands**: add a key to `commandDB` with its valid
   subcommands. Keep the list sorted and complete: a real subcommand missing
   from it will be "corrected" to a neighbour, which is worse than no
   correction at all.
2. **A Windows standalone command**: add it to `windowsCommands`.
3. **A shell alias that must never be corrected**: add it to `windowsCommands`
   too. Exact matches are left alone, which is why `ls` survives despite being
   one insertion from `cls`.
4. **A habitual transposition**: add it to `commandAliases`, which applies
   unconditionally and ignores the threshold.

Add a case to `corrector/windows_test.go` or `engine_test.go` alongside it.
Nothing needs rebuilding beyond the binary and no restart is required, because
the hook shells out on every prompt.

### Tuning sensitivity

`similarity_threshold` in `config.json` sets how close a match must be, in the
range (0.0, 1.0], defaulting to 0.6. Lower catches more typos and risks
correcting things you meant.

---

## Adding a New CLI Command

1. Add a new `case` in `dispatch()` in `main.go`.
2. Implement `cmdFoo(args []string, cfgPath string) error`.
3. Add tests in `commands_test.go` following the existing pattern or in
   `install_test.go` if the command writes to a PowerShell profile.
4. Update `printUsage()`.

---

## Debugging

### Verbose correction check

```powershell
.\commandfixer.exe correct "git sattus"
```

If it prints `git sattus` unchanged, check:

- Config file path: `.\commandfixer.exe stats` should load without error.
- Config syntax: validate JSON with `Get-Content config.json | ConvertFrom-Json`.
- Rule content: the `from` value must be a substring of or match the full command.

### Profile hook not firing

```powershell
# Check the profile exists and contains the hook:
Get-Content $PROFILE | Select-String "CommandFixer"

# Re-install:
.\commandfixer.exe install

# Reload profile in current session:
. $PROFILE
```

### View the log

```powershell
.\commandfixer.exe stats

# Or read raw JSONL:
Get-Content "$env:USERPROFILE\.typo-fixer\corrections.log"
```

### Concurrent state

The logger uses `sync.Mutex` so concurrent corrections are safe. There is no
race detector to check that: it needs cgo, which this project does without on
purpose. Protect anything new with the existing mutex or a new one and expect
review rather than a tool to catch it.

---

## Dependency Management

This project has **no third-party dependencies**. Only the Go standard library is used. The `go.sum` file is therefore empty (or absent).

To confirm:

```powershell
go mod tidy   # should not add anything
go mod verify
```

---

## Release Build

```powershell
.\build.ps1
```

That is the release build. It reads the version from the root `VERSION` file and
injects it, so there is nothing to remember and nothing to keep in step by hand.

The `-s -w` flags strip debug info. `-X main.appVersion=...` sets the version at
link time, which only works because `appVersion` is a `var`: the linker cannot
write to a `const`, so while it was one this flag was accepted and silently did
nothing and the binary kept whatever literal was in the source.

Never write a version anywhere but `VERSION`. Nothing else in the repository
holds a real version string.
