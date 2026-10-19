// isTypeArgument/isExpressionArgument: new(expr) is Go 1.26 syntax; make and
// new still take types; a method named make is not the builtin.
package p

type S struct{}

func (S) make(n int) int { return n }
func (S) new(n int) int  { return n }

var a = new(42)
var b = new(int)
var c = make([]int, 3)
var d = make(map[string]int)
var e = S{}.make(1)
var f = S{}.new(2)
var g = new(S{})
