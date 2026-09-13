// errorcheck -0 -m -l

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// escape_runtime_atomic's shape: an errorcheck root with optimizer
// diagnostics importing internal/runtime/atomic, compiled by upstream with
// `-p=p`. Interpreted mode has no -m meaning (declared unsupported, as
// before); compiled mode must transpile under the identity `p`, be admitted,
// and the pinned compiler's escape diagnostics on the generated file must
// satisfy the expectations on the upstream lines.

package escape

import (
	"internal/runtime/atomic"
	"unsafe"
)

func Loadp(addr unsafe.Pointer) unsafe.Pointer { // ERROR "leaking param: addr( to result ~r0 level=1)?$"
	return atomic.Loadp(addr)
}

var ptr unsafe.Pointer

func Storep() {
	var x int // ERROR "moved to heap: x"
	atomic.StorepNoWB(unsafe.Pointer(&ptr), unsafe.Pointer(&x))
}

func Load(addr *uint32) uint32 { // ERROR "addr does not escape"
	return atomic.Load(addr)
}

func Sink() *int {
	x := new(int) // ERROR "new\(int\) escapes to heap"
	return x
}
