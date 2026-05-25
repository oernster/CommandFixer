# Development Guide

Local setup, build steps, and debugging for CommandFixer.

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

Or manually:

```powershell
$env:GOOS   = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o commandfixer.exe .
```

### Cross-compile from Linux/macOS (Makefile)

```bash
make build-windows    # produces commandfixer.exe
make test             # run tests
make coverage         # generate coverage.out + print summary
make coverage-html    # open coverage.html in browser
```

### Other targets

```powershell
.\build.ps1 -Coverage    # HTML coverage report, opens in browser
.\build.ps1 -Race        # tests with race detector
.\build.ps1 -Clean       # remove build artefacts
```

---

## Project Layout

```
CommandFixer/
├── main.go                  Entry point and CLI dispatch
├── main_test.go             Integration-style tests for CLI commands
├── go.mod                   Module definition (no external deps)
├── config/
│   ├── loader.go            JSON config load/save, defaults
│   └── loader_test.go
├── corrector/
│   ├── engine.go            Rule compilation and command correction
│   └── engine_test.go
├── shell/
│   ├── powershell.go        Profile snippet generation and install/uninstall
│   └── powershell_test.go
├── logger/
│   ├── stats.go             JSONL log writer and stats aggregator
│   └── stats_test.go
├── config.example.json      Starter typo dictionary
├── install.ps1              One-shot installer
├── build.ps1                Build/test/coverage helper
└── Makefile                 Cross-platform make targets
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

## Adding a New Typo Rule

1. Open `%USERPROFILE%\.typo-fixer\config.json` (or `config.example.json` for the repo starter set).
2. Add an entry to `"typos"`:

```json
{ "from": "git comit", "to": "git commit" }
```

For regex patterns set `"regex": true`:

```json
{ "from": "gti\\s+", "to": "git ", "regex": true }
```

3. No restart needed. Each invocation of `commandfixer correct` re-reads the config.

---

## Adding a New CLI Command

1. Add a new `case` in `dispatch()` in `main.go`.
2. Implement `cmdFoo(args []string, cfgPath string) error`.
3. Add tests in `main_test.go` following the existing pattern.
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

### Race conditions

```powershell
go test -race ./...
```

The logger uses `sync.Mutex` so concurrent corrections are safe.

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
$env:GOOS   = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w -X main.appVersion=1.2.0" -o commandfixer.exe .
```

The `-s -w` flags strip debug info. `-X main.appVersion=...` injects the version at link time.
