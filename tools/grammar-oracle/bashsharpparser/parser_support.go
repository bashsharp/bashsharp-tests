// Parser predicates of the Bash# mixed-lane grammar (Sprint 384, #1586).
// Hand-written; generate.sh never touches it. grammar/bashsharp/README.md
// lists them with the plan R6 positions they decide.

package bashsharpparser

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

// SetBashSharp selects the dialect: Bash# on (sh/syntax LangBashPP) or off
// (LangBash, as bash -n).
func (p *BashSharpParser) SetBashSharp(on bool) { p.bashsharp = on }

func (p *BashSharpParser) lt(k int) antlr.Token { return p.GetTokenStream().LT(k) }

func (p *BashSharpParser) identIs(s string) bool {
	t := p.lt(1)
	return t.GetTokenType() == BashSharpParserIDENTIFIER && t.GetText() == s
}

// embedLineHere: `embed FORMAT "./path" [as ALIAS] [!RUNNER]` followed by the
// end of the command (README P12, transcribed from the delta embed_line).
func (p *BashSharpParser) embedLineHere() bool {
	if t := p.lt(1); !p.bashsharp || t.GetTokenType() != BashSharpParserWORD || t.GetText() != "embed" {
		return false
	}
	var words []string
	for k := 2; ; k++ {
		t := p.lt(k)
		if !IsWordType(t.GetTokenType()) {
			switch t.GetTokenType() {
			case BashSharpParserSH_NEWLINE, BashSharpParserSH_SEMI, BashSharpParserSH_AMP, BashSharpParserRBRACE,
				BashSharpParserREGION_END, antlr.TokenEOF:
			default:
				return false
			}
			break
		}
		words = append(words, t.GetText())
	}
	if len(words) < 2 || !validFormat(words[0]) || !validEmbedPath(words[1]) {
		return false
	}
	return validFenceTail(words[0], words[2:])
}

func validFormat(f string) bool {
	if f == "" || !(f[0] == '_' || isASCIILetter(f[0])) {
		return false
	}
	for i := 1; i < len(f); i++ {
		if c := f[i]; !(c == '-' || c == '+' || c == '_' || c >= '0' && c <= '9' || isASCIILetter(c)) {
			return false
		}
	}
	return true
}

func validEmbedPath(w string) bool {
	if len(w) < 2 || w[0] != '"' || w[len(w)-1] != '"' {
		return false
	}
	path := w[1 : len(w)-1]
	if strings.ContainsAny(path, "\"\\\n") {
		return false
	}
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}

var condUnary = map[string]bool{
	"-a": true, "-b": true, "-c": true, "-d": true, "-e": true, "-f": true, "-g": true, "-h": true,
	"-k": true, "-p": true, "-r": true, "-s": true, "-t": true, "-u": true, "-w": true, "-x": true,
	"-G": true, "-L": true, "-N": true, "-O": true, "-S": true, "-z": true, "-n": true, "-o": true,
	"-v": true, "-R": true,
}

var condBinary = map[string]bool{
	"==": true, "=": true, "!=": true, "=~": true, "-eq": true, "-ne": true, "-lt": true, "-le": true,
	"-gt": true, "-ge": true, "-nt": true, "-ot": true, "-ef": true,
}

// condUnaryHere: a [[ ]] unary operator followed by its operand.
func (p *BashSharpParser) condUnaryHere() bool {
	t := p.lt(1)
	if t.GetTokenType() != BashSharpParserWORD || !condUnary[t.GetText()] {
		return false
	}
	n := p.lt(2)
	return IsWordType(n.GetTokenType()) && n.GetTokenType() != BashSharpParserW_DRBRACK
}

// condUnaryWord: LT(1) is spelled as a unary operator.
func (p *BashSharpParser) condUnaryWord() bool {
	t := p.lt(1)
	return t.GetTokenType() == BashSharpParserWORD && condUnary[t.GetText()]
}

func (p *BashSharpParser) condBinaryHere() bool {
	t := p.lt(1)
	return (t.GetTokenType() == BashSharpParserWORD || t.GetTokenType() == BashSharpParserASSIGNMENT_WORD) && condBinary[t.GetText()]
}
