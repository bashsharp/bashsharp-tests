// buildrundir

//go:build amd64

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// Positive control for the D3(a) assembly-companion seam: a body-less Go
// declaration implemented in the directory's .s file (retjmp/linknameasm
// shape). The transpiled Go is compiled with -symabis, the assembly is
// assembled natively, and the linked program must run silently.

package ignored
