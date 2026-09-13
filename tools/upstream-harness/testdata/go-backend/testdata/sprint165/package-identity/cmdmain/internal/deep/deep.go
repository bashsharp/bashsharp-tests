// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// An internal package of the tested package itself: cmd/go grants it to
// example.com/pkgid/cmdmain (the parent of internal) and to the files of its
// test packages, never to example.com/pkgid/cmdmain.test.
package deep

// Word is what the tested package and its tests read through the internal
// import.
func Word() string { return "deep" }
