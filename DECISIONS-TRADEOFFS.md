# Decisions and trade-offs

The deliberate choices CommandFixer rests on: what was chosen, what was given
up for it and why. Each entry is the decision as the product makes it today.
The detail behind each one lives in [ARCHITECTURE.md](ARCHITECTURE.md), with
the tests that hold it in [TESTING.md](TESTING.md) and the build workflow in
[DEVELOPMENT.md](DEVELOPMENT.md); [TECH_DEBT.md](TECH_DEBT.md) records what
only looks like debt and is to be left alone.

## The product as a whole

### PowerShell on Windows, nothing else

CommandFixer hooks the Enter key of PowerShell 7 and Windows PowerShell 5
through PSReadLine. The build script produces a Windows binary; there is no
version for any other shell.

- **Rather than:** a general shell tool for bash, zsh or cmd.
- **Gains:** one prompt mechanism to get right; the correction list can
  include the Windows commands and PowerShell's own aliases.
- **Costs:** it has no meaning outside a Windows PowerShell prompt.

### Spelling, not intent

A typed command is compared with known command names by how alike the letters
are. A misspelt command can become the one that was meant; a correctly spelt
wrong command is left as it is.

- **Rather than:** guessing what the user was trying to do.
- **Gains:** every suggestion can be explained by the letters typed.
- **Costs:** it never helps with a command that is spelt right but wrong.

### A built-in database rather than a dictionary to write

The tools, their subcommands, the Windows commands and the PowerShell aliases
are compiled into the binary. The settings file holds settings alone: where
the log goes, its size and how close a match must be. The first version
worked from a list of typo rules the user wrote in the settings file,
optionally as regular expressions; that
list was retired.

- **Rather than:** a hand-maintained typo dictionary.
- **Gains:** it corrects useful mistakes the moment it is installed, with
  nothing written first.
- **Costs:** teaching it a new tool is a change to the source and a rebuild.
  A user cannot add a correction of their own.

### A suggestion is asked about, never applied silently

When there is a correction the prompt shows it and waits for one key. The
first version replaced the line on its own and only reported what it had done.

- **Rather than:** rewriting the line behind the user's back.
- **Gains:** nothing runs that the user did not see first.
- **Costs:** one key press for every correction.

### Nothing leaves the machine

No Go file in the product imports a networking package. The binary reads its
settings and writes its log in the user's own folder; there is no account, no
telemetry and nothing running in the background.

- **Rather than:** shared or downloaded correction lists; usage figures.
- **Gains:** nothing about what is typed goes anywhere.
- **Costs:** no figures on which corrections are useful to anybody else.
  Only the correction engine is held to this by a test; for the rest it is
  held by review.

## Correction

### Transpositions count as one mistake

Similarity is measured with the Damerau-Levenshtein distance, which counts two
swapped neighbouring letters as one edit, scaled by the longer word's length.

- **Rather than:** plain Levenshtein distance, which counts a swap as two.
- **Gains:** the commonest typing slip (psuh for push, gti for git) scores as
  close as it looks.
- **Costs:** none recorded.

### One threshold the user can tune

A match must reach a similarity of 0.6 to be offered. The settings file can
change that within the range above zero up to one; a value outside it falls
back to the default.

- **Rather than:** a fixed figure; per-tool sensitivities.
- **Gains:** one setting explains the behaviour.
- **Costs:** lowering it catches more typos at the price of corrections
  nobody wanted.

### An exact match is never corrected

A word that already appears in the database is left alone. That is why
PowerShell's POSIX-style aliases are listed as commands in their own right.

- **Rather than:** always taking the nearest entry.
- **Gains:** a valid command is never rewritten into its neighbour. Before ls
  was listed, it was rewritten to cls.
- **Costs:** every valid name has to be in the database. A real subcommand
  missing from a tool's list is corrected to its nearest neighbour.

### A real command on PATH is never renamed

Before renaming the first word, the engine asks whether a command of that name
resolves on PATH, extensions such as .cmd included. If it does, the line is
left alone. A program found only in the current folder does not count, since
PowerShell will not run it without a .\ prefix either. Only the first word
is asked about: the subcommand of a known tool is still corrected.

- **Rather than:** trusting the database to know every program a machine has.
  Without this check code became mode, node became mode and tar became start.
- **Gains:** installed programs the database has never heard of are safe.
- **Costs:** a typo that happens to be the name of a real program is not
  caught.

