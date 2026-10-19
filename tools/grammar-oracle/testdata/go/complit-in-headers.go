// accept: composite literals in if/for/switch headers that the spec admits:
// enclosed in parentheses, brackets or braces, a non-TypeName literal type,
// a function literal body, case clauses and the statement bodies.
package p

type T struct{ a int }

type G[E any] struct{ e E }

func f(x T, m map[T]bool, s []int) {
	if x == (T{}) {
	}
	if f2(T{}) {
	}
	if m[T{}] {
	}
	if (G[int]{}) == (G[int]{}) {
	}
	for _, v := range []T{{1}, {2}} {
		_ = v
	}
	for i := range [2]int{} {
		_ = i
	}
	if v := (struct{ a int }{}); v.a == 0 {
	}
	if g := func() T { return T{} }; g() == x {
	}
	switch y := map[string]int{"a": 1}; y["a"] {
	case len([]T{{}}):
	}
	switch x {
	case T{}:
		_ = T{a: 1}
	}
	if x == x {
		_ = T{}
	} else if (x == T{}) {
	}
	select {
	case s[0] = <-make(chan int):
		_ = T{}
	}
	_ = s
}

func f2(T) bool { return true }
