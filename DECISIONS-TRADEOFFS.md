# Decisions and trade-offs

The deliberate choices CommandFixer rests on: what was chosen, what was given
up for it and why. Each entry is the decision as the product makes it today.
The detail behind each one lives in [ARCHITECTURE.md](ARCHITECTURE.md), with
the tests that hold it in [TESTING.md](TESTING.md) and the build workflow in
[DEVELOPMENT.md](DEVELOPMENT.md); [TECH_DEBT.md](TECH_DEBT.md) records what
only looks like debt and is to be left alone.

## The product as a whole

### PowerShell on Windows, nothing else

CommandFixer hooks the Enter key of PowerShell 7 and Windows PowerShell 5.
There is no version for any other shell or platform.

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
are compiled into the program. The settings hold settings alone: where the
log goes, its size and how close a match must be.

- **Rather than:** a typo dictionary the user writes and maintains.
- **Gains:** it corrects useful mistakes the moment it is installed, with
  nothing written first.
- **Costs:** teaching it a new tool is a change to the source and a rebuild.
  A user cannot add a correction of their own.

### A suggestion is asked about, never applied silently

When there is a correction the prompt shows it and waits for one key.

- **Rather than:** rewriting the line behind the user's back.
- **Gains:** nothing runs that the user did not see first.
- **Costs:** one key press for every correction.

### Nothing leaves the machine

The program reads its settings and writes its log in the user's own folder.
It makes no network connection; there is no account, no telemetry and nothing
running in the background.

- **Rather than:** shared or downloaded correction lists; usage figures.
- **Gains:** nothing about what is typed goes anywhere.
- **Costs:** no figures on which corrections are useful to anybody else.
  Only the correction engine is held to this by a test; for the rest it is
  held by review.

## Correction

### Transpositions count as one mistake

Similarity counts two swapped neighbouring letters as a single edit, scaled by
the length of the longer word.

- **Rather than:** an edit distance that counts a swap as two edits.
- **Gains:** the commonest typing slip (psuh for push, gti for git) scores as
  close as it looks.
- **Costs:** none recorded.

### One threshold the user can tune

A match must reach one similarity threshold to be offered. The settings can
move it; a value outside the valid range falls back to the default.

- **Rather than:** a fixed figure; per-tool sensitivities.
- **Gains:** one setting explains the behaviour.
- **Costs:** lowering it catches more typos at the price of corrections
  nobody wanted.

### A line of one word is left alone

Correction needs at least two words on the line. A lone mistyped word is
never offered a suggestion; the same word followed by anything is.

- **Rather than:** correcting every line, however short.
- **Gains:** a bare word that may be a function or alias of the user's own,
  which the PATH check cannot see, is never rewritten into a command it only
  resembles.
- **Costs:** a mistyped command typed on its own is not caught.

### An exact match is never corrected

A word that already appears in the database is left alone. That is why
PowerShell's POSIX-style aliases are listed as commands in their own right.

- **Rather than:** always taking the nearest entry.
- **Gains:** a valid command is never rewritten into its neighbour.
- **Costs:** every valid name has to be in the database. A real subcommand
  missing from a tool's list is corrected to its nearest neighbour.

### A real command on PATH is never renamed

Before renaming the first word, the program asks whether a command of that
name resolves on PATH, script shims included. If it does, the line is left
alone. A program found only in the current folder does not count, since
PowerShell will not run it without a path either. Only the first word is
asked about: the subcommand of a known tool is still corrected.

- **Rather than:** trusting the database to know every program a machine has.
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
structural test forbids it anything that reaches outside the process.

- **Rather than:** letting the engine look things up for itself.
- **Gains:** its tests are plain cases with no fixtures to build.
- **Costs:** the wiring has to supply the PATH lookup; an engine built
  without one knows only the database.

## The prompt hook

### A fresh process on every Enter

The hook runs the installed program by its full path each time Enter is
pressed on a line that is not blank.

- **Rather than:** a resident service the hook talks to.
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

A missing program is skipped by a check before it is called. A program that
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

### The documented hook is the real hook

The hook shown in the architecture notes is checked by a test against what the
program generates.

- **Rather than:** a hand-kept copy.
- **Gains:** a reader sees exactly what is installed, without installing it.
- **Costs:** the documentation is held to the exact text of the code.

### Both PowerShells, one profile each

The hook is written into the all-hosts profile of the current user for
PowerShell 7 and for Windows PowerShell 5 alike.

