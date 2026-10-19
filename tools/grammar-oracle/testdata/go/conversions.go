// isConversion: only non-identifier types take the conversion alternative;
// identifier and qualified types parse as operand + call.
package p

import "time"

type T int

var a = []byte("x")
var b = (*T)(nil)
var c = (func())(nil)
var d = chan int(nil)
var e = map[string]int(nil)
var f = interface{}(1)
var g = struct{}(struct{}{})
var h = T(1)
var i = time.Duration(1)
var j = [2]int([2]int{})
var k = (<-chan int)(nil)
