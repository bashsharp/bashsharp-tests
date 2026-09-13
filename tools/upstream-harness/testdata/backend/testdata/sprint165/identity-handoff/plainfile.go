// errorcheck

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// Canary for the `-p=p` handoff on an ordinary errorcheck root: no internal
// import, one ordinary diagnostic. Both modes must still report it exactly.

package p

var x int = "s" // ERROR "cannot use .s. \(untyped string constant\) as int value in variable declaration"

var _ = x
