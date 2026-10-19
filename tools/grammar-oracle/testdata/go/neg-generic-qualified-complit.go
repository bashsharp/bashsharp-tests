// reject: unterminated instantiated composite literal behind a versioned import.
package p

import "math/rand/v2"

var a = rand.PCG{
