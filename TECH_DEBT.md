# CommandFixer: Technical Debt

A standing reference to the project's outstanding technical debt. It records what is still open, weighs whether each item is worth doing and gives the rationale. Every item is a behaviour-preserving internal concern: nothing here proposes reverting a feature or changing observable behaviour. Scope is the whole repository (the Go packages, the PowerShell install and uninstall scripts, `build.ps1` and the GitHub Pages site) read against `ARCHITECTURE.md` and `TESTING.md`.

This is a small tool (roughly 3,300 lines across nineteen Go files) and the file is short in proportion. `build.ps1` carries the whole workflow (build, test, lint, coverage and clean) and `ARCHITECTURE.md` describes the packages honestly as modules rather than claiming a layering the code does not have.

**There is currently no open technical debt.** The sections below record what is deliberately left alone and what only looks like debt, so it is not re-raised.

---

## Looks like debt, not worth touching

- The flat package layout (`config`, `corrector`, `logger`, `shell`) rather than `internal/{domain,application,infrastructure}`. At 3,300 lines with one clear input and one clear output, four cohesive packages is the proportionate structure and `ARCHITECTURE.md` describes them accurately as modules. `structural_test.go` holds one import rule over that layout, which is what the separation actually needed; four directories was never the answer.
- **The absence of a race detector.** Removed by decision on 2026-08-06 rather than left as a switch that cannot run: Go's detector requires cgo, this tool is CGO-free on purpose and the machines it is built on have no C toolchain, so it could only ever exit `-race requires cgo`. A check that cannot run reads as a check being performed, which is worse than no check. The `sync.Mutex` in `logger` is unchanged and is now held by review. Do not re-add the switch without also adding a toolchain that can run it.
- `commandfixer.exe` in the working tree. Build output, correctly untracked.
- `config.example.json` at root beside the `config` package. One example file for one loader.
- `ARCHITECTURE.md`'s "Extending the Architecture" section describing a possible HTTP service mode and log rotation. Documented future options, not commitments.

## Not debt (do not "fix" these)

These look like candidates but are correct as they stand; changing them would regress or add cost for nothing.

- **The absence of CGO and any dependency beyond the standard library.** A single static binary that hooks a shell prompt must start fast and must not fail to load. Pure Go is the whole point.
- **The uninstall script's manual fallback path.** It must exist: an uninstall that only works while the binary is present is not an uninstall. It no longer restates the markers or the profile paths, which come from `profile-hook.ps1`; `shell/markers_test.go` fails if the Go side drifts from that file.
- **The hook markers existing in both Go and PowerShell.** Two languages cannot share a literal and the fallback exists precisely for when the binary that would supply them is gone. One definition per language with a test forbidding drift is the honest shape; a generated file would add a build step to save nothing.
- **The coverage floor sitting at 83% rather than 100%.** `corrector` is at 100% because it is pure computation; `main.go`'s install and uninstall commands write to a real user's PowerShell profile; the honest floor is the level the suite actually holds. A floor picked from an aspiration only teaches people to lower it.
- **`corrector/engine.go` being pure string logic with no I/O.** This is why its tests are simple table tests and why the package is worth protecting. `structural_test.go` keeps it that way and that test is verified to fail when the rule is broken rather than assumed to.
- **GPL-3.0.** The portfolio default for tools.
