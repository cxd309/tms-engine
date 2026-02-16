//go:build js && wasm

// Command wasm exposes the TMS engine to the browser via WebAssembly.
// After loading, it registers a global JavaScript function:
//
//	runSimulation(jsonString) -> jsonString
//
// The input and output are JSON-encoded SimulationInput and SimulationLog
// respectively, matching the same contract used by the CLI and Python wrapper.
package main

import (
	"syscall/js"

	"github.com/cxd309/tms-engine/internal/engine"
)

func main() {
	js.Global().Set("runSimulation", js.FuncOf(runSimulation))
	select {} // keep the WASM module alive until the page is closed
}

func runSimulation(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return map[string]any{"error": "no input provided"}
	}

	result, err := engine.RunJSON(args[0].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}
