package main

const answer = 42

type pair struct {
	a int64
	b int64
}

var (
	answerAsm int64
	pairSize  int64
	pairB     int64
)

func main() {
	if answerAsm != answer {
		println("const_answer =", answerAsm, "want", answer)
		panic("asmhdr constant")
	}
	if pairSize != 16 || pairB != 8 {
		println("pair__size =", pairSize, "pair_b =", pairB)
		panic("asmhdr layout")
	}
}
