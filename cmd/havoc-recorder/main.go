// havoc-recorder consumes experiment results from Kafka, writes every
// result to Postgres as an immutable record, clears the matching Redis
// active-experiment lock, and emits structured JSON logs for ELK.
//
// Phase 1 scaffold — the recorder lands in Phase 3.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "havoc-recorder: not yet implemented (arrives in Phase 3)")
	os.Exit(1)
}
