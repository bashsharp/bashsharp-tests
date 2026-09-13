// compile

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// Positive control for the single-file identity handoff (D8): upstream's
// compileFile compiles this file with `-p=p`; the seam hands that identity
// to Bash++ as --go-import-path, and a standard identity may import a
// top-level internal package. Both modes must reach the checker and be
// admitted (check-only / transpile-compile-only, exit 0).

package p

import "internal/runtime/atomic"

func Load(addr *uint32) uint32 { return atomic.Load(addr) }

func Store(addr *uint32, v uint32) { atomic.Store(addr, v) }
