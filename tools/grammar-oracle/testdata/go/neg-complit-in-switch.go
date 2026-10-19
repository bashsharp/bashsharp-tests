// reject: a generic TypeName composite literal in a switch
// header (spec: Composite literals, parsing ambiguity).
package p

type G[E any] struct{ e E }

func f() {
	switch x := G[int]{}; x.e {
	}
}
