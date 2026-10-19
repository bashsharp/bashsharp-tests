// reject: a composite literal of a bare type name in an if header needs
// parentheses (spec: Composite literals, parsing ambiguity).
package p

type T struct{}

func f(x T) {
	if x == T{} {
	}
}
