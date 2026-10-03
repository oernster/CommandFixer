package shell

import (
	"errors"
	"path/filepath"
	"testing"
)

// Tests for where the profiles are placed, with the Documents folder and the
// home directory handed in, so a moved Documents folder can be tested on a
// machine whose own folder has never moved.

// redirectedDocuments looks like OneDrive folder backup: Documents moved out of
// the home directory into a folder of another name.
const redirectedDocuments = `D:\OneDrive - Contoso\Dokumente`

const testHome = `C:\Users\someone`

var errLookup = errors.New("lookup failed")

func folder(path string) func() (string, error) {
	return func() (string, error) { return path, nil }
}

func failing() (string, error) { return "", errLookup }

func TestResolveDocumentsDir_PrefersTheKnownFolder(t *testing.T) {
	t.Parallel()
	got, err := resolveDocumentsDir(folder(redirectedDocuments), folder(testHome))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != redirectedDocuments {
		t.Errorf(
			"got %q, want the known folder %q: a hook written under the home"+
				" directory is never read once Documents has moved",
			got, redirectedDocuments,
		)
	}
}

func TestResolveDocumentsDir_FallsBackWhenTheLookupFails(t *testing.T) {
	t.Parallel()
	got, err := resolveDocumentsDir(failing, folder(testHome))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(testHome, documentsFallbackDir); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveDocumentsDir_FallsBackWhenTheLookupIsEmpty(t *testing.T) {
	t.Parallel()
	got, err := resolveDocumentsDir(folder(""), folder(testHome))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(testHome, documentsFallbackDir); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveDocumentsDir_HomeNotConsultedWhenTheLookupAnswers(t *testing.T) {
	t.Parallel()
	if _, err := resolveDocumentsDir(folder(redirectedDocuments), failing); err != nil {
		t.Errorf("home was consulted although the known folder answered: %v", err)
	}
}

func TestResolveDocumentsDir_BothFail(t *testing.T) {
	t.Parallel()
	_, err := resolveDocumentsDir(failing, failing)
	if !errors.Is(err, errLookup) {
		t.Errorf("expected the home directory error to be wrapped, got %v", err)
	}
}

func TestProfilePathUnder_RedirectedDocuments(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		pwshProfileDir:      filepath.Join(redirectedDocuments, "PowerShell", "profile.ps1"),
		windowsPSProfileDir: filepath.Join(redirectedDocuments, "WindowsPowerShell", "profile.ps1"),
	}
	for shellDir, want := range cases {
		if got := profilePathUnder(redirectedDocuments, shellDir); got != want {
			t.Errorf("profilePathUnder(%q): got %q, want %q", shellDir, got, want)
		}
	}
}
