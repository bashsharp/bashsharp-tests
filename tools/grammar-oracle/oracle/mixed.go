package oracle

import (
	"bytes"
	"sort"
	"strings"
	"unicode/utf8"

	"bashpp-tests/tools/grammar-oracle/bashsharpparser"

	"github.com/antlr4-go/antlr/v4"
)

// Site is one extension site of the mixed lane (CONTRACT.md "boundary"): its
// kind and its [start,end) byte span in the original input.
type Site struct {
	Kind  string `json:"kind"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// maxNest bounds the command-substitution re-entry depth (README P11).
const maxNest = 64

// mixed holds one mixed-lane recognition: the input as bytes and as the code
// points ANTLR reads, and the byte offset of every code point.
type mixed struct {
	bashsharp bool
	src       []byte
	runes     []rune
	offsets   []int // offsets[i] = byte offset of rune i; offsets[len(runes)] = len(src)
	diags     []Diagnostic
	sites     []Site
}

// recognizeMixed parses src with the combined grammar (entry rule `script`)
// in the Bash dialect (Bash# off) or the Bash# dialect.
func recognizeMixed(dialect string, src []byte) Result {
	m := &mixed{bashsharp: dialect == "bashsharp", src: src}
	for off := 0; off < len(src); {
		r, w := utf8.DecodeRune(src[off:])
		m.runes = append(m.runes, r)
		m.offsets = append(m.offsets, off)
		off += w
	}
	m.offsets = append(m.offsets, len(src))
	m.parse(m.runes, nil, 0, len(m.runes), 0)
	if len(m.diags) > 0 {
		return Result{Verdict: Reject, Diagnostics: m.diags, Boundary: []Site{}}
	}
	sort.SliceStable(m.sites, func(i, j int) bool { return m.sites[i].Start < m.sites[j].Start })
	if m.sites == nil {
		m.sites = []Site{}
	}
	return Result{Verdict: Accept, Diagnostics: []Diagnostic{}, Boundary: m.sites}
}

// parse recognizes data[start:end] (data is the input, or a copy whose prefix
// is the input and whose tail is an unescaped backquote body). back maps an
// index of data at or after start back to an input rune index (nil: identity).
func (m *mixed) parse(data []rune, back []int, start, end, nest int) {
	orig := func(i int) int {
		if back == nil || i < start {
			return i
		}
		if i-start < len(back) {
			return back[i-start]
		}
		return back[len(back)-1]
	}
	in := antlr.NewInputStream(string(data[:end]))
	in.Seek(start)
	lex := bashsharpparser.NewBashSharpLexer(in)
	lex.SetBashSharp(m.bashsharp)
	ll := &mixedListener{DefaultErrorListener: antlr.NewDefaultErrorListener(), m: m, orig: orig, lex: lex}
	lex.RemoveErrorListeners()
	lex.AddErrorListener(ll)
	toks := antlr.NewCommonTokenStream(lex, antlr.TokenDefaultChannel)
	p := bashsharpparser.NewBashSharpParser(toks)
	p.SetBashSharp(m.bashsharp)
	pl := &mixedListener{DefaultErrorListener: antlr.NewDefaultErrorListener(), m: m, orig: orig}
	p.RemoveErrorListeners()
	p.AddErrorListener(pl)
	tree := p.Script()
	if len(m.diags) > 0 {
		return
	}
	m.collectSites(tree, orig)
	// Command substitution bodies are parsed again as scripts (README P11).
	for _, t := range toks.GetAllTokens() {
		switch ty := t.GetTokenType(); {
		case bashsharpparser.IsWordType(ty), ty == bashsharpparser.BashSharpLexerSHELL_OPERAND,
			ty == bashsharpparser.BashSharpLexerARRAY_ASSIGN, ty == bashsharpparser.BashSharpLexerARITH_CMD:
		default:
			continue
		}
		for _, s := range substs(data[t.GetStart() : t.GetStop()+1]) {
			if nest >= maxNest {
				m.diags = append(m.diags, m.diagAt(orig(t.GetStart()), "command substitutions nested too deeply"))
				return
			}
			s0, s1 := t.GetStart()+s.start, t.GetStart()+s.end
			if !s.backquote {
				m.parse(data, back, s0, s1, nest+1)
			} else {
				body, idx := unescapeBackquote(data[s0:s1])
				buf := append(append([]rune{}, data[:s0]...), body...)
				bm := make([]int, len(body)+1)
				for i, k := range idx {
					bm[i] = orig(s0 + k)
				}
				bm[len(body)] = orig(s1)
				m.parse(buf, bm, s0, len(buf), nest+1)
			}
			if len(m.diags) > 0 {
				return
			}
		}
	}
}

// collectSites records the extension sites of a successful parse, keyed and
// anchored as the sh/syntax reference adapter keys them.
func (m *mixed) collectSites(tree antlr.Tree, orig func(int) int) {
	add := func(kind string, ctx antlr.ParserRuleContext) {
		start, stop := ctx.GetStart(), ctx.GetStop()
		end := stop.GetStop() + 1
		if kind == "fence" {
			// A fence ends with its closing tilde run, not the newline (or
			// "\r\n") after it.
			end -= len(stop.GetText()) - len(strings.TrimSuffix(strings.TrimSuffix(stop.GetText(), "\n"), "\r"))
		}
		m.sites = append(m.sites, Site{kind, m.offsets[orig(start.GetStart())], m.offsets[orig(end)]})
	}
	var walk func(t antlr.Tree)
	walk = func(t antlr.Tree) {
		switch c := t.(type) {
		case *bashsharpparser.FenceContext:
			add("fence", c)
		case *bashsharpparser.Embed_lineContext:
			add("embed-line", c)
		case *bashsharpparser.DecoratorContext:
			add("decorator", c)
		case *bashsharpparser.At_functionContext:
			add("at-function", c)
		case *bashsharpparser.Agentic_blockContext:
			add("agentic-block", c)
		case *bashsharpparser.Agentic_shell_funcContext:
			add("agentic-shell-func", c)
		case *bashsharpparser.Agentic_typed_funcContext:
			add("agentic-typed-func", c)
			walk(c.Typed_func().Block())
			return
		case *bashsharpparser.Agentic_typed_methodContext:
			add("agentic-typed-method", c)
			walk(c.Typed_method().Block())
			return
		case *bashsharpparser.Typed_funcContext:
			add("typed-func", c)
		case *bashsharpparser.Typed_methodContext:
			add("typed-method", c)
		}
		for i := 0; i < t.GetChildCount(); i++ {
			walk(t.GetChild(i))
		}
	}
	walk(tree)
}

type mixedListener struct {
	*antlr.DefaultErrorListener
	m    *mixed
	orig func(int) int
	lex  *bashsharpparser.BashSharpLexer
}

func (l *mixedListener) SyntaxError(_ antlr.Recognizer, sym any, _, _ int, msg string, _ antlr.RecognitionException) {
	idx := 0
	switch {
	case l.lex != nil:
		idx = l.lex.TokenStartCharIndex
	case sym != nil:
		if t, ok := sym.(antlr.Token); ok {
			idx = t.GetStart()
			if t.GetTokenType() == antlr.TokenEOF {
				idx = t.GetStart()
			}
		}
	}
	l.m.diags = append(l.m.diags, l.m.diagAt(l.orig(idx), msg))
}

// diagAt places a diagnostic at input rune index i (CONTRACT.md positions).
func (m *mixed) diagAt(i int, msg string) Diagnostic {
	i = min(max(i, 0), len(m.runes))
	off := m.offsets[i]
	start := bytes.LastIndexByte(m.src[:off], '\n') + 1
	return Diagnostic{
		Offset: off,
		Line:   bytes.Count(m.src[:off], []byte("\n")) + 1,
		Col:    off - start + 1,
		Char:   utf8.RuneCount(m.src[start:off]) + 1,
		Msg:    msg,
	}
}

type subst struct {
	start, end int // the body, relative to the token
	backquote  bool
}

// substs finds the command substitution bodies at the outer level of a word
// (README P11): $( … ), <( … ), >( … ) and backquotes, including those inside
// double quotes and ${ … }. Arithmetic $(( … )) and (( … )) bodies are not
// command text. The structure follows the WORD fragments of the lexer.
func substs(w []rune) []subst {
	s := &substScanner{w: w}
	s.scan(0, false, 0)
	return s.out
}

type substScanner struct {
	w   []rune
	out []subst
}

// scan walks word parts from i up to the rune stop (0: the end); dq is true
// inside double quotes. It returns the index of stop.
func (s *substScanner) scan(i int, dq bool, stop rune) int {
	w := s.w
	for i < len(w) {
		c := w[i]
		switch {
		case c == stop && stop != 0:
			return i
		case c == '\\':
			i += 2
		case c == '\'' && !dq:
			i = skipSQ(w, i)
		case c == '"' && !dq:
			i = s.scan(i+1, true, '"') + 1
		case c == '`':
			end := skipBQ(w, i)
			s.out = append(s.out, subst{start: i + 1, end: end - 1, backquote: true})
			i = end
		case c == '$':
			i = s.dollar(i)
		case (c == '<' || c == '>') && !dq && i+1 < len(w) && w[i+1] == '(':
			end := s.balanced(i + 2)
			s.out = append(s.out, subst{start: i + 2, end: end})
			i = end + 1
		default:
			i++
		}
	}
	return i
}

