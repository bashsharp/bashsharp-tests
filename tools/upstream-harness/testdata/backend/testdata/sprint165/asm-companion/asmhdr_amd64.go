// buildrundir

//go:build amd64

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// The asmhdr shape: the assembly reads the compiler's -asmhdr output
// (go_asm.h) for a constant and a struct layout of the generated file. The
// seam keeps -asmhdr on the direct compile; whether the generated file
// preserves the names and layout is the lowering's fidelity, recorded as a
// product verdict here, never adjusted by the seam.

package ignored
