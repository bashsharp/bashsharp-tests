// isOperand/addImportSpec: the import table records "v2", not the package
// name rand, so rand.PCG{} has only the methodExpr alternative upstream.
package p

import "math/rand/v2"

var g = rand.PCG{}
var h = &rand.ChaCha8{}
