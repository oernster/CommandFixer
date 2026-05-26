# CommandFixer

A lightweight Go binary that auto-corrects your common typing mistakes in PowerShell before commands execute.

Type `git sattus`, press Enter, and CommandFixer silently swaps it for `git status` - then runs it, if you approve.

Works in both **PowerShell 7** (`pwsh`) and **Windows PowerShell 5** (`powershell.exe`).

---

## Quick Start

### 1. Build

```powershell
git clone https://github.com/oernster/commandfixer
cd commandfixer
.\build.ps1 -Test        # run tests then build
```

Or with `go` directly:

```powershell
go build -o commandfixer.exe .
```

### 2. Install

```powershell
.\install.ps1
```

The installer:
- Copies `commandfixer.exe` to `%LOCALAPPDATA%\CommandFixer\`
- Adds that directory to your user `PATH`
- Copies `config.example.json` to `%USERPROFILE%\.typo-fixer\config.json`
- Hooks into your PowerShell profile for both PS7 and PS5

Then **restart PowerShell** (both `pwsh` and `powershell.exe` will have the hook).

### 3. Uninstall

```powershell
.\uninstall.ps1
```

The uninstaller:
- Removes the hook from both PS7 (`Documents\PowerShell\profile.ps1`) and PS5 (`Documents\WindowsPowerShell\profile.ps1`)
- Removes the binary from `%LOCALAPPDATA%\CommandFixer\`
- Removes that directory from your user `PATH`
- Keeps your config and log at `%USERPROFILE%\.typo-fixer\` (data is yours)

To also delete config and log:

```powershell
.\uninstall.ps1 -RemoveConfig
```

Then **restart PowerShell** to complete removal.

> You can also uninstall via the binary directly (if it is on your PATH):
> ```powershell
> commandfixer uninstall   # removes profile hooks only; does not touch binary or PATH
> ```

---

### 4. Configure

Edit `%USERPROFILE%\.typo-fixer\config.json`:

```json
{
  "typos": [
    { "from": "git sattus",  "to": "git status" },
    { "from": "git statsu",  "to": "git status" },
    { "from": "docker pss",  "to": "docker ps"  }
  ],
  "settings": {
    "show_corrections": true,
    "max_log_lines": 10000
  }
}
```

See `config.example.json` in this repo for a larger starter set.

### 5. Use It

Type any misconfigured command and press Enter:

```
PS> git sattus
CommandFixer: did you mean: git status [Y/n]
On branch main ...
```

---

## How It Works

CommandFixer hooks into **PSReadLine** (built into both PowerShell 7 and Windows PowerShell 5) and intercepts the Enter key. When you press Enter:

1. PSReadLine captures the current buffer.
2. It calls `commandfixer suggest <your-command>`.
3. CommandFixer loads `config.json`, applies all rules, and prints the corrected form.
4. If the command changed, PowerShell prompts you to confirm.
5. The corrected command executes.

No system-wide keyboard hooks. No persistent service required. The binary runs in milliseconds.

---

## CLI Commands

```
commandfixer suggest <cmd>       Fuzzy-match and print correction (used by hook)
commandfixer correct <cmd>       Like suggest but also logs the correction
commandfixer install [profile]   Add PSReadLine hook to your profile(s)
commandfixer uninstall [profile] Remove the hook from your profile(s)
commandfixer stats               Show correction count and rule breakdown
commandfixer version             Print version
commandfixer help                Show help
```

Without a `[profile]` argument, `install` and `uninstall` target both PS7 and PS5 profiles simultaneously.

### Manual correction check

```powershell
commandfixer correct "git sattus"
# Prints: git status
```

---

## Files

| Path | Purpose |
|------|---------|
| `%USERPROFILE%\.typo-fixer\config.json` | Typo rules and settings |
| `%USERPROFILE%\.typo-fixer\corrections.log` | JSONL corrections log |
| `%LOCALAPPDATA%\CommandFixer\commandfixer.exe` | Installed binary |
| `Documents\PowerShell\profile.ps1` | PS7 profile (hook appended here) |
| `Documents\WindowsPowerShell\profile.ps1` | PS5 profile (hook appended here) |

---

## Further Reading

- [DEVELOPMENT.md](DEVELOPMENT.md) - build steps, local dev workflow, debugging
- [ARCHITECTURE.md](ARCHITECTURE.md) - system design, module breakdown, data flow
- [TESTING.md](TESTING.md) - testing strategy, coverage requirements, how to run tests
