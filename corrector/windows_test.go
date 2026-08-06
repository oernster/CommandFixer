package corrector

import "testing"

// Tests for correction over the Windows entries in the command database.
//
// Two kinds, and the difference matters: the tools with subcommand structure
// (winget, choco, net, reg and so on) go through subcommand correction, while
// the standalone commands (dir, mkdir, ipconfig) have the command name itself
// corrected and every remaining argument preserved.

// ---------------------------------------------------------------------------
// Suggest - Windows subcommand tools (commandDB entries)
// ---------------------------------------------------------------------------

func TestSuggest_Winget_Install_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("winget instal")
	if !found {
		t.Fatal("expected found=true for 'winget instal'")
	}
	if result != "winget install" {
		t.Errorf("expected %q, got %q", "winget install", result)
	}
}

func TestSuggest_Winget_ExactSubcommand_NoCorrection(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("winget install")
	if found {
		t.Error("expected found=false when subcommand is already exact")
	}
	if result != "winget install" {
		t.Errorf("expected input unchanged, got %q", result)
	}
}

func TestSuggest_Choco_Install_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("choco instal")
	if !found {
		t.Fatal("expected found=true for 'choco instal'")
	}
	if result != "choco install" {
		t.Errorf("expected %q, got %q", "choco install", result)
	}
}

func TestSuggest_Scoop_Install_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("scoop instal")
	if !found {
		t.Fatal("expected found=true for 'scoop instal'")
	}
	if result != "scoop install" {
		t.Errorf("expected %q, got %q", "scoop install", result)
	}
}

func TestSuggest_Net_Start_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("net stat")
	if !found {
		t.Fatal("expected found=true for 'net stat'")
	}
	if result != "net start" {
		t.Errorf("expected %q, got %q", "net start", result)
	}
}

func TestSuggest_Sc_Query_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("sc queyr")
	if !found {
		t.Fatal("expected found=true for 'sc queyr'")
	}
	if result != "sc query" {
		t.Errorf("expected %q, got %q", "sc query", result)
	}
}

func TestSuggest_Reg_Query_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("reg queyr")
	if !found {
		t.Fatal("expected found=true for 'reg queyr'")
	}
	if result != "reg query" {
		t.Errorf("expected %q, got %q", "reg query", result)
	}
}

func TestSuggest_Netsh_Interface_Typo(t *testing.T) {
	t.Parallel()
	e := New(0)
	result, found := e.Suggest("netsh interfce")
	if !found {
		t.Fatal("expected found=true for 'netsh interfce'")
	}
	if result != "netsh interface" {
		t.Errorf("expected %q, got %q", "netsh interface", result)
	}
}

// ---------------------------------------------------------------------------
// suggestStandalone - Windows standalone command correction
// ---------------------------------------------------------------------------

func TestSuggest_Standalone_ExactCommand_NoCorrection(t *testing.T) {
	t.Parallel()
	// "dir" is already an exact standalone command.
	e := New(0)
	result, found := e.Suggest("dir C:\\Users")
	if found {
		t.Error("expected found=false for exact standalone command")
	}
	if result != "dir C:\\Users" {
		t.Errorf("expected input unchanged, got %q", result)
	}
}

func TestSuggest_Standalone_Dir_Typo(t *testing.T) {
	t.Parallel()
	// "dirs" has edit distance 1 from "dir" (extra 's'): sim = 0.75 > threshold.
	e := New(0)
	result, found := e.Suggest("dirs C:\\Users")
	if !found {
		t.Fatal("expected found=true for 'dirs C:\\Users'")
	}
	if result != "dir C:\\Users" {
		t.Errorf("expected %q, got %q", "dir C:\\Users", result)
	}
}

func TestSuggest_Standalone_Mkdir_Typo(t *testing.T) {
	t.Parallel()
	// "mkdri" is a transposition of "mkdir".
	e := New(0)
	result, found := e.Suggest("mkdri newfolder")
	if !found {
		t.Fatal("expected found=true for 'mkdri newfolder'")
	}
	if result != "mkdir newfolder" {
		t.Errorf("expected %q, got %q", "mkdir newfolder", result)
	}
}

