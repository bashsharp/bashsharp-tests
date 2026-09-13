package main

func f() int64

func main() {
	if f() != 42 {
		panic("f")
	}
}
