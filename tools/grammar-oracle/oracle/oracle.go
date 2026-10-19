// Package oracle is the independent grammar-oracle recognizer (Sprint 384,
// #1586). It imports only the generated goparser and bashsharpparser packages
// and the ANTLR Go runtime, never sh/syntax, sh/gosource or go/parser;
// oracle_test.go enforces that with go list -deps.
//
// Accept means zero lexer and zero parser error events, with the entry rule
// ending at EOF. Any event, including one the default strategy recovered from,
// is a reject. The runtime's BailErrorStrategy (antlr4-go v4.13.1) drops the
// position of the error it bails on, so the oracle records every event of the
// default strategy instead. The first event is the first error.
package oracle

import (
	"fmt"
	"unicode/utf8"

	"bashpp-tests/tools/grammar-oracle/goparser"

	"github.com/antlr4-go/antlr/v4"
)

// Verdicts (docs/grammar-oracle/CONTRACT.md).
const (
	Accept = "accept"
	Reject = "reject"
	Error  = "error"
)

// Diagnostic is the CONTRACT.md position record: offset is a 0-based byte
// index into src, line is 1-based (LF only), col the 1-based byte column and
// char the 1-based code-point column.
type Diagnostic struct {
	Offset int    `json:"offset"`
	Line   int    `json:"line"`
	Col    int    `json:"col"`
	Char   int    `json:"char"`
	Msg    string `json:"msg"`
}

// Result is a verdict with its diagnostics in event order; Diagnostics[0] is
// the first error. An accept has none.
type Result struct {
	Verdict     string       `json:"verdict"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	// Boundary lists the extension sites of an accepted mixed-lane input
	// in source order (nil for the unit lane).
	Boundary []Site `json:"boundary,omitempty"`
}

// Lanes and their entries. The unit lane parses a Go source file
// (sourceFile); the mixed lane parses a script in a dialect: bash (Bash# off)
// or bashsharp.
var entries = map[string]map[string]bool{
	"unit":  {"sourceFile": true},
	"mixed": {"bash": true, "bashsharp": true},
}

// Recognize parses src as entry in lane.
func Recognize(lane, entry string, src []byte) (res Result) {
	if !entries[lane][entry] {
		return Result{Verdict: Error, Diagnostics: []Diagnostic{{Msg: fmt.Sprintf("unknown lane/entry %s/%s", lane, entry)}}}
	}
	if lane == "mixed" {
		defer func() {
			if r := recover(); r != nil {
				res = Result{Verdict: Error, Diagnostics: []Diagnostic{{Msg: fmt.Sprintf("panic: %v", r)}}}
			}
		}()
		return recognizeMixed(entry, src)
	}
	l := &listener{DefaultErrorListener: antlr.NewDefaultErrorListener(), pos: newPositions(src)}
	defer func() {
		if r := recover(); r != nil {
			res = Result{Verdict: Error, Diagnostics: append(l.diags, Diagnostic{Msg: fmt.Sprintf("panic: %v", r)})}
		}
	}()
	lex := goparser.NewGoLexer(antlr.NewInputStream(string(src)))
	lex.RemoveErrorListeners()
	lex.AddErrorListener(l)
	toks := antlr.NewCommonTokenStream(lex, antlr.TokenDefaultChannel)
	p := goparser.NewGoParser(toks)
	p.RemoveErrorListeners()
	p.AddErrorListener(l)
	p.SourceFile()
	if t := toks.LT(1); len(l.diags) == 0 && t.GetTokenType() != antlr.TokenEOF {
		l.SyntaxError(p, t, t.GetLine(), t.GetColumn(), "input not consumed to EOF", nil)
	}
	if len(l.diags) > 0 {
		return Result{Verdict: Reject, Diagnostics: l.diags}
	}
	return Result{Verdict: Accept, Diagnostics: []Diagnostic{}}
}

type listener struct {
	*antlr.DefaultErrorListener
	pos   positions
	diags []Diagnostic
}

func (l *listener) SyntaxError(_ antlr.Recognizer, _ any, line, column int, msg string, _ antlr.RecognitionException) {
	d := l.pos.at(line, column)
	d.Msg = msg
	l.diags = append(l.diags, d)
}

// positions maps ANTLR's (line, code-point column) back to byte offsets. The
// runtime decodes the input with []rune(string), so an invalid UTF-8 byte is
// one code point, exactly as utf8.DecodeRune counts it.
type positions struct {
	src        []byte
	lineStarts []int // byte offset of each line (index 0 = line 1)
}

func newPositions(src []byte) positions {
	p := positions{src: src, lineStarts: []int{0}}
	for i, b := range src {
		if b == '\n' {
			p.lineStarts = append(p.lineStarts, i+1)
		}
	}
	return p
}

func (p positions) at(line, column int) Diagnostic {
	if line < 1 {
		line = 1
	}
	if line > len(p.lineStarts) {
		line = len(p.lineStarts)
	}
	off := p.lineStarts[line-1]
	for n := 0; n < column && off < len(p.src) && p.src[off] != '\n'; n++ {
		_, w := utf8.DecodeRune(p.src[off:])
		off += w
	}
	start := p.lineStarts[line-1]
	return Diagnostic{Offset: off, Line: line, Col: off - start + 1, Char: utf8.RuneCount(p.src[start:off]) + 1}
}
