package main

func f() int64
func g()

func main() {
	if f() != 42 {
		panic("f")
	}
	g()
}
