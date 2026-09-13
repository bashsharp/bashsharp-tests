// func f() int64
TEXT ·f(SB),4,$0-8
	MOVQ	$42, ret+0(FP)
	RET
