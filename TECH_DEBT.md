# CommandFixer: Technical Debt

A standing reference to the project's outstanding technical debt. It records what is still open, weighs whether each item is worth doing and gives the rationale. Every item is a behaviour-preserving internal concern: nothing here proposes reverting a feature or changing observable behaviour. Scope is the whole repository (the Go packages, the PowerShell install and uninstall scripts, the Makefile and the GitHub Pages site) read against `ARCHITECTURE.md` and `TESTING.md`.

This is a small tool (roughly 3,300 lines across ten Go files) and the file is short in proportion. The Makefile already carries `test`, `test-race`, `coverage`, `coverage-html` and `lint` targets, and `ARCHITECTURE.md` describes the packages honestly as modules rather than claiming a layering the code does not have.

---

## 1. The version is a literal in `main.go`

`main.go:18` declares `const appVersion = "2.0.0"`, and `commandfixer version` prints it. There is no `VERSION` file and no other version string in the repository.

This is a distributed binary that a user installs into their PowerShell profile, so the version is part of its interface: it is the first thing a bug report should carry. Holding it as a constant means bumping it is a source edit that can be forgotten, and `git tag` and the binary can silently disagree.

Add a `VERSION` file at root and read it at build time through `-ldflags="-X main.appVersion=$(cat VERSION)"` in the Makefile and `build.ps1`, with the constant kept as a `0.0.0-dev` fallback so a plain `go build` still works. The `GOFLAGS` variable in the Makefile is already the right place for it.

## 2. Four files exceed the 400-line cap and nothing measures them

| File | Lines |
|---|---|
| `corrector/engine_test.go` | 672 |
| `main_test.go` | 471 |
| `shell/powershell_test.go` | 464 |
| `corrector/engine.go` | 422 |

There is no structural test in the repository at all, so none of these is reported. `corrector/engine.go` is the one that matters: it is the correction logic, the thing the whole tool exists to do, and it is the file most likely to grow as new command patterns are added. Its test file at 672 lines is the largest thing here by half again.

The engine splits naturally along the kinds of correction it performs (command-name matching, subcommand transposition, argument handling), and the test file splits the same way. The rule applies to test files exactly as to source.

Adding a size assertion beside the existing tests would cost a few lines and would stop this recurring.

## 3. The coverage target has no threshold

`make coverage` runs `go test -coverprofile -covermode=atomic ./...` then `go tool cover -func`, which prints the total and exits zero at any level. `TESTING.md` documents the position in prose.

Go has no `--cov-fail-under`, so the usual answer is a few lines in the Makefile: parse the total from `go tool cover -func` and fail below a stated figure. Pick the figure from what the suite achieves today rather than from an aspiration. Without it, the number is reported and never acted on.

`make lint` runs `go vet` only. The portfolio standard adds `staticcheck`, which is a strict superset, and it needs no persistent install:

```
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

Add it to the `lint` target and add `gofmt -l .` beside it so formatting is checked rather than assumed.

## 4. The install and uninstall scripts repeat each other's knowledge

`install.ps1` writes a hook block into both the PowerShell 5 and PowerShell 7 profiles, delegating to the binary. `uninstall.ps1` removes it the same way, and then carries a manual fallback that rebuilds the list of profile paths and strips the block itself, for the case where the binary is already gone.

That fallback is the right instinct: an uninstall must work when the thing being uninstalled is missing. The cost is that the profile paths and the hook-block markers now exist in two scripts and in the Go code, so a change to any one of them can leave a user with a dead hook line in their profile that runs on every prompt.

The markers (`# CommandFixer Integration - DO NOT EDIT` and its closing line) should be defined once and referenced, and the fallback should read the same definition rather than restating it. This is small and it protects the one thing this tool does that persists outside its own directory.

## 5. The Makefile does not work on the tool's only supported platform

The header says so plainly: "Targets work on Linux/macOS; for Windows use build.ps1 or 'go build' directly." `clean` uses `rm -f`, `build-windows` uses shell-style `GOOS=` prefixes.

CommandFixer corrects PowerShell commands. Its users are on Windows. So the file carrying the project's test, lint and coverage workflow is the one file its developer cannot run, and `build.ps1` covers building but not testing or linting.

`build.ps1` already carries build, test, coverage, race and clean, so the gap is narrower than it first reads: it lacks only a lint step. Either finish `build.ps1` and delete the Makefile, or keep both in step by hand. The first is more honest for a Windows-only tool and removes the duplication rather than maintaining it twice. Whichever is chosen, items 2 and 3 both add checks that need somewhere to live, and that somewhere has to be runnable on Windows.

---

## Looks like debt, not worth touching

- The flat package layout (`config`, `corrector`, `logger`, `shell`) rather than `internal/{domain,application,infrastructure}`. At 3,300 lines with one clear input and one clear output, four cohesive packages is the proportionate structure and `ARCHITECTURE.md` describes them accurately as modules. `structural_test.go` holds one import rule over that layout, which is what the separation actually needed; four directories was never the answer.
- `commandfixer.exe` in the working tree. Build output, correctly untracked.
- `config.example.json` at root beside the `config` package. One example file for one loader.
- `ARCHITECTURE.md`'s "Extending the Architecture" section describing a possible HTTP service mode and log rotation. Documented future options, not commitments.

## Not debt (do not "fix" these)

These look like candidates but are correct as they stand; changing them would regress or add cost for nothing.

- **The absence of CGO and any dependency beyond the standard library.** A single static binary that hooks a shell prompt must start fast and must not fail to load. Pure Go is the whole point.
- **`go test -race` as its own target.** The tool runs inside a PowerShell prompt hook, so a data race would surface as an intermittent failure in someone else's terminal. Keeping the race detector one command away is right.
- **The uninstall script's manual fallback path.** It duplicates knowledge (item 5) and it must exist: an uninstall that only works while the binary is present is not an uninstall.
- **`corrector/engine.go` being pure string logic with no I/O.** This is why its tests are simple table tests and why the package is worth protecting. `structural_test.go` keeps it that way, and that test is verified to fail when the rule is broken rather than assumed to.
- **GPL-3.0.** The portfolio default for tools.
