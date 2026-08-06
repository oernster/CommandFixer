# CommandFixer: Technical Debt

A standing reference to the project's outstanding technical debt. It records what is still open, weighs whether each item is worth doing and gives the rationale. Every item is a behaviour-preserving internal concern: nothing here proposes reverting a feature or changing observable behaviour. Scope is the whole repository (the Go packages, the PowerShell install and uninstall scripts, `build.ps1` and the GitHub Pages site) read against `ARCHITECTURE.md` and `TESTING.md`.

This is a small tool (roughly 3,300 lines across eleven Go files) and the file is short in proportion. `build.ps1` carries the whole workflow (build, test, race, lint, coverage and clean) and `ARCHITECTURE.md` describes the packages honestly as modules rather than claiming a layering the code does not have.

---

## 1. Four files exceed the 400-line cap and nothing measures them

| File | Lines |
|---|---|
| `corrector/engine_test.go` | 672 |
| `main_test.go` | 471 |
| `shell/powershell_test.go` | 464 |
| `corrector/engine.go` | 422 |

`structural_test.go` now holds the import boundaries but says nothing about size, so none of these is reported. `corrector/engine.go` is the one that matters: it is the correction logic, the thing the whole tool exists to do, and it is the file most likely to grow as new command patterns are added. Its test file at 672 lines is the largest thing here by half again.

The engine splits naturally along the kinds of correction it performs (command-name matching, subcommand transposition, argument handling), and the test file splits the same way. The rule applies to test files exactly as to source.

Adding a size assertion to `structural_test.go` would cost a few lines and would stop this recurring. The portfolio rule has two halves and both are worth carrying: over 400 fails, and 381 to 399 fails too, so a file cannot be shaved to just under the cap and break again on the next edit. Derive the band from the cap rather than writing a second literal.

## 2. The install and uninstall scripts repeat each other's knowledge

`install.ps1` writes a hook block into both the PowerShell 5 and PowerShell 7 profiles, delegating to the binary. `uninstall.ps1` removes it the same way, and then carries a manual fallback that rebuilds the list of profile paths and strips the block itself, for the case where the binary is already gone.

That fallback is the right instinct: an uninstall must work when the thing being uninstalled is missing. The cost is that the profile paths and the hook-block markers now exist in two scripts and in the Go code, so a change to any one of them can leave a user with a dead hook line in their profile that runs on every prompt.

The markers (`# CommandFixer Integration - DO NOT EDIT` and its closing line) should be defined once and referenced, and the fallback should read the same definition rather than restating it. This is small and it protects the one thing this tool does that persists outside its own directory.

---

## Looks like debt, not worth touching

- The flat package layout (`config`, `corrector`, `logger`, `shell`) rather than `internal/{domain,application,infrastructure}`. At 3,300 lines with one clear input and one clear output, four cohesive packages is the proportionate structure and `ARCHITECTURE.md` describes them accurately as modules. `structural_test.go` holds one import rule over that layout, which is what the separation actually needed; four directories was never the answer.
- `commandfixer.exe` in the working tree. Build output, correctly untracked.
- `config.example.json` at root beside the `config` package. One example file for one loader.
- `ARCHITECTURE.md`'s "Extending the Architecture" section describing a possible HTTP service mode and log rotation. Documented future options, not commitments.

## Not debt (do not "fix" these)

These look like candidates but are correct as they stand; changing them would regress or add cost for nothing.

- **The absence of CGO and any dependency beyond the standard library.** A single static binary that hooks a shell prompt must start fast and must not fail to load. Pure Go is the whole point.
- **`go test -race` as its own switch.** The tool runs inside a PowerShell prompt hook, so a data race would surface as an intermittent failure in someone else's terminal. Keeping the race detector one command away is right.
- **The uninstall script's manual fallback path.** It duplicates knowledge (item 2) and it must exist: an uninstall that only works while the binary is present is not an uninstall.
- **The coverage floor sitting at 83% rather than 100%.** `corrector` is at 100% because it is pure computation; `main.go`'s install and uninstall commands write to a real user's PowerShell profile, and the honest floor is the level the suite actually holds. A floor picked from an aspiration only teaches people to lower it.
- **`corrector/engine.go` being pure string logic with no I/O.** This is why its tests are simple table tests and why the package is worth protecting. `structural_test.go` keeps it that way, and that test is verified to fail when the rule is broken rather than assumed to.
- **GPL-3.0.** The portfolio default for tools.
