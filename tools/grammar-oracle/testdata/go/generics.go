// Generic declarations, instantiation, and instantiated composite literals,
// conversions and method expressions, plain and package-qualified.
package p

import (
	"math/rand/v2"
	"slices"
)

type List[T any] struct{ xs []T }

func (l *List[T]) Len() int { return len(l.xs) }

type Pair[K comparable, V any] struct {
	K K
	V V
}

func Map[T, U any](xs []T, f func(T) U) []U {
	var out []U
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

var l = List[int]{}
var p = Pair[string, int]{"a", 1}
var q = &Pair[string, List[int]]{}
var n = rand.N[int64](10)
var s = slices.Collect[int]
var lenOf = (*List[int]).Len
var ys = Map[int, string]([]int{1}, func(int) string { return "" })
var conv = List[int](List[int]{})