// dollar scans the expansion at w[i] == '$' and returns the index after it.
func (s *substScanner) dollar(i int) int {
	w := s.w
	if i+1 >= len(w) {
		return i + 1
	}
	switch w[i+1] {
	case '(':
		if i+2 < len(w) && w[i+2] == '(' {
			return s.arith(i+3) + 2 // $(( … ))
		}
		end := s.balanced(i + 2)
		s.out = append(s.out, subst{start: i + 2, end: end})
		return end + 1
	case '{':
		return s.scan(i+2, false, '}') + 1
	case '\'':
		j := i + 2
		for j < len(w) && w[j] != '\'' {
			if w[j] == '\\' {
				j++
			}
			j++
		}
		return j + 1
	}
	return i + 1
}

// balanced returns the index of the ")" that closes a command substitution
// body starting at i, skipping quotes and nested parentheses.
func (s *substScanner) balanced(i int) int {
	w := s.w
	for i < len(w) {
		switch w[i] {
		case '\\':
			i += 2
		case '\'':
			i = skipSQ(w, i)
		case '"':
			i = skipDQ(w, i)
		case '`':
			i = skipBQ(w, i)
		case '$':
			i = skipDollar(w, i)
		case '(':
			i = s.balanced(i+1) + 1
		case ')':
			return i
		default:
			i++
		}
	}
	return i
}