func TestSuggest_Standalone_Copy_Typo(t *testing.T) {
	t.Parallel()
	// "coppy" has edit distance 1 from "copy" (double 'p'): sim = 0.8 > threshold.
	e := New(0)
	result, found := e.Suggest("coppy file1.txt file2.txt")
	if !found {
		t.Fatal("expected found=true for 'coppy file1.txt file2.txt'")
	}
	if result != "copy file1.txt file2.txt" {
		t.Errorf("expected %q, got %q", "copy file1.txt file2.txt", result)
	}
}

func TestSuggest_Standalone_PreservesAllArgs(t *testing.T) {
	t.Parallel()
	// Extra args beyond the command name must be preserved verbatim.
	// "dirs" is distance 1 from "dir"; sim = 0.75 > threshold.
	e := New(0)
	result, found := e.Suggest("dirs /b /s C:\\Users")
	if !found {
		t.Fatal("expected found=true")
	}
	if result != "dir /b /s C:\\Users" {
		t.Errorf("expected %q, got %q", "dir /b /s C:\\Users", result)
	}
}

func TestSuggest_Standalone_TooFarOff_NoCorrection(t *testing.T) {
	t.Parallel()
	// "zyxwvu" is not similar to any standalone command.
	e := New(0)
	_, found := e.Suggest("zyxwvu somepath")
	if found {
		t.Error("expected found=false for completely unknown first token")
	}
}

func TestSuggest_Standalone_BelowThreshold_NoCorrection(t *testing.T) {
	t.Parallel()
	// With a very high threshold, even a close match should not fire.
	e := New(1.0)
	_, found := e.Suggest("dri C:\\Users")
	if found {
		t.Error("expected found=false when similarity is below threshold 1.0")
	}
}

func TestSuggest_Standalone_Ipconfig_Typo(t *testing.T) {
	t.Parallel()
	// "ipconifg" is a common transposition of "ipconfig".
	e := New(0)
	result, found := e.Suggest("ipconifg /all")
	if !found {
		t.Fatal("expected found=true for 'ipconifg /all'")
	}
	if result != "ipconfig /all" {
		t.Errorf("expected %q, got %q", "ipconfig /all", result)
	}
}

func TestSuggest_Standalone_Tasklist_Typo(t *testing.T) {
	t.Parallel()
	// "tasklit" is one deletion away from "tasklist".
	e := New(0)
	result, found := e.Suggest("tasklit /v")
	if !found {
		t.Fatal("expected found=true for 'tasklit /v'")
	}
	if result != "tasklist /v" {
		t.Errorf("expected %q, got %q", "tasklist /v", result)
	}
}

func TestSuggest_Ls_NeverSuggestsCls(t *testing.T) {
	t.Parallel()
	// "ls" is a valid PowerShell alias. It is one insertion from "cls" and must
	// never be rewritten to it.
	e := New(0)
	for _, in := range []string{"ls", "ls -la", "ls C:\\Users", "ls somefile.txt"} {
		result, found := e.Suggest(in)
		if found {
			t.Errorf("expected no suggestion for %q, got %q", in, result)
		}
		if result != in {
			t.Errorf("expected %q unchanged, got %q", in, result)
		}
	}
}

func TestSuggest_PowerShellAliases_NoCorrection(t *testing.T) {
	t.Parallel()
	e := New(0)
	for _, in := range []string{
		"cat file.txt", "cp a b", "mv a b", "rm file.txt",
		"pwd now", "ps -name foo", "clear screen",
	} {
		result, found := e.Suggest(in)
		if found {
			t.Errorf("expected no suggestion for %q, got %q", in, result)
		}
		if result != in {
			t.Errorf("expected %q unchanged, got %q", in, result)
		}
	}
}
