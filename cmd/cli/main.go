// Command tms-engine reads a SimulationInput JSON from a file argument (or stdin),
// runs the simulation, and writes the SimulationLog JSON to stdout.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cxd309/tms-engine/internal/engine"
)

func main() {
	var (
		data []byte
		err  error
	)

	if len(os.Args) > 1 {
		data, err = os.ReadFile(os.Args[1])
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	result, err := engine.RunJSON(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "simulation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}
