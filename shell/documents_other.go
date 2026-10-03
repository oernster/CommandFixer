//go:build !windows

package shell

import "errors"

// errNoKnownFolders is why the lookup fails off Windows: there is no
// known-folder database to ask, so documentsDir falls back to home\Documents.
var errNoKnownFolders = errors.New("known folders exist only on Windows")

// knownDocumentsFolder has nothing to ask off Windows.
func knownDocumentsFolder() (string, error) {
	return "", errNoKnownFolders
}
