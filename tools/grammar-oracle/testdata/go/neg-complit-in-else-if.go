// reject: the else-if header is a header too
// (spec: Composite literals, parsing ambiguity).
package p

type T struct{}

func f(x T) {
	if false {
	} else if x == T{} {
	}
}
