#include "go_asm.h"
#define RODATA	8

DATA ·answerAsm(SB)/8, $const_answer
GLOBL ·answerAsm(SB),RODATA,$8

DATA ·pairSize(SB)/8, $pair__size
GLOBL ·pairSize(SB),RODATA,$8

DATA ·pairB(SB)/8, $pair_b
GLOBL ·pairB(SB),RODATA,$8