// arith returns the index of the "))" that closes an arithmetic body at i;
// substitutions inside it are still command text.
func (s *substScanner) arith(i int) int {
	w := s.w
	depth := 0
	for i < len(w) {
		switch c := w[i]; {
		case c == '\\':
			i += 2
			continue
		case c == '$':
			i = s.dollar(i)
			continue
		case c == '`':
			end := skipBQ(w, i)
			s.out = append(s.out, subst{start: i + 1, end: end - 1, backquote: true})
			i = end
			continue
		case c == '(':
			depth++
		case c == ')':
			if depth == 0 {
				return i
			}
			depth--
		}
		i++
	}
	return i
}

// skipDQ and skipDollar skip a part inside a substitution body, which its own
// parse examines.
func skipDQ(w []rune, i int) int {
	s := &substScanner{w: w}
	return s.scan(i+1, true, '"') + 1
}

func skipDollar(w []rune, i int) int {
	s := &substScanner{w: w}
	return s.dollar(i)
}

func skipSQ(w []rune, i int) int {
	j := i + 1
	for j < len(w) && w[j] != '\'' {
		j++
	}
	return j + 1
}

// skipBQ returns the index after the backquote that closes the one at i.
func skipBQ(w []rune, i int) int {
	j := i + 1
	for j < len(w) && w[j] != '`' {
		if w[j] == '\\' {
			j++
		}
		j++
	}
	return j + 1
}

// unescapeBackquote removes the backslash before \\, \` and \$ in a
// backquote body, as the shell does before parsing it, and returns the body
// with the index in the escaped body of each remaining rune.
func unescapeBackquote(b []rune) ([]rune, []int) {
	var out []rune
	var idx []int
	for i := 0; i < len(b); i++ {
		if b[i] == '\\' && i+1 < len(b) && (b[i+1] == '\\' || b[i+1] == '`' || b[i+1] == '$') {
			i++
		}
		out = append(out, b[i])
		idx = append(idx, i)
	}
	return out, idx
}
