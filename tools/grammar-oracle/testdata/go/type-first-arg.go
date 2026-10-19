// The spec's Arguments production admits a Type first argument in any call;
// gcsyntax accepts it, and only type checking refuses these calls.
package p

func f(...any) {}

func g() {
	f([]int)
	f(map[string]int, 1)
	f(chan int)
}