### A habitual slip is corrected outright

A short list of always-wrong spellings (gti for git) is replaced whatever the
threshold, unless a command of that name really exists on PATH.

- **Rather than:** relying on the similarity score for them.
- **Gains:** the slips made every day are caught every time.
- **Costs:** the list is kept by hand.

### Tools and Windows commands compete on closeness

A mistyped first word is compared with both the known tools and the Windows
commands; the closer wins and a tie goes to the tool. A corrected tool then
has its subcommand corrected too.

- **Rather than:** checking one list before the other.
- **Gains:** a slip is corrected to its nearest name whichever list holds it.
- **Costs:** none recorded.

### The engine is pure computation

The correction engine takes a string and returns a string. It reads no files,
no environment and no clock; the PATH check is handed to it from outside. A
structural test forbids it any import that reaches outside the process.

- **Rather than:** letting the engine look things up for itself.
- **Gains:** its tests are plain cases with no fixtures to build.
- **Costs:** the wiring has to supply the PATH lookup; an engine built
  without one knows only the database.

## The prompt hook

### A fresh process on every Enter

The hook runs the installed binary by its full path each time Enter is
pressed on a line that is not blank.

- **Rather than:** a resident service the hook talks to, which the
  architecture notes describe as a possible later option.
- **Gains:** nothing runs in the background; a new build is used on the very
  next Enter without restarting the shell.
- **Costs:** a process starts for every command typed.

### Enter means yes

At the prompt, Y or Enter accepts the suggestion; any other key keeps the line
as typed.

- **Rather than:** a default of no.
- **Gains:** accepting takes the key the user was already pressing.
- **Costs:** a second Enter pressed without reading accepts the suggestion.

### A failure is never visible

A missing binary is skipped by a check before it is called. A binary that
fails or prints nothing leaves the line as typed. Either way the command runs
unchanged.

- **Rather than:** reporting errors at the prompt.
- **Gains:** an uninstalled or broken CommandFixer costs nothing but a missing
  correction; it never raises an error on every key press.
- **Costs:** a broken install goes unnoticed until somebody wonders why
  nothing is being corrected.

### No continuation prompt from Enter

Before submitting, the hook strips a trailing backtick and parses the line
with PowerShell's own parser. Input the parser calls incomplete (an unclosed
quote, a dangling pipe) beeps and stays on the line.

- **Rather than:** letting PowerShell open its continuation prompt.
- **Gains:** a slip of the finger never leaves the user stranded at the
  continuation prompt.
- **Costs:** a trailing backtick is always removed; incomplete input cannot be
  submitted with Enter.

### Both PowerShells, one profile each

The hook is written into the all-hosts profile of the current user for
PowerShell 7 and for Windows PowerShell 5 alike.

- **Rather than:** PowerShell 7 alone, as at first.
- **Gains:** the same behaviour in whichever PowerShell is opened.
- **Costs:** a profile file is created for a shell that may never be used.

## Installing and removing

### Installed for one user

The binary goes into the user's local application data folder, which is added
to the user's own PATH. Settings and the log live in a folder in the user's
home directory.

- **Rather than:** a machine-wide install.
- **Gains:** nothing outside the user's own folders and settings is changed.
- **Costs:** each account on a machine installs separately.

### Installing again leaves the hook as it is

A profile that already holds the hook is not touched by a second install. The
installer overwrites the binary and keeps the PATH entry and any existing
settings.

- **Rather than:** replacing the hook block on every install.
- **Gains:** re-running the installer never rewrites a profile.
- **Costs:** a release that changes the hook needs an uninstall before the
  install, then a restart of PowerShell.

### Removal works without the binary

The hook sits between fixed marker lines. The binary removes the block when
it is present; the uninstall script carries its own way of removing it for
when the binary has already gone. The markers are defined once in Go and once
in a shared PowerShell file; a test fails if the two disagree.

- **Rather than:** an uninstall that needs the binary; one marker definition
  generated for both languages.
- **Gains:** an uninstall always uninstalls. A marker changed on one side only
  cannot leave a hook nothing can find.
- **Costs:** the markers exist twice, held together by a test.

### The user's data is kept on removal

Uninstalling removes the hook, the binary and the PATH entry. The settings and
the log stay unless removal is asked for explicitly.

- **Rather than:** removing everything.
- **Gains:** a reinstall picks up where the user left off.
- **Costs:** a full clean needs the extra switch.

### Settings are optional

