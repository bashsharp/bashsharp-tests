package main

func f() int64

func main() {
	if got := f(); got != 42 {
		println("f() =", got, "want 42")
		panic("assembly companion")
	}
}
