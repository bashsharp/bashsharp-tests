// reject: a bare TypeName composite literal in a for range header
// (spec: Composite literals, parsing ambiguity).
package p

type T []int

func f() {
	for range T{1} {
	}
}
