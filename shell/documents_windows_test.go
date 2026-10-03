//go:build windows

package shell

import (
	"os"
	"path/filepath"
	"testing"
)

// The real known-folder call, on the machine running the tests. It cannot show
// a moved Documents folder unless this machine has one; documents_test.go
// covers that case with the folder handed in.

func TestKnownDocumentsFolder_IsAnExistingDirectory(t *testing.T) {
	t.Parallel()
	dir, err := knownDocumentsFolder()
	if err != nil {
		t.Fatalf("knownDocumentsFolder: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatalf("expected an absolute path, got %q", dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %q: %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", dir)
	}
}

func TestDefaultProfilePath_UsesTheKnownFolder(t *testing.T) {
	t.Parallel()
	dir, err := knownDocumentsFolder()
	if err != nil {
		t.Fatalf("knownDocumentsFolder: %v", err)
	}
	got, err := DefaultProfilePath()
	if err != nil {
		t.Fatalf("DefaultProfilePath: %v", err)
	}
	if want := profilePathUnder(dir, pwshProfileDir); got != want {
		t.Errorf("got %q, want %q under the known Documents folder", got, want)
	}
}

func TestUTF16PtrToString_Nil(t *testing.T) {
	t.Parallel()
	if got := utf16PtrToString(nil); got != "" {
		t.Errorf("expected empty string for nil, got %q", got)
	}
}
