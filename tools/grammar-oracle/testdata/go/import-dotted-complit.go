// isOperand/addImportSpec: "gopkg.in/yaml.v3" is recorded as "v3"; the
// package name is yaml. Syntax only: the module need not exist.
package p

import "gopkg.in/yaml.v3"

var n = yaml.Node{Kind: yaml.ScalarNode}
