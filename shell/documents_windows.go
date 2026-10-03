//go:build windows

package shell

import (
	"fmt"
	"syscall"
	"unsafe"
)

// folderIDDocuments is FOLDERID_Documents from the Windows SDK's KnownFolders.h:
// {FDD39AD0-238F-46AF-ADB4-6C85480369C7}. It names the user's Documents folder
// wherever it currently lives, which is what PowerShell reads its profile from.
var folderIDDocuments = syscall.GUID{
	Data1: 0xFDD39AD0,
	Data2: 0x238F,
	Data3: 0x46AF,
	Data4: [8]byte{0xAD, 0xB4, 0x6C, 0x85, 0x48, 0x03, 0x69, 0xC7},
}

// Arguments and result of SHGetKnownFolderPath that are fixed by its contract.
const (
	knownFolderDefaultFlags = 0 // KF_FLAG_DEFAULT: the folder as it is now
	currentUserToken        = 0 // a null token means the calling user
	hresultOK               = 0 // S_OK
)

// Both DLLs are on the Windows KnownDLLs list, so the loader takes them from
// System32 and never from a directory an attacker could write to.
var (
	procSHGetKnownFolderPath = syscall.NewLazyDLL("shell32.dll").NewProc("SHGetKnownFolderPath")
	procCoTaskMemFree        = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
)

// knownDocumentsFolder asks Windows where the user's Documents folder is.
func knownDocumentsFolder() (string, error) {
	if err := procSHGetKnownFolderPath.Find(); err != nil {
		return "", fmt.Errorf("find SHGetKnownFolderPath: %w", err)
	}
	var raw *uint16
	hr, _, _ := procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderIDDocuments)),
		knownFolderDefaultFlags,
		currentUserToken,
		uintptr(unsafe.Pointer(&raw)),
	)
	// The caller frees the buffer whether the call succeeded or not.
	if raw != nil {
		defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(raw)))
	}
	if hr != hresultOK {
		return "", fmt.Errorf("SHGetKnownFolderPath: HRESULT %#x", hr)
	}
	return utf16PtrToString(raw), nil
}

// utf16PtrToString copies a NUL-terminated UTF-16 string out of memory Windows
// owns, so the copy outlives the buffer once it is freed.
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	length := 0
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; length++ {
		ptr = unsafe.Add(ptr, unsafe.Sizeof(*p))
	}
	return syscall.UTF16ToString(unsafe.Slice(p, length))
}
