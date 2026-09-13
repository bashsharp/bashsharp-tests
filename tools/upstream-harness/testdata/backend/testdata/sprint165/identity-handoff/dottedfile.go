// errorcheck -p=example.com/dotted

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// Negative: the recipe's own `-p=example.com/dotted` follows upstream's
// `-p=p` in the compile argv, so the identity gc compiles under is the
// dotted one — a user identity, never standard, never inside internal/.
// The seam must hand exactly that identity (the last -p, as the compiler's
// flag parsing reads it) and the checker must refuse the import with gc's
// wording at the import line, in both modes. The refusal is the expected
// diagnostic: a missing, extra or reworded refusal fails upstream's own
// errorCheck.

package p

import "internal/runtime/atomic" // ERROR "use of internal package internal/runtime/atomic not allowed"

func Load(addr *uint32) uint32 { return atomic.Load(addr) }
