// Aliases, including generic aliases (aliasDecl typeParameters?), as
// composite-literal types, conversions and method-expression receivers.
package p

import "time"

type M = map[string]int
type L[T any] = []T
type D = time.Duration
type S struct{ X int }
type A = S

func (S) Get() int { return 0 }

var m = M{"a": 1}
var l = L[int]{1, 2}
var d = D(3)
var c = L[string](nil)
var a = A{X: 1}
var f = A.Get
var u = time.Duration.String
