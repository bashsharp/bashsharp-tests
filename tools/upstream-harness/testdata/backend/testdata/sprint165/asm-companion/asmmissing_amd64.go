// buildrundir

//go:build amd64

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// Negative control: the Go side declares g without a body and no assembly
// implements it. The link phase must fail; the seam never turns a missing
// assembly body into a pass.

package ignored
