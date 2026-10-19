// isMethodExpr: method expressions on local, pointer, qualified and
// instantiated receivers, and a local value whose name is not an import.
package p

import "time"

type T struct{ x int }

func (T) M(int) int { return 0 }
func (*T) P() int   { return 0 }

var a = T.M
var b = (*T).P
var c = time.Time.Unix
var d = (*time.Time).Unix
var e = T.M(T{}, 1)

func f(t T, err error) {
	_ = t.x
	_ = t.M
	_ = err.Error()
}
