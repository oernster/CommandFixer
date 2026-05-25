# CommandFixer Makefile
# Targets work on Linux/macOS; for Windows use build.ps1 or 'go build' directly.

BINARY      := commandfixer
BINARY_WIN  := commandfixer.exe
GOFLAGS     := -ldflags="-s -w"
COVER_FILE  := coverage.out
COVER_HTML  := coverage.html

.PHONY: all build build-windows test test-race coverage coverage-html lint clean install

## all: build and test
all: test build

## build: compile for the current OS
build:
	go build $(GOFLAGS) -o $(BINARY) .

## build-windows: cross-compile for Windows amd64
build-windows:
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -o $(BINARY_WIN) .

## test: run all tests
test:
	go test ./...

## test-race: run tests with race detector
test-race:
	go test -race ./...

## coverage: generate coverage profile
coverage:
	go test -coverprofile=$(COVER_FILE) -covermode=atomic ./...
	go tool cover -func=$(COVER_FILE)

## coverage-html: open coverage in browser
coverage-html: coverage
	go tool cover -html=$(COVER_FILE) -o $(COVER_HTML)
	@echo "Coverage report: $(COVER_HTML)"

## lint: run go vet
lint:
	go vet ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY_WIN) $(COVER_FILE) $(COVER_HTML)

## install: install binary to GOPATH/bin
install:
	go install .
