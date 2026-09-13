// run

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// Negative: a run root reaches the seam as `go run` — a native argv with no
// -p at all. The seam must invent no identity, so the checker keeps its
// directory rule and refuses the import with gc's wording; the root fails
// in both modes at the first phase.

package main

import (
	"fmt"
	"internal/runtime/atomic"
)

func main() {
	var v uint32
	atomic.Store(&v, 7)
	fmt.Println(atomic.Load(&v))
}
