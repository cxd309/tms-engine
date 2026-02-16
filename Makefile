.PHONY: all cli wasm test test-python clean

BINARY   := dist/tms-engine
WASM_OUT := dist/sim.wasm

all: cli

## cli: build the CLI binary for the current platform
cli:
	mkdir -p dist
	go build -o $(BINARY) ./cmd/cli

## binaries: cross-compile the CLI binary for all supported platforms
binaries:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -o dist/tms-engine-linux-amd64   ./cmd/cli
	GOOS=linux   GOARCH=arm64 go build -o dist/tms-engine-linux-arm64   ./cmd/cli
	GOOS=darwin  GOARCH=amd64 go build -o dist/tms-engine-darwin-amd64  ./cmd/cli
	GOOS=darwin  GOARCH=arm64 go build -o dist/tms-engine-darwin-arm64  ./cmd/cli
	GOOS=windows GOARCH=amd64 go build -o dist/tms-engine-windows-amd64.exe ./cmd/cli

## wasm: compile the engine to WebAssembly for browser use
wasm:
	mkdir -p dist
	GOOS=js GOARCH=wasm go build -o $(WASM_OUT) ./cmd/wasm
	@echo "Copy the JS glue file from your Go installation:"
	@echo "  cp \$$(go env GOROOT)/misc/wasm/wasm_exec.js dist/"

## test: run all Go unit tests
test:
	go test ./...

## test-python: run all Python tests (unit + e2e; requires make cli first)
test-python:
	PATH="$(PWD)/dist:$(PATH)" uv run --project pytms pytest -v pytms/tests

## clean: remove build artefacts
clean:
	rm -rf dist/