- **Rather than:** PowerShell 7 alone.
- **Gains:** the same behaviour in whichever PowerShell is opened.
- **Costs:** a profile file is created for a shell that may never be used.

## Installing and removing

### Installed for one user

The program goes into the user's local application data folder, which is added
to the user's own PATH. Settings and the log live in a folder in the user's
home directory.

- **Rather than:** a machine-wide install.
- **Gains:** nothing outside the user's own folders and settings is changed;
  no administrator rights are asked for.
- **Costs:** each account on a machine installs separately.

### Installing again leaves the hook as it is

A profile that already holds the hook is not touched by a second install. The
installer overwrites the program and keeps the PATH entry and any existing
settings.

- **Rather than:** replacing the hook block on every install.
- **Gains:** re-running the installer never rewrites a profile.
- **Costs:** a release that changes the hook needs an uninstall before the
  install, then a restart of PowerShell.

### Removal works without the program

The hook sits between fixed marker lines. The program removes the block when
it is present; the uninstall script carries its own way of removing it for
when the program has already gone. The markers are defined once in the
program and once in a script both install scripts share; a test fails if the
two disagree.

- **Rather than:** an uninstall that needs the program; one marker definition
  generated for both languages.
- **Gains:** an uninstall always uninstalls. A marker changed on one side only
  cannot leave a hook nothing can find.
- **Costs:** the markers exist twice, held together by a test.

### The user's data is kept on removal

Uninstalling removes the hook, the program and the PATH entry. The settings
and the log stay unless removal is asked for explicitly.

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

The log grows without limit. The settings carry a maximum line count; nothing
acts on it.

- **Rather than:** trimming the log as it grows.
- **Gains:** each correction is one append with nothing read back.
- **Costs:** a setting that does nothing; a log that is never trimmed.

## Engineering

### Go, standard library only, no C

The program is one Go binary using only the standard library, with no C code.

- **Rather than:** third-party packages; cgo.
- **Gains:** a single file with nothing beside it to fail to load.
- **Costs:** Go's race detector needs cgo, so it cannot run here. It is
  absent rather than offered as a switch that could only fail; the one lock in
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

### Rules held by tests, not by habit

The shape of the code is checked by the suite: a size cap on every Go file,
tests included, with a band just below it that also fails; the package
boundaries; the purity of the engine. Tests write to temporary folders and
call the code directly, with no mocking library; the PATH check runs against
a real folder holding a real script shim.

- **Rather than:** conventions kept by review; mocks standing in for the
  filesystem and PATH.
- **Gains:** a file that has to be split is split at a real seam rather than
  shaved; a passing test means the real thing works.
- **Costs:** more, smaller files; the PATH tests change settings for the
  whole process, so they run alone.

### A coverage floor at the level the suite holds

The coverage gate fails below the level the suite already held when it was
set, not below a round target.

- **Rather than:** a round aspirational figure.
- **Gains:** the gate is never lowered to let a run pass; it is raised when
  the suite earns it.
- **Costs:** the commands that write to a real user's profile stay partly
  untested.

### One build script, run where the users are

Build, test, lint, coverage and clean are one PowerShell script; an unknown
switch is an error rather than being ignored. Go source is checked out
with LF line endings whatever the local git settings, so the formatter agrees
with a Windows checkout.

- **Rather than:** a second workflow for a platform the tool's users do not
  use; the Windows default of CRLF.
- **Gains:** one runner, runnable where the work happens; a mistyped switch
  cannot run a plain build and report success.
- **Costs:** the lint step runs the latest staticcheck, so its version is not
  pinned; a working tree cloned before the line-ending rule needs a one-time
  rewrite.

### One home for the version

The version lives in one file. The build writes it into the binary at link
time and stamps it into the site, refusing anything that is not a plain
three-part version; the repository's documents carry no version at all. The
site's stylesheet links also carry a hash of their content.

- **Rather than:** version strings edited by hand in several places.
- **Gains:** a release cannot ship a site naming the previous version; a
  browser fetches a changed stylesheet at once rather than a cached one.
- **Costs:** the site carries stamp markers and must be stamped from the
  source.

### GPL-3.0

CommandFixer is released under the GPL, version 3, with a commercial licence
available separately.

- **Rather than:** a permissive licence.
- **Gains:** changes stay open; a closed-source user pays for the right.
- **Costs:** none recorded.
