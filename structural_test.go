package main

// Structural assertions over the shape of the repository rather than over what
// it computes. Nothing here exercises behaviour: these tests fail when the
// separation between packages erodes, which is a thing code review catches
// only while someone is looking.
//
// British spelling is used in comments.

import (
	"bufio"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lineCap is the size a file may reach and no further. It applies to test
// files exactly as to source: a 672-line test file is as hard to navigate as a
// 672-line implementation, and harder to be sure is complete.
const lineCap = 400

// dangerBandPercent is how far below the cap a file is already too close to it.
const dangerBandPercent = 5

// dangerBandFloor is the last comfortable size. Derived from the cap rather
// than written as a second literal, so the two can never drift apart: change
// the cap and the band follows.
const dangerBandFloor = lineCap - (lineCap * dangerBandPercent / 100)

// modulePath is the import prefix every package in this repository shares.
const modulePath = "github.com/oernster/commandfixer"

// leafPackages are the four packages main wires together. None may import
// another: main.go is the only place the wiring lives, and that is what lets
// each package be read and tested on its own.
var leafPackages = []string{"config", "corrector", "logger", "shell"}

// purePackage is the one whose simplicity is worth protecting by name.
const purePackage = "corrector"

// impureStdlib are the standard-library packages that reach outside the
// process: the filesystem, the environment, the network, the clock,
// randomness. The correction engine must import none of them. It is
// computation over strings, which is precisely why its tests are a long table
// of plain cases with no fixture to build; the moment it reaches for any of
// these, that goes and does not come back.
var impureStdlib = []string{
	"bufio",
	"database/sql",
	"encoding/json",
	"io",
	"io/fs",
	"io/ioutil",
	"log",
	"math/rand",
	"net",
	"net/http",
	"os",
	"os/exec",
	"path/filepath",
	"time",
}

// goFilesIn returns every Go file directly inside dir, test files included.
// Test files are held to the same rules: a package whose tests need the
// filesystem is a package that touches the filesystem.
func goFilesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	if len(files) == 0 {
		t.Fatalf("no Go files found in %s", dir)
	}
	return files
}

// importsOf returns the import paths of one Go file.
func importsOf(t *testing.T, path string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	paths := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		paths = append(paths, strings.Trim(spec.Path.Value, `"`))
	}
	return paths
}

// TestLeafPackagesDoNotImportEachOther holds main.go as the only wiring point.
//
// Without this, the cheapest way to reuse anything is a direct import between
// two packages, and the separation is gone before anyone notices it went.
func TestLeafPackagesDoNotImportEachOther(t *testing.T) {
	t.Parallel()
	for _, pkg := range leafPackages {
		for _, file := range goFilesIn(t, pkg) {
			for _, imported := range importsOf(t, file) {
				if strings.HasPrefix(imported, modulePath) {
					t.Errorf(
						"%s imports %s: only main.go may wire packages together",
						file, imported,
					)
				}
			}
		}
	}
}

// TestCorrectorStaysPureComputation holds the correction engine to strings in,
// strings out. This is the highest-value rule in the file: it is the reason
// the engine can be tested exhaustively without a single fixture.
func TestCorrectorStaysPureComputation(t *testing.T) {
	t.Parallel()
	forbidden := make(map[string]bool, len(impureStdlib))
	for _, pkg := range impureStdlib {
		forbidden[pkg] = true
	}
	for _, file := range goFilesIn(t, purePackage) {
		for _, imported := range importsOf(t, file) {
			if forbidden[imported] {
				t.Errorf(
					"%s imports %s: %s is pure computation over strings and"+
						" must not reach outside the process",
					file, imported, purePackage,
				)
			}
		}
	}
}

// TestEveryLeafPackageIsCovered guards the two tests above against quietly
// checking nothing: a package renamed or added without this list being updated
// would otherwise pass by not being looked at.
func TestEveryLeafPackageIsCovered(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading repository root: %v", err)
	}
	listed := make(map[string]bool, len(leafPackages))
	for _, pkg := range leafPackages {
		listed[pkg] = true
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if len(goFilesInOrNone(entry.Name())) > 0 && !listed[entry.Name()] {
			t.Errorf(
				"package %s exists but is not in leafPackages, so no"+
					" structural rule reaches it",
				entry.Name(),
			)
		}
	}
}

// countLines returns the number of lines in a file.
func countLines(t *testing.T, path string) int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), bufio.MaxScanTokenSize)
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return count
}

// everyGoFile walks the repository, skipping hidden directories.
func everyGoFile(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the repository: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no Go files found, so this test is checking nothing")
	}
	return files
}

// TestNoFileExceedsTheLineCap is the first half of the size rule.
func TestNoFileExceedsTheLineCap(t *testing.T) {
	t.Parallel()
	for _, file := range everyGoFile(t) {
		if lines := countLines(t, file); lines > lineCap {
			t.Errorf("%s is %d lines, over the cap of %d", file, lines, lineCap)
		}
	}
}

// TestNoFileSitsJustBelowTheLineCap is the second half, and the half that is
// easy to argue away.
//
// Shaving a file to 399 buys nothing: the next edit breaks it and the same
// file gets decomposed again, so the work is paid for twice and the reader
// still has a 399-line file in between. A file that has to be split is split
// properly, to a size with room in it.
//
// The cap itself stays legal, because 400 is the stated target rather than a
// number to creep up on. It is the approach to it that this refuses.
func TestNoFileSitsJustBelowTheLineCap(t *testing.T) {
	t.Parallel()
	for _, file := range everyGoFile(t) {
		lines := countLines(t, file)
		if lines > dangerBandFloor && lines < lineCap {
			t.Errorf(
				"%s is %d lines, inside the danger band %d to %d:"+
					" decompose it to a size with room in it, not to %d",
				file, lines, dangerBandFloor+1, lineCap-1, lineCap-1,
			)
		}
	}
}

// goFilesInOrNone answers whether a directory is a Go package, without failing
// the test for directories that are not (the site, the docs, build output).
func goFilesInOrNone(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, entry.Name())
		}
	}
	return files
}
