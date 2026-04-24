// havoc-agent runs as a Kubernetes DaemonSet, consumes experiment commands
// from Kafka, filters for pods scheduled on its own node, executes the
// requested chaos action, and publishes the outcome.
//
// Phase 1 scaffold — the agent lands in Phase 4.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "havoc-agent: not yet implemented (arrives in Phase 4)")
	os.Exit(1)
}
