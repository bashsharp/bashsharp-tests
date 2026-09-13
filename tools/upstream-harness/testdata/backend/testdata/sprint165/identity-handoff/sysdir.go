// rundir

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// intrinsic's shape: a single-package directory program importing
// internal/runtime/sys. Upstream compiles sysdir.dir/main.go with
// `-D test -p=main`, links it and runs it. The seam must check/transpile
// under that identity (admitted) and remember it for the execute phase
// of this single-package program too (request (b)): the interpreted run
// carries --go-import-base/--go-import-path exactly like a multi-package
// one. The compiled artifact's output is compared with sysdir.out.

package ignored
