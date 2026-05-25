# CommandFixer

A lightweight Go binary that auto-corrects your common typing mistakes in PowerShell before commands execute.

Type `git sattus`, press Enter, and CommandFixer silently swaps it for `git status` - then runs it, if you approve.

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
- Adds that directory to your `$PATH`
- Copies `config.example.json` to `%USERPROFILE%\.typo-fixer\config.json`
- Hooks into your PowerShell profile

Then **restart PowerShell**.

### 3. Configure

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

### 4. Use It

Type any misconfigured command and press Enter:

```
PS> git sattus
CommandFixer: 'git sattus' -> 'git status'
On branch main ...
```

---

## How It Works

CommandFixer hooks into **PSReadLine** (already built into PowerShell 7) and intercepts the Enter key. When you press Enter:

1. PSReadLine captures the current buffer.
2. It calls `commandfixer correct <your-command>`.
3. CommandFixer loads `config.json`, applies all rules, and prints the corrected form.
4. If the command changed, PowerShell shows you what was fixed and replaces the buffer.
5. The corrected command executes.

No system-wide keyboard hooks. No persistent service required. The binary runs in milliseconds.

---

## CLI Commands

```
commandfixer correct <cmd>       Check and print the corrected form
commandfixer install [profile]   Add PSReadLine hook to your profile
commandfixer uninstall [profile] Remove the hook from your profile
commandfixer stats               Show correction count and rule breakdown
commandfixer version             Print version
commandfixer help                Show help
```

### Manual correction check

```powershell
commandfixer correct "git sattus"
# Prints: git status
```

---

## Running as a Windows Startup Service

CommandFixer does **not** need a background service to function - it runs on demand (invoked by the PS hook). No service setup is required.

If you want the binary available system-wide without modifying PATH, you can copy it to a directory already on the system PATH (e.g., `C:\Windows\System32` - requires admin) or use the installer which adds `%LOCALAPPDATA%\CommandFixer` to your user PATH.

To auto-run something on startup (e.g., a future HTTP-mode variant), add a shortcut to `shell:startup` or create a scheduled task:

```powershell
$action  = New-ScheduledTaskAction -Execute "$env:LOCALAPPDATA\CommandFixer\commandfixer.exe" -Argument "service"
$trigger = New-ScheduledTaskTrigger -AtLogon
Register-ScheduledTask -TaskName "CommandFixer" -Action $action -Trigger $trigger -RunLevel Highest
```

---

## Files

| Path | Purpose |
|------|---------|
| `%USERPROFILE%\.typo-fixer\config.json` | Typo rules and settings |
| `%USERPROFILE%\.typo-fixer\corrections.log` | JSONL corrections log |
| `%LOCALAPPDATA%\CommandFixer\commandfixer.exe` | Installed binary |

---

## Further Reading

- [DEVELOPMENT.md](DEVELOPMENT.md) - build steps, local dev workflow, debugging
- [ARCHITECTURE.md](ARCHITECTURE.md) - system design, module breakdown, data flow
- [TESTING.md](TESTING.md) - testing strategy, coverage requirements, how to run tests