A missing settings file means defaults; a partial one has its gaps filled.

- **Rather than:** refusing to run until a settings file exists.
- **Gains:** a fresh machine works at once.
- **Costs:** none recorded.

## The record of corrections

### A local log, one line per correction

Each accepted correction is appended to a plain file as one self-contained
line of JSON. Reading the statistics skips any line it cannot parse.

- **Rather than:** a database; rewriting a single structured file.
- **Gains:** several open shells can append to it without rewriting what is
  there; a damaged line costs that line alone.
- **Costs:** the statistics are a simple count.

### No rotation

The log grows without limit. The settings file carries a maximum line count;
nothing acts on it yet.

- **Rather than:** trimming the log as it grows.
- **Gains:** each correction is one append with nothing read back.
- **Costs:** a setting that does nothing; a log that is never trimmed.

## Engineering

### Go, standard library only, no C

The program is one Go binary using only the standard library, with no C code.

- **Rather than:** third-party packages; cgo.
- **Gains:** a single file with nothing beside it to fail to load.
- **Costs:** Go's race detector needs cgo, so it cannot run here. It was
  removed rather than left as a switch that could only fail; the one lock in
  the logger is held by review.

### Four packages, one place where they meet

Settings, correction, the log and the shell hook are four flat packages. Only
the entry point wires them together; a structural test forbids any of them to
import another.

- **Rather than:** a layered domain, application and infrastructure layout.
- **Gains:** each package can be read and tested on its own; the separation
  is a test result rather than a habit.
- **Costs:** none recorded. At this size four cohesive packages is the
  proportionate shape.

### Small files

No Go file may exceed four hundred lines, tests included. None may sit in the
last five per cent below that. A file that has to be split is split to a
size with room in it.

- **Rather than:** letting files grow; shaving a file to just under the cap.
- **Gains:** files split at real seams. The correction package became three
  files (data, policy and metric) along this rule.
- **Costs:** more files.

### A coverage floor at the level the suite holds

The coverage gate fails below 83 per cent, the level the suite already held
when it was set.

- **Rather than:** a round aspirational figure.
- **Gains:** the gate is never lowered to let a run pass; it is raised when
  the suite earns it.
- **Costs:** the commands that write to a real user's profile stay partly
  untested.

### Tests with real files

Tests write to temporary folders and call the code directly. There is no
mocking library. The PATH check is tested against a real folder holding a
.cmd shim. The recent regression tests and guards were each seen to fail
with the fix removed before they were kept.

- **Rather than:** mocks standing in for the filesystem and PATH.
- **Gains:** a passing test means the real thing works.
- **Costs:** the PATH tests change settings for the whole process, so they
  run alone.

### The documented hook is the real hook

The hook shown in ARCHITECTURE.md is checked by a test against what the code
generates. An earlier copy drifted until it described a hook that no longer
existed.

- **Rather than:** a hand-kept copy.
- **Gains:** a reader sees exactly what is installed; the failure message
  carries the block to paste in.
- **Costs:** the documentation is held to the exact text of the code.

### One build script

The build, test, lint, coverage and clean steps are one PowerShell script.
An unknown switch is an error rather than being ignored.

- **Rather than:** a Makefile, which duplicated the workflow on a platform
  the tool's users do not use and went stale.
- **Gains:** one runner, runnable where the work happens. A mistyped switch
  can no longer run a plain build and report success.
- **Costs:** the lint step runs the latest staticcheck, so its version is not
  pinned.

### One home for the version

The version lives in one file. The build writes it into the binary at link
time and stamps it into the docs and the site, refusing anything that is not
a plain three-part version. The site's stylesheet and script links also
carry a hash of their content.

- **Rather than:** version strings edited by hand in several places.
- **Gains:** a release cannot ship a site naming the previous version; a
  browser fetches a changed stylesheet at once rather than a cached one.
- **Costs:** the docs carry stamp markers; the site must be stamped from the
  source.

### Go files checked out with LF

Every Go file is checked out with LF line endings whatever the local git
settings.

- **Rather than:** the Windows default of CRLF.
- **Gains:** the formatter no longer lists every clean file as wrong on a
  Windows checkout.
- **Costs:** a working tree cloned before the rule needs a one-time rewrite.

### GPL-3.0

CommandFixer is released under the GPL, version 3.

- **Rather than:** a permissive licence.
- **Gains:** the portfolio default for tools; changes stay open.
- **Costs:** none recorded.
