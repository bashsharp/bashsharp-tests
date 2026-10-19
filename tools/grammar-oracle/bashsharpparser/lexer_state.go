// Contextual-scanning state of the Bash# mixed-lane lexer (Sprint 384,
// #1586). This file is hand-written; generate.sh never touches it. The
// grammar (grammar/bashsharp/BashSharpLexer.g4) names these predicates and
// actions, and grammar/bashsharp/README.md enumerates the primitives (P1-P12)
// they implement. It is transcribed from the permissive sh/syntax
// implementation (the start-site decision table in bashpp_startsites.go and
// the Go-region dispatch in parser.go), never from GNU Bash parse.y, and it
// imports nothing but the ANTLR runtime and the standard library.

package bashsharpparser

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
)

// SH frame kinds and Go region kinds (README P5).
const (
	shRoot       = iota // shell text of a script or substitution body
	shGoBlock           // the body of a Go statement block: BLOCK_OPEN … RBRACE
	shSwitchBody        // the body of a Go switch or select
	goDecl              // var, const, type, import, package
	goFunc              // func NAME(…, func (…)…: typed declarations, literal calls
	goSimple            // short declaration, call, assignment, send, receive
	goControl           // if, for, switch, select (inside a Go block only)
	goStmt              // return, defer, go
	goDecorator         // @name(…)
	goCase              // case/default head of a switch or select clause
)

// Open shell parentheses (README P3).
const (
	parenSubshell = iota
	parenOther
	parenPattern
)

// Case-statement positions (README P3).
const (
	caseSubject = iota
	caseIn
	casePattern
	caseBody
)

// Function-signature stages (README P4).
const (
	sigParams = iota
	sigInParams
	sigAfterParams
)

type funcSig struct{ depth, stage int }

type lexFrame struct {
	kind int

	// Shell frames.
	cmdPos     bool
	cases      []int
	forStage   int // 1 after for/select, 2 after its name, 3 in its word list
	funcName   bool
	cond       bool
	declArgs   bool
	redirect   bool
	parens     []int
	groups     int
	heredocOp  int
	afterArith bool

	// Go regions.
	paren, bracket, brace int
	header                bool
	headerKw              int
	litType               bool
	prev                  int
	prevText              string
	sigs                  []funcSig
	ntok                  int
}

func (f *lexFrame) goRegion() bool { return f.kind >= goDecl }

func (f *lexFrame) depth() int { return f.paren + f.bracket + f.brace }

type heredoc struct {
	delim string
	dash  bool
}

type lexState struct {
	mode      int   // the current lexer mode, mirrored (README P5)
	modes     []int // the mirrored mode stack
	bashsharp bool
	started   bool
	frames    []*lexFrame
	pending   []antlr.Token
	heredocs  []heredoc
	fence     string
	fenceLang string // "" for an aliased fence, which promotes no names
	fenceBody strings.Builder
	funcs     map[string]bool
}

// SetBashSharp selects the dialect: Bash# on (sh/syntax LangBashPP) or off
// (LangBash, as bash -n).
func (l *BashSharpLexer) SetBashSharp(on bool) { l.st.bashsharp = on }

// SetMode, PushMode and PopMode mirror the runtime's mode stack, which it
// does not expose; grammar commands reach them through the lexer interface.
func (l *BashSharpLexer) SetMode(m int) {
	l.st.mode = m
	l.BaseLexer.SetMode(m)
}

func (l *BashSharpLexer) PushMode(m int) {
	l.st.modes = append(l.st.modes, l.st.mode)
	l.st.mode = m
	l.BaseLexer.PushMode(m)
}

func (l *BashSharpLexer) PopMode() int {
	n := len(l.st.modes)
	l.st.mode = l.st.modes[n-1]
	l.st.modes = l.st.modes[:n-1]
	return l.BaseLexer.PopMode()
}

func (l *BashSharpLexer) top() *lexFrame { return l.st.frames[len(l.st.frames)-1] }

// shTop is the innermost shell frame.
func (l *BashSharpLexer) shTop() *lexFrame {
	for i := len(l.st.frames) - 1; i >= 0; i-- {
		if !l.st.frames[i].goRegion() {
			return l.st.frames[i]
		}
	}
	return l.st.frames[0]
}

// goBlockContext: a standalone "}" closes the innermost brace (README P5).
// True inside a Go statement block, outside any subshell opened there.
func goBlockContext(f *lexFrame) bool {
	if f.kind != shGoBlock && f.kind != shSwitchBody {
		return false
	}
	for _, p := range f.parens {
		if p == parenSubshell {
			return false
		}
	}
	return true
}

// NextToken runs the base lexer between the enumerated state transitions:
// Go-region entry at a command position (README P5), region ends that
// consume no input (P6), and the bookkeeping of every emitted token.
func (l *BashSharpLexer) NextToken() antlr.Token {
	st := &l.st
	if !st.started {
		st.started = true
		l.SetMode(BashSharpLexerSH)
		st.frames = []*lexFrame{{kind: shRoot, cmdPos: true}}
		st.funcs = map[string]bool{}
	}
	if len(st.pending) > 0 {
		t := st.pending[0]
		st.pending = st.pending[1:]
		return t
	}
	f := l.top()
	switch mode := l.st.mode; {
	case !f.goRegion() && mode == BashSharpLexerSH:
		if st.bashsharp && f.cmdPos && !f.cond && f.forStage == 0 && !f.redirect && f.heredocOp == 0 {
			if kind := l.startSite(f); kind != 0 {
				l.PushMode(antlr.LexerDefaultMode)
				st.frames = append(st.frames, &lexFrame{kind: kind})
				f.cmdPos = false
			}
		}
	case f.goRegion() && (mode == antlr.LexerDefaultMode || mode == BashSharpLexerNLSEMI):
		if l.regionEndsAhead(f) {
			return l.endRegion(l.GetInputStream().Index())
		}
	}
	t := l.BaseLexer.NextToken()
	if t.GetTokenType() == antlr.TokenEOF {
		// A here-document whose delimiter never appears is unclosed (README P9).
		if len(st.heredocs) > 0 {
			l.GetErrorListenerDispatch().SyntaxError(l, nil, t.GetLine(), t.GetColumn(), "unclosed here-document "+st.heredocs[0].delim, nil)
			st.heredocs = nil
		}
		// A region still open at end of input ends there (README P6).
		if l.top().goRegion() {
			for l.top().goRegion() {
				st.pending = append(st.pending, l.endRegion(t.GetStart()))
			}
			st.pending = append(st.pending, t)
			r := st.pending[0]
			st.pending = st.pending[1:]
			return r
		}
		return t
	}
	if t.GetChannel() == antlr.TokenDefaultChannel {
		l.after(t)
	}
	return t
}

// endRegion pops the innermost Go region and returns its zero-width
// REGION_END at index.
func (l *BashSharpLexer) endRegion(index int) antlr.Token {
	t := antlr.NewCommonToken(l.GetTokenSourceCharStreamPair(), BashSharpLexerREGION_END, antlr.TokenDefaultChannel, index, index-1)
	l.popRegion()
	return t
}

func (l *BashSharpLexer) popRegion() {
	st := &l.st
	st.frames = st.frames[:len(st.frames)-1]
	l.PopMode()
	sh := l.top()
	sh.cmdPos = true
	sh.declArgs = false
}

// regionEndsAhead: at the outer level of a Go region, a "}" that closes the
// enclosing Go block, a ")" that closes an enclosing subshell, or a ";;"
// case terminator ends the region without being part of it (README P6).
func (l *BashSharpLexer) regionEndsAhead(f *lexFrame) bool {
	if f.depth() != 0 || f.header || f.kind == goCase {
		return false
	}
	in := l.GetInputStream()
	i := 1
	for c := in.LA(i); c == ' ' || c == '\t'; c = in.LA(i) {
		i++
	}
	sh := l.shTop()
	switch in.LA(i) {
	case '}':
		// A "}" glued to the word before it (`"x"}`) is part of that word,
		// not a closer (sh/syntax reads Go statements from shell words).
		glued := i == 1 && !strings.ContainsRune("\n;&|()", rune(in.LA(-1)))
		return goBlockContext(sh) && l.standalone(i) && !glued
	case ')':
		return len(sh.parens) > 0 && sh.parens[len(sh.parens)-1] == parenSubshell
	case ';':
		return in.LA(i+1) == ';' || in.LA(i+1) == '&'
	}
	return false
}

// standalone reports whether the "}" at LA(i) is a whole shell word.
func (l *BashSharpLexer) standalone(i int) bool {
	switch l.GetInputStream().LA(i + 1) {
	case ' ', '\t', '\n', ';', '&', '|', '(', ')', '<', '>', antlr.TokenEOF:
		return true
	}
	return false
}

// after does the bookkeeping for a default-channel token.
func (l *BashSharpLexer) after(t antlr.Token) {
	f := l.top()
	if f.goRegion() {
		l.afterGo(f, t)
		return
	}
	l.afterSh(f, t)
}

func (l *BashSharpLexer) afterSh(f *lexFrame, t antlr.Token) {
	st := &l.st
	ty := t.GetTokenType()
	if IsWordType(ty) {
		l.afterShWord(f, t.GetText())
		return
	}
	f.redirect = false
	switch ty {
	case BashSharpLexerSH_NEWLINE:
		f.cmdPos = len(f.cases) == 0 || f.cases[len(f.cases)-1] == caseBody
		f.declArgs = false
		if f.forStage >= 2 {
			f.forStage = 0
		}
		if f.forStage == 1 {
			f.forStage = 0
		}
		f.afterArith = false
		if len(st.heredocs) > 0 {
			l.PushMode(BashSharpLexerHEREDOC)
		}
	case BashSharpLexerSH_SEMI, BashSharpLexerSH_AMP, BashSharpLexerAND_IF, BashSharpLexerOR_IF,
		BashSharpLexerSH_PIPE, BashSharpLexerPIPE_AMP:
		if f.cond {
			return
		}
		f.cmdPos = true
		f.declArgs = false
		f.forStage = 0
		f.afterArith = false
	case BashSharpLexerDSEMI, BashSharpLexerSEMI_AMP, BashSharpLexerDSEMI_AMP:
		f.cmdPos = false
		f.declArgs = false
		if n := len(f.cases); n > 0 {
			f.cases[n-1] = casePattern
		}
	case BashSharpLexerSH_LPAREN:
		if f.cond {
			return
		}
		switch {
		case len(f.cases) > 0 && f.cases[len(f.cases)-1] == casePattern:
			f.parens = append(f.parens, parenPattern)
		case f.cmdPos:
			f.parens = append(f.parens, parenSubshell)
		default:
			f.parens = append(f.parens, parenOther)
		}
		f.cmdPos = true
	case BashSharpLexerSH_RPAREN:
		if f.cond {
			return
		}
		if n := len(f.parens); n > 0 {
			p := f.parens[n-1]
			f.parens = f.parens[:n-1]
			switch p {
			case parenSubshell:
				f.cmdPos = false
			case parenPattern:
				f.cases[len(f.cases)-1] = caseBody
				f.cmdPos = true
			default:
				f.cmdPos = true
			}
			return
		}
		if n := len(f.cases); n > 0 && f.cases[n-1] == casePattern {
			f.cases[n-1] = caseBody
			f.cmdPos = true
		}
	case BashSharpLexerDLESS:
		f.heredocOp = 1
	case BashSharpLexerDLESSDASH:
		f.heredocOp = 2
	case BashSharpLexerSH_LESS, BashSharpLexerSH_GREAT, BashSharpLexerDGREAT, BashSharpLexerLESSAND,
		BashSharpLexerGREATAND, BashSharpLexerLESSGREAT, BashSharpLexerCLOBBER, BashSharpLexerAMP_GREAT,
		BashSharpLexerAMP_DGREAT, BashSharpLexerTLESS:
		if !f.cond {
			f.redirect = true
		}
	case BashSharpLexerARITH_CMD:
		f.cmdPos = false
		if f.forStage == 1 {
			f.forStage = 0
			f.afterArith = true
		}
	case BashSharpLexerFENCE_CLOSE:
		f.cmdPos = true
	case BashSharpLexerRBRACE:
		if f.groups > 0 {
			f.groups--
			f.cmdPos = false
			return
		}
		// The brace closes the Go block: back to its Go region, where a
		// newline is an inserted semicolon (README P4, P6).
		st.frames = st.frames[:len(st.frames)-1]
		l.PopMode()
		l.SetMode(BashSharpLexerNLSEMI)
		g := l.top()
		g.prev = BashSharpLexerR_CURLY
	}
}

var shReservedKeep = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "while": true, "until": true,
	"do": true, "!": true, "time": true, "{": true,
}

func (l *BashSharpLexer) afterShWord(f *lexFrame, w string) {
	st := &l.st
	w = strings.ReplaceAll(w, "\\\n", "")
	switch {
	case f.heredocOp != 0:
		st.heredocs = append(st.heredocs, heredoc{delim: heredocDelim(w), dash: f.heredocOp == 2})
		f.heredocOp = 0
		return
	case f.redirect:
		f.redirect = false
		return
	case f.cond:
		switch w {
		case "]]":
			f.cond = false
			f.cmdPos = false
		case "=~":
			l.PushMode(BashSharpLexerCOND_REGEX)
		}
		return
	case f.funcName:
		f.funcName = false
		f.cmdPos = true
		return
	case f.forStage == 1:
		f.forStage = 2
		return
	case f.forStage == 2:
		switch w {
		case "in":
			f.forStage = 3
		case "do":
			f.forStage = 0
			f.cmdPos = true
		default:
			f.forStage = 0
		}
		return
	case f.forStage == 3:
		return
	}
	if n := len(f.cases); n > 0 {
		switch f.cases[n-1] {
		case caseSubject:
			f.cases[n-1] = caseIn
			return
		case caseIn:
			if w == "in" {
				f.cases[n-1] = casePattern
			}
			return
		case casePattern:
			if w == "esac" {
				f.cases = f.cases[:n-1]
				f.cmdPos = false
			}
			return
		}
	}
	if !f.cmdPos {
		return
	}
	if f.afterArith && w == "do" {
		f.afterArith = false
		return
	}
	switch {
	case shReservedKeep[w]:
		if w == "{" && goBlockContext(f) {
			f.groups++
		}
	case w == "case":
		f.cases = append(f.cases, caseSubject)
		f.cmdPos = false
	case w == "for" || w == "select":
		f.forStage = 1
		f.cmdPos = false
	case w == "function":
		f.funcName = true
		f.cmdPos = false
	case w == "[[":
		f.cond = true
		f.cmdPos = false
	case w == "esac":
		if n := len(f.cases); n > 0 {
			f.cases = f.cases[:n-1]
		}
		f.cmdPos = false
	case w == "declare" || w == "local" || w == "export" || w == "readonly" || w == "typeset":
		f.declArgs = true
		f.cmdPos = false
	case w == "agentic" && st.bashsharp:
	case isAssignmentWord(w):
	default:
		f.cmdPos = false
	}
}

func (l *BashSharpLexer) afterGo(f *lexFrame, t antlr.Token) {
	ty := t.GetTokenType()
	d := f.depth()
	f.ntok++
	if f.kind == goFunc && f.ntok == 2 && f.prev == BashSharpLexerFUNC && ty == BashSharpLexerIDENTIFIER {
		l.st.funcs[t.GetText()] = true
	}
	switch ty {
	case BashSharpLexerREGION_END:
		l.popRegion()
		return
	case BashSharpLexerEOS:
		l.SetMode(antlr.LexerDefaultMode)
		if d == 0 {
			f.litType = false
		}
		l.endSigs(f, d)
	case BashSharpLexerL_PAREN:
		if n := len(f.sigs); n > 0 && f.sigs[n-1].stage == sigParams && f.sigs[n-1].depth == d {
			f.sigs[n-1].stage = sigInParams
		}
		f.paren++
	case BashSharpLexerR_PAREN:
		f.paren--
		l.closeTo(f, f.depth())
		if n := len(f.sigs); n > 0 && f.sigs[n-1].stage == sigInParams && f.sigs[n-1].depth == f.depth() {
			f.sigs[n-1].stage = sigAfterParams
		}
	case BashSharpLexerL_BRACKET:
		if d == 0 && f.header && !operandEnd(f.prev) {
			f.litType = true
		}
		f.bracket++
	case BashSharpLexerR_BRACKET:
		f.bracket--
		l.closeTo(f, f.depth())
	case BashSharpLexerL_CURLY:
		if d == 0 && f.header {
			f.litType = false
		}
		f.brace++
	case BashSharpLexerR_CURLY:
		f.brace--
		l.closeTo(f, f.depth())
	case BashSharpLexerBLOCK_OPEN:
		kind := shGoBlock
		if d == 0 && f.header {
			f.header = false
			if f.headerKw == BashSharpLexerSWITCH || f.headerKw == BashSharpLexerSELECT {
				kind = shSwitchBody
			}
		}
		for n := len(f.sigs); n > 0 && f.sigs[n-1].depth >= d; n = len(f.sigs) {
			f.sigs = f.sigs[:n-1]
		}
		f.prev = ty
		l.st.frames = append(l.st.frames, &lexFrame{kind: kind, cmdPos: true})
		l.PushMode(BashSharpLexerSH)
		return
	case BashSharpLexerFUNC:
		f.sigs = append(f.sigs, funcSig{depth: d, stage: sigParams})
	case BashSharpLexerIF, BashSharpLexerFOR, BashSharpLexerSWITCH, BashSharpLexerSELECT, BashSharpLexerELSE:
		if d == 0 {
			f.header = true
			f.headerKw = ty
			f.litType = false
		}
	case BashSharpLexerMAP, BashSharpLexerSTRUCT:
		if d == 0 && f.header {
			f.litType = true
		}
	case BashSharpLexerCOLON:
		if f.kind == goCase && d == 0 {
			l.popRegion()
			return
		}
		if d == 0 {
			f.litType = false
		}
	case BashSharpLexerASSIGN, BashSharpLexerCOMMA, BashSharpLexerDECLARE_ASSIGN, BashSharpLexerSEMI:
		if d == 0 {
			f.litType = false
		}
		l.endSigs(f, d)
	default:
		if d == 0 && !operandEnd(ty) && ty != BashSharpLexerIDENTIFIER && ty != BashSharpLexerDOT && ty != BashSharpLexerSTAR {
			f.litType = false
		}
	}
	f.prev = ty
	f.prevText = t.GetText()
}

// closeTo drops function signatures nested deeper than depth d.
func (l *BashSharpLexer) closeTo(f *lexFrame, d int) {
	for n := len(f.sigs); n > 0 && f.sigs[n-1].depth > d; n = len(f.sigs) {
		f.sigs = f.sigs[:n-1]
	}
}

// endSigs: a function type at depth d without a body ends at "=", ",", ";".
func (l *BashSharpLexer) endSigs(f *lexFrame, d int) {
	for n := len(f.sigs); n > 0 && f.sigs[n-1].depth >= d && f.sigs[n-1].stage == sigAfterParams; n = len(f.sigs) {
		f.sigs = f.sigs[:n-1]
	}
}

// operandEnd: tokens after which "[" indexes rather than opens a type.
func operandEnd(ty int) bool {
	switch ty {
	case BashSharpLexerIDENTIFIER, BashSharpLexerR_PAREN, BashSharpLexerR_BRACKET, BashSharpLexerR_CURLY,
		BashSharpLexerDECIMAL_LIT, BashSharpLexerBINARY_LIT, BashSharpLexerOCTAL_LIT, BashSharpLexerHEX_LIT,
		BashSharpLexerFLOAT_LIT, BashSharpLexerIMAGINARY_LIT, BashSharpLexerRUNE_LIT, BashSharpLexerRAW_STRING_LIT,
		BashSharpLexerINTERPRETED_STRING_LIT, BashSharpLexerSHELL_OPERAND, BashSharpLexerNIL_LIT:
		return true
	}
	return false
}

// ---- actions ----

// goLCurly classifies a "{" in a Go region (README P4): a function body, or
// the block of an if/for/switch/select/else header at the region's outer
// level, is a BLOCK_OPEN whose body is shell text; any other brace (a
// composite literal, a struct, interface or enum body) stays L_CURLY. In a
// header, a brace after an array, slice, map or struct literal type is a
// literal, as the Go spec's composite-literal parsing ambiguity requires.
func (l *BashSharpLexer) goLCurly() {
	f := l.top()
	if !f.goRegion() {
		return
	}
	d := f.depth()
	switch {
	case f.prev == BashSharpLexerSTRUCT || f.prev == BashSharpLexerINTERFACE:
	case f.prev == BashSharpLexerIDENTIFIER && f.prevText == "enum" && f.kind == goDecl:
	case len(f.sigs) > 0 && f.sigs[len(f.sigs)-1].depth == d && f.sigs[len(f.sigs)-1].stage == sigAfterParams:
		l.SetType(BashSharpLexerBLOCK_OPEN)
	case d == 0 && f.header && !f.litType:
		l.SetType(BashSharpLexerBLOCK_OPEN)
	}
}

// goEOS: an inserted semicolon at the outer level of a region, outside a
// control header, ends the region (README P6). After a complete function
// signature, newlines and comment lines before a "{" are the delta's
// header_end and are hidden (README P4).
func (l *BashSharpLexer) goEOS() {
	f := l.top()
	if n := len(f.sigs); n > 0 && f.sigs[n-1].stage == sigAfterParams && f.sigs[n-1].depth == f.depth() &&
		strings.Contains(l.GetText(), "\n") && strings.HasPrefix(l.nextCodeLine(1), "{") {
		l.SetType(BashSharpLexerEOS)
		l.SetChannel(antlr.TokenHiddenChannel)
		return
	}
	if f.goRegion() && f.depth() == 0 && !f.header && f.kind != goCase {
		l.SetType(BashSharpLexerREGION_END)
		return
	}
	l.SetType(BashSharpLexerEOS)
}

// reservedSpelling maps the spelling of each Bash reserved word to its token
// type (README P1).
var reservedSpelling = map[string]int{
	"if": BashSharpLexerW_IF, "then": BashSharpLexerW_THEN, "elif": BashSharpLexerW_ELIF,
	"else": BashSharpLexerW_ELSE, "fi": BashSharpLexerW_FI, "do": BashSharpLexerW_DO,
	"done": BashSharpLexerW_DONE, "case": BashSharpLexerW_CASE, "esac": BashSharpLexerW_ESAC,
	"while": BashSharpLexerW_WHILE, "until": BashSharpLexerW_UNTIL, "for": BashSharpLexerW_FOR,
	"select": BashSharpLexerW_SELECT, "in": BashSharpLexerW_IN, "function": BashSharpLexerW_FUNCTION,
	"time": BashSharpLexerW_TIME, "coproc": BashSharpLexerW_COPROC, "!": BashSharpLexerW_BANG,
	"{": BashSharpLexerW_LBRACE, "}": BashSharpLexerW_RBRACE, "[[": BashSharpLexerW_DLBRACK,
	"]]": BashSharpLexerW_DRBRACK,
}

// IsWordType reports whether a token type is one of the shell word types.
func IsWordType(ty int) bool {
	switch ty {
	case BashSharpLexerWORD, BashSharpLexerAT_WORD, BashSharpLexerASSIGNMENT_WORD, BashSharpLexerW_AGENTIC:
		return true
	}
	return reservedType[ty]
}

var reservedType = func() map[int]bool {
	m := map[int]bool{}
	for _, ty := range reservedSpelling {
		m[ty] = true
	}
	return m
}()

// shWord gives a word its spelling type (README P1), and retypes a standalone
// "}" that closes a Go block or a brace group inside one as RBRACE (P5).
func (l *BashSharpLexer) shWord() {
	f := l.top()
	if strings.HasPrefix(l.GetText(), "#") && l.atWordStart() {
		l.commentWord()
		return
	}
	if !f.goRegion() && !f.cond && (f.cmdPos || f.declArgs) && f.forStage == 0 && !f.redirect && f.heredocOp == 0 {
		l.subscriptWord()
	}
	text := strings.ReplaceAll(l.GetText(), "\\\n", "") // a line continuation joins the word
	switch ty, ok := reservedSpelling[text]; {
	case text == "}" && !f.goRegion() && !f.cond && goBlockContext(f):
		l.SetType(BashSharpLexerRBRACE)
	case ok:
		l.SetType(ty)
	case l.st.bashsharp && text == "agentic":
		l.SetType(BashSharpLexerW_AGENTIC)
	case l.st.bashsharp && strings.HasPrefix(text, "@"):
		l.SetType(BashSharpLexerAT_WORD)
	case isAssignmentWord(text):
		l.SetType(BashSharpLexerASSIGNMENT_WORD)
	}
}

// commentWord: a "#" at the start of a word begins a comment through the
// end of the line (README P2), even where WORD's longest match ran on, for
// example through a quote that the comment holds. The input is rewound to
// the end of the line.
func (l *BashSharpLexer) commentWord() {
	text, _, _ := strings.Cut(l.GetText(), "\n")
	n := len([]rune(text))
	l.GetInputStream().Seek(l.TokenStartCharIndex + n)
	if sim, ok := l.Interpreter.(*antlr.LexerATNSimulator); ok {
		sim.Line, sim.CharPositionInLine = l.TokenStartLine, l.TokenStartColumn+n
	}
	l.SetText(text)
	l.SetType(BashSharpLexerSH_COMMENT)
	l.SetChannel(antlr.TokenHiddenChannel)
}

// subscriptWord: where an assignment may stand, a word that opens with
// NAME[ reads the subscript as a matched pair through its "]", across blanks,
// operators and newlines, and then continues the word (README P1, as
// sh/syntax and Bash 5.3 read `a[i]=v`). End of input inside the pair is an
// error.
func (l *BashSharpLexer) subscriptWord() {
	text := l.GetText()
	k := 0
	for k < len(text) && (text[k] == '_' || isASCIILetter(text[k]) || k > 0 && text[k] >= '0' && text[k] <= '9') {
		k++
	}
	if k == 0 || k == len(text) || text[k] != '[' {
		return
	}
	var sc subscan
	for _, c := range text[k:] {
		if sc.step(c) {
			return // closed inside the word the rule matched
		}
	}
	in := l.GetInputStream()
	var b strings.Builder
	b.WriteString(text)
	for closed := false; !closed; {
		c := in.LA(1)
		if c == antlr.TokenEOF {
			l.GetErrorListenerDispatch().SyntaxError(l, nil, l.TokenStartLine, l.TokenStartColumn, "unexpected EOF while looking for matching `]'", nil)
			return
		}
		b.WriteRune(rune(c))
		l.Interpreter.Consume(in)
		closed = sc.step(rune(c))
	}
	// The rest of the word after the subscript.
	for c := in.LA(1); c != antlr.TokenEOF && !strings.ContainsRune(" \t\n;&|()<>", rune(c)); c = in.LA(1) {
		b.WriteRune(rune(c))
		l.Interpreter.Consume(in)
	}
	l.SetText(b.String())
}

// subscan is the matched-pair scan of a subscript: bracket depth, an open
// quote (single, double or back), the depth of $( ) nesting, and a pending
// backslash escape.
type subscan struct {
	depth, paren int
	q, prev      rune
	esc          bool
}

// step consumes c and reports whether the subscript's "]" closed.
func (s *subscan) step(c rune) bool {
	prev := s.prev
	s.prev = c
	switch {
	case s.esc:
		s.esc = false
	case s.q == '\'':
		if c == '\'' {
			s.q = 0
		}
	case c == '\\':
		s.esc = true
	case s.q != 0:
		if c == s.q {
			s.q = 0
		}
	case c == '\'' || c == '"' || c == '`':
		s.q = c
	case c == '(' && (s.paren > 0 || prev == '$'):
		s.paren++
	case c == ')' && s.paren > 0:
		s.paren--
	case s.paren > 0:
	case c == '[':
		s.depth++
	case c == ']':
		s.depth--
		return s.depth == 0
	}
	return false
}

// fenceOpen records the opener's tilde run (README P7).
func (l *BashSharpLexer) fenceOpen() {
	text := l.GetText()
	n := 0
	for n < len(text) && text[n] == '~' {
		n++
	}
	l.st.fence = text[:n]
	fields := strings.Fields(text)
	l.st.fenceLang = ""
	if len(fields) < 2 || fields[1] != "as" {
		l.st.fenceLang = strings.ToLower(fields[0][n:])
	}
	l.st.fenceBody.Reset()
}

// fenceLine: the first line equal to the opener's tilde run closes the fence;
// every other line is opaque body text. A line ends at "\n" or end of input;
// sh/syntax reads "\r\n" as "\n" (Parser.rune), so a "\r" just before the
// newline is not part of the line, while any other "\r" is.
func (l *BashSharpLexer) fenceLine() {
	line, nl := strings.CutSuffix(l.GetText(), "\n")
	if nl {
		line = strings.TrimSuffix(line, "\r")
	}
	if line == l.st.fence {
		l.SetType(BashSharpLexerFENCE_CLOSE)
		l.PopMode()
		l.fenceFuncs()
		return
	}
	l.st.fenceBody.WriteString(line)
	if nl {
		l.st.fenceBody.WriteByte('\n')
	}
	l.SetChannel(antlr.TokenHiddenChannel)
}

// fenceFuncs: a closed, unaliased fence declares the functions its language's
// look-ahead reads, so a later `name()` is a call, not a shell function head
// (sh/syntax bashppRegisterSourceBlockFuncs and bashppFenceLookahead).
func (l *BashSharpLexer) fenceFuncs() {
	read := fenceLookahead[l.st.fenceLang]
	if read == nil {
		return
	}
	for _, line := range strings.Split(l.st.fenceBody.String(), "\n") {
		decl := read(line)
		if decl == "" {
			continue
		}
		name, _, ok := strings.Cut(decl, "(")
		name = strings.TrimSpace(name)
		if ok && !strings.HasPrefix(name, "_") && validIdent(name) {
			l.st.funcs[name] = true
		}
	}
}

// fenceLookahead maps a fence language spelling to its declaration-line
// reader, which returns the text from the declared name on, or "".
var fenceLookahead = map[string]func(string) string{}

func init() {
	add := func(read func(string) string, spellings ...string) {
		for _, s := range spellings {
			fenceLookahead[s] = read
		}
	}
	add(func(line string) string {
		if r, ok := strings.CutPrefix(line, "def "); ok {
			return r
		}
		return ""
	}, "python", "py")
	add(func(line string) string {
		if r, ok := strings.CutPrefix(strings.TrimPrefix(line, "export "), "function "); ok {
			return r
		}
		return ""
	}, "typescript", "ts", "javascript", "js")
	add(func(line string) string {
		if r, ok := strings.CutPrefix(strings.TrimSpace(line), "pub fn "); ok {
			return r
		}
		return ""
	}, "rust", "rs")
	add(func(line string) string {
		decl := strings.TrimSpace(line)
		if strings.HasPrefix(decl, "static ") || strings.HasPrefix(decl, "#") {
			return ""
		}
		before, after, ok := strings.Cut(decl, "(")
		if !ok || before == "" {
			return ""
		}
		head := strings.TrimSpace(before)
		if strings.HasSuffix(head, "if") || strings.HasSuffix(head, "for") || strings.HasSuffix(head, "while") {
			return ""
		}
		parts := strings.Fields(before)
		if len(parts) <= 1 {
			return ""
		}
		return parts[len(parts)-1] + "(" + after
	}, "c", "cpp", "cxx")
	add(func(line string) string {
		if r, ok := strings.CutPrefix(strings.TrimSpace(line), "func "); ok {
			return r
		}
		return ""
	}, "go")
	add(func(line string) string {
		decl, rest := strings.TrimSpace(line), ""
		for _, kw := range []string{"function ", "filter "} {
			if r, ok := strings.CutPrefix(decl, kw); ok {
				rest = strings.TrimSpace(r)
				break
			}
		}
		if rest == "" {
			return ""
		}
		name := rest
		if i := strings.IndexAny(rest, " \t({"); i >= 0 {
			name = rest[:i]
		}
		if i := strings.LastIndex(name, ":"); i >= 0 {
			name = name[i+1:]
		}
		if name == "" {
			return ""
		}
		return name + "("
	}, "powershell", "pwsh", "ps1")
	add(func(line string) string {
		decl := strings.TrimSpace(line)
		if !strings.HasPrefix(decl, "public static ") {
			return ""
		}
		before, after, ok := strings.Cut(decl, "(")
		if !ok {
			return ""
		}
		fields := strings.Fields(before)
		if len(fields) < 4 || strings.ContainsAny(before, "=;") {
			return ""
		}
		return fields[len(fields)-1] + "(" + after
	}, "csharp", "cs")
	add(func(line string) string {
		before, _, ok := strings.Cut(strings.TrimSpace(line), "()")
		if !ok {
			return ""
		}
		return strings.TrimSpace(before) + "("
	}, "bash", "sh")
}

// heredocLine consumes here-document body lines in queue order (README P9).
func (l *BashSharpLexer) heredocLine() {
	st := &l.st
	if len(st.heredocs) == 0 {
		l.PopMode()
		return
	}
	line := strings.TrimSuffix(l.GetText(), "\n")
	h := st.heredocs[0]
	if h.dash {
		line = strings.TrimLeft(line, "\t")
	}
	if line == h.delim {
		st.heredocs = st.heredocs[1:]
		if len(st.heredocs) == 0 {
			l.PopMode()
		}
	}
}

// heredocDelim removes quoting from a here-document delimiter word.
func heredocDelim(w string) string {
	var b strings.Builder
	for i := 0; i < len(w); i++ {
		switch c := w[i]; c {
		case '\'', '"':
			continue
		case '\\':
			if i+1 < len(w) {
				i++
				b.WriteByte(w[i])
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// ---- predicates ----

// before returns the character before the current token (0 at the start of
// the input). Predicates run at the end of their rule, so they read the
// token start rather than the current index.
func (l *BashSharpLexer) before() rune {
	i := l.TokenStartCharIndex
	if i == 0 {
		return 0
	}
	r := []rune(l.GetInputStream().GetText(i-1, i-1))
	if len(r) == 0 {
		return 0
	}
	return r[0]
}

// atWordStart: the character before the token ends a word (README P2).
func (l *BashSharpLexer) atWordStart() bool {
	switch l.before() {
	case 0, ' ', '\t', '\n', ';', '&', '|', '(', ')', '<', '>':
		return true
	}
	return false
}

// dollarLiteral: a "$" is literal only before a character that cannot begin
// an expansion (in double quotes, a quote ends the string instead).
func (l *BashSharpLexer) dollarLiteral(dq bool) bool {
	c := l.GetInputStream().LA(1)
	switch {
	case c == antlr.TokenEOF:
		return true
	case c == '{' || c == '(' || c == '[' || c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9':
		return false
	case strings.ContainsRune("@*#?$!-", rune(c)):
		return false
	case c == '\'' || c == '"':
		return dq
	}
	return true
}

func (l *BashSharpLexer) slashCommentHere() bool {
	return l.st.bashsharp && !l.top().cond && l.atWordStart()
}

func (l *BashSharpLexer) redirectFollows() bool {
	c := l.GetInputStream().LA(1)
	return c == '<' || c == '>'
}

func (l *BashSharpLexer) arithCmdHere() bool {
	f := l.top()
	return !f.cond && (f.cmdPos || f.forStage == 1)
}

func (l *BashSharpLexer) arrayAssignHere() bool {
	f := l.top()
	return !f.cond && (f.cmdPos || f.declArgs)
}

// atLineStart: the current index is at column one (called between tokens).
func (l *BashSharpLexer) atLineStart() bool {
	in := l.GetInputStream()
	return in.Index() == 0 || in.LA(-1) == '\n'
}

// lookLine returns the runes from LA(1) up to (not including) the next
// newline or end of input.
func (l *BashSharpLexer) lookLine() []rune {
	in := l.GetInputStream()
	var out []rune
	for i := 1; ; i++ {
		c := in.LA(i)
		if c == antlr.TokenEOF || c == '\n' {
			return out
		}
		out = append(out, rune(c))
	}
}

// fenceHere: a column-one fence header at a command position (README P7),
// transcribed from sh/syntax recognizeSourceFence: ~~~ (or more) then a
// language name, an optional `as ALIAS`, an optional `!RUNNER`, and nothing
// else on the line.
func (l *BashSharpLexer) fenceHere() bool {
	f := l.top()
	if !l.st.bashsharp || !f.cmdPos || f.cond {
		return false
	}
	if b := l.before(); b != 0 && b != '\n' {
		return false
	}
	return validFenceHeader(strings.TrimSuffix(l.GetText(), "\n"))
}

func validFenceHeader(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	opener := fields[0]
	n := 0
	for n < len(opener) && opener[n] == '~' {
		n++
	}
	if n < 3 || n == len(opener) || !validFenceLanguage(opener[n:]) {
		return false
	}
	return validFenceTail(opener[n:], fields[1:])
}

func validFenceLanguage(lang string) bool {
	if validIdent(lang) || lang == "go" {
		return true
	}
	if lang == "" || !(lang[0] == '_' || isASCIILetter(lang[0])) {
		return false
	}
	for i := 1; i < len(lang); i++ {
		c := lang[i]
		if !(c == '-' || c == '+' || c == '_' || c >= '0' && c <= '9' || isASCIILetter(c)) {
			return false
		}
	}
	return true
}

func validFenceTail(lang string, fields []string) bool {
	alias := func(a string) bool { return validIdent(a) || strings.EqualFold(lang, "go") && a == "go" }
	if len(fields) > 0 && fields[0] == "as" {
		if len(fields) < 2 {
			return false
		}
		if r, ok := strings.CutPrefix(fields[1], "!"); ok {
			return validRunner(r) && alias(r) && len(fields) == 2
		}
		if !alias(fields[1]) {
			return false
		}
		fields = fields[2:]
	}
	if len(fields) == 0 {
		return true
	}
	r, ok := strings.CutPrefix(fields[0], "!")
	return len(fields) == 1 && ok && validRunner(r)
}

func validRunner(r string) bool {
	if r == "" || r[0] == '-' || r[len(r)-1] == '-' {
		return false
	}
	for i := 0; i < len(r); i++ {
		if c := r[i]; !(c == '-' || c == '_' || c >= '0' && c <= '9' || isASCIILetter(c)) {
			return false
		}
	}
	return true
}

func isASCIILetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// ---- the start-site decision (README P5) ----

// startSite decides, at a shell command position, whether a Go region opens.
// It transcribes sh/syntax RecognizeStartSite and the parser.go dispatch that
// consults it, and it returns the region kind or 0 (stay shell).
func (l *BashSharpLexer) startSite(f *lexFrame) int {
	line := l.lookLine()
	col1 := l.atLineStart()
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	if i > 0 {
		col1 = false
	}
	s := string(line[i:])
	if s == "" {
		return 0
	}
	goBlock := goBlockContext(f)
	if f.kind == shSwitchBody && goBlock {
		if _, ok := cutKeyword(s, "case"); ok || s == "case" {
			return goCase
		}
		if rest, ok := strings.CutPrefix(s, "default"); ok && strings.HasPrefix(strings.TrimLeft(rest, " \t"), ":") {
			return goCase
		}
	}
	if col1 && strings.HasPrefix(s, "@") {
		switch decoratorHere(s) {
		case decoYes:
			return goDecorator
		case decoNextLine:
			if !compoundStart(l.nextCodeLine(len(line) + 1)) {
				return goDecorator
			}
		}
		return 0
	}
	for _, kw := range []string{"var", "const", "type"} {
		if rest, ok := cutKeyword(s, kw); ok {
			if name := leadingIdent(rest); name != "" && !goKeywords[name] && (kw != "type" || typeShape(rest[len(name):]) && l.bodyCloses()) {
				return goDecl
			}
		}
	}
	// A grouped const declaration (sh/syntax bashppConstGroup); var and type
	// groups stay shell, as there.
	if rest, ok := cutKeyword(s, "const"); ok && strings.HasPrefix(rest, "(") {
		return goDecl
	}
	if _, ok := cutKeyword(s, "import"); ok || s == "import" {
		return goDecl
	}
	if _, ok := cutKeyword(s, "package"); ok {
		return goDecl
	}
	if rest, ok := cutKeyword(s, "func"); ok {
		if name := leadingIdent(rest); name != "" && !goKeywords[name] {
			sig := rest[len(name):]
			// Type parameters: `func Index[T comparable](` (sh/syntax
			// bashppFuncTypeParams reads them from shell words, so a list
			// holding a shell operator, as in `[T int | float64]`, the S6
			// M11 gap, is not this site).
			if end := matchBracket(sig); strings.HasPrefix(sig, "[") && end > 0 && !strings.ContainsAny(sig[:end], "|&;<>()") {
				sig = sig[end+1:]
			}
			if strings.HasPrefix(strings.TrimLeft(sig, " \t"), "(") {
				return goFunc
			}
		}
	}
	if lit, empty := funcLitPrefix(s); lit && !empty {
		return goFunc
	}
	if rest, ok := cutKeyword(s, "defer"); ok && goBlock && callOrLit(rest) {
		return goStmt
	}
	if rest, ok := cutKeyword(s, "go"); ok && callOrLit(rest) {
		return goStmt
	}
	if goBlock {
		if _, ok := cutKeyword(s, "return"); ok || s == "return" {
			return goStmt
		}
		if rest, ok := cutKeyword(s, "if"); ok && goIfHeader(rest) {
			return goControl
		}
		if rest, ok := cutKeyword(s, "for"); ok && goForHeader(rest) {
			return goControl
		}
		if rest, ok := cutKeyword(s, "switch"); ok && strings.Contains(rest, "{") || strings.HasPrefix(s, "switch {") {
			return goControl
		}
		if rest, ok := cutKeyword(s, "select"); ok && strings.HasPrefix(rest, "{") || strings.HasPrefix(s, "select{") {
			return goControl
		}
	}
	// The := is a word of its own: `ech:=o` and `x :=1` stay shell words
	// (sh/syntax bashppShortDecl).
	if rest, ok := cutShortDeclLhs(s); ok && (strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "\t")) && strings.TrimLeft(rest, " \t") != "" {
		return goSimple
	}
	if l.goCallHere(s) {
		return goSimple
	}
	if sendOrReceive(s) {
		return goSimple
	}
	if strings.HasPrefix(s, "*") {
		if rest, ok := cutAssign(s[1:], true); ok && rest != "" {
			return goSimple
		}
	}
	if goBlock {
		if _, ok := cutAssignLhs(s); ok {
			return goSimple
		}
		if incDec(s) {
			return goSimple
		}
	} else if rest, ok := cutPlainAssign(s); ok && hasCallShape(strings.TrimLeft(rest, " \t")) {
		return goSimple
	}
	return 0
}

// bodyCloses: a type body opened on this line (a struct, interface or enum
// "{") has its matching "}" before end of input. An unclosed body is not a
// complete type shape, and `type T struct {` stays the shell builtin, as in
// sh/syntax. Quotes and // comments are skipped.
func (l *BashSharpLexer) bodyCloses() bool {
	in := l.GetInputStream()
	depth, opened := 0, false
	for i := 1; ; i++ {
		c := in.LA(i)
		switch {
		case c == antlr.TokenEOF:
			return !opened
		case c == '\n' && !opened:
			return true
		case c == '"' || c == '\'' || c == '`':
			for i++; in.LA(i) != c && in.LA(i) != antlr.TokenEOF; i++ {
				if in.LA(i) == '\\' && c != '`' {
					i++
				}
			}
		case c == '/' && in.LA(i+1) == '/':
			for in.LA(i+1) != '\n' && in.LA(i+1) != antlr.TokenEOF {
				i++
			}
		case c == '{':
			depth++
			opened = true
		case c == '}':
			depth--
			if depth == 0 {
				return true
			}
		}
	}
}

// typeShape: after `type NAME`, a complete Go type shape commits (the
// start-site table row "`type NAME TYPE` with complete Go type shape"): an
// alias, a struct, interface or enum body, a type with parentheses or
// braces, or a single type word ending the command. `type foo`, and a type
// word followed by more words, stay the shell builtin (sh/syntax
// bashppTypeDecl).
func typeShape(rest string) bool {
	if strings.HasPrefix(rest, "[") { // type parameters
		end := matchBracket(rest)
		if end < 0 {
			return true
		}
		rest = rest[end+1:]
	}
	cmd, _, _ := strings.Cut(rest, ";")
	words := lineWords(cmd)
	switch {
	case len(words) == 0:
		return false
	case len(words) == 1, words[0] == "=", strings.ContainsAny(cmd, "({"):
		return true
	}
	switch words[0] {
	case "struct", "interface", "enum", "chan", "<-chan", "func":
		return true
	}
	return false
}

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// predeclared Go functions Bash# implements: callable without a declaration.
var predeclared = map[string]bool{
	"append": true, "cap": true, "clear": true, "copy": true, "delete": true, "len": true, "make": true,
	"max": true, "min": true, "new": true, "panic": true, "print": true, "println": true, "recover": true,
}

func cutKeyword(s, kw string) (string, bool) {
	rest, ok := strings.CutPrefix(s, kw)
	if !ok || rest == "" || rest[0] != ' ' && rest[0] != '\t' {
		return "", false
	}
	return strings.TrimLeft(rest, " \t"), true
}

func identRune(r rune, first bool) bool {
	return r == '_' || unicode.IsLetter(r) || !first && unicode.IsDigit(r)
}

func leadingIdent(s string) string {
	for i, r := range s {
		if !identRune(r, i == 0) {
			return s[:i]
		}
	}
	return s
}

func validIdent(s string) bool {
	return s != "" && leadingIdent(s) == s && !goKeywords[s]
}

func leadingSelector(s string) string {
	i := 0
	for {
		part := leadingIdent(s[i:])
		if part == "" {
			return ""
		}
		i += len(part)
		if i < len(s) && s[i] == '.' {
			i++
			continue
		}
		return s[:i]
	}
}

// cutShortDeclLhs matches `a, b :=` and returns the remainder.
func cutShortDeclLhs(s string) (string, bool) {
	rest := s
	for {
		name := leadingIdent(rest)
		if name == "" || goKeywords[name] {
			return "", false
		}
		rest = strings.TrimLeft(rest[len(name):], " \t")
		if r, ok := strings.CutPrefix(rest, ","); ok {
			rest = strings.TrimLeft(r, " \t")
			continue
		}
		if r, ok := strings.CutPrefix(rest, ":="); ok {
			return r, true
		}
		return "", false
	}
}

func hasCallShape(s string) bool {
	sel := leadingSelector(s)
	return sel != "" && strings.HasPrefix(s[len(sel):], "(")
}

// funcLitPrefix: `func(` and whether the parameter list is empty.
func funcLitPrefix(s string) (lit, empty bool) {
	rest, ok := strings.CutPrefix(s, "func")
	if !ok {
		return false, false
	}
	rest = strings.TrimLeft(rest, " \t")
	if !strings.HasPrefix(rest, "(") {
		return false, false
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	if rest == "" {
		return false, false
	}
	return true, strings.HasPrefix(rest, ")")
}

// callOrLit: the operand of defer/go is a function literal or a call.
func callOrLit(rest string) bool {
	if lit, _ := funcLitPrefix(rest); lit {
		return true
	}
	sel := leadingSelector(rest)
	if sel == "" {
		return false
	}
	head, _, _ := strings.Cut(sel, ".")
	return !goKeywords[head] && strings.HasPrefix(rest[len(sel):], "(")
}

// goCallHere: a call at a command position. `f(…)` with arguments or a dotted
// callee commits; `f()` commits only for a declared or predeclared function
// followed by a statement end, because `f()` is otherwise the head of a
// shell function definition (sh/syntax bashppCallable).
func (l *BashSharpLexer) goCallHere(s string) bool {
	sel := leadingSelector(s)
	if sel == "" {
		return false
	}
	head, _, _ := strings.Cut(sel, ".")
	if goKeywords[head] {
		return false
	}
	rest := s[len(sel):]
	if !strings.HasPrefix(rest, "(") {
		return false
	}
	if !strings.HasPrefix(rest, "()") || strings.Contains(sel, ".") {
		return true
	}
	if !l.st.funcs[sel] && !predeclared[sel] {
		return false
	}
	tail := strings.TrimLeft(rest[2:], " \t")
	return tail == "" || strings.ContainsRune(";#}&|)", rune(tail[0])) || strings.HasPrefix(tail, "//")
}

// sendOrReceive: `ch <- v` and `<-ch` (sh/syntax bashppChanForm).
func sendOrReceive(s string) bool {
	if rest, ok := strings.CutPrefix(s, "<-"); ok {
		return leadingIdent(rest) != "" || strings.HasPrefix(rest, "(")
	}
	sel := leadingSelector(s)
	if sel == "" || goKeywords[sel] {
		return false
	}
	after := s[len(sel):]
	rest := strings.TrimLeft(after, " \t")
	return len(rest) < len(after) && strings.HasPrefix(rest, "<-") && !strings.HasPrefix(rest, "<--")
}

// cutAssign: an assignment operator after an operand, with a blank before
// it so that `x=1` stays a shell assignment.
func cutAssign(s string, needBlank bool) (string, bool) {
	sel := leadingSelector(s)
	if sel == "" {
		return "", false
	}
	rest := s[len(sel):]
	for strings.HasPrefix(rest, "[") {
		end := matchBracket(rest)
		if end < 0 {
			return "", false
		}
		rest = rest[end+1:]
	}
	trim := strings.TrimLeft(rest, " \t")
	if needBlank && len(trim) == len(rest) {
		return "", false
	}
	for _, op := range []string{"<<=", ">>=", "&^=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "="} {
		if r, ok := strings.CutPrefix(trim, op); ok && !strings.HasPrefix(r, "=") {
			return strings.TrimLeft(r, " \t"), true
		}
	}
	return "", false
}

// cutAssignLhs matches a Go assignment `lhs, lhs op= …` inside a Go block;
// the operator must follow the operand list after a blank.
func cutAssignLhs(s string) (string, bool) {
	rest := s
	for {
		if strings.HasPrefix(rest, "*") {
			rest = rest[1:]
		}
		sel := leadingSelector(rest)
		if sel == "" || goKeywords[sel] {
			return "", false
		}
		after := rest[len(sel):]
		for strings.HasPrefix(after, "[") {
			end := matchBracket(after)
			if end < 0 {
				return "", false
			}
			after = after[end+1:]
		}
		trim := strings.TrimLeft(after, " \t")
		if r, ok := strings.CutPrefix(trim, ","); ok {
			rest = strings.TrimLeft(r, " \t")
			continue
		}
		if len(trim) == len(after) {
			return "", false
		}
		return assignOp(trim)
	}
}

// assignOp cuts a leading assignment operator (not "==").
func assignOp(s string) (string, bool) {
	for _, op := range []string{"<<=", ">>=", "&^=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "="} {
		if r, ok := strings.CutPrefix(s, op); ok && !strings.HasPrefix(r, "=") {
			return strings.TrimLeft(r, " \t"), true
		}
	}
	return "", false
}

// cutPlainAssign matches `a, b = …` with plain identifiers.
func cutPlainAssign(s string) (string, bool) {
	rest := s
	for {
		name := leadingIdent(rest)
		if name == "" || goKeywords[name] {
			return "", false
		}
		after := rest[len(name):]
		trim := strings.TrimLeft(after, " \t")
		if r, ok := strings.CutPrefix(trim, ","); ok {
			rest = strings.TrimLeft(r, " \t")
			continue
		}
		if len(trim) == len(after) {
			return "", false
		}
		if r, ok := strings.CutPrefix(trim, "="); ok && !strings.HasPrefix(r, "=") {
			return r, true
		}
		return "", false
	}
}

func incDec(s string) bool {
	sel := leadingSelector(s)
	if sel == "" || goKeywords[sel] {
		return false
	}
	rest := s[len(sel):]
	if !strings.HasPrefix(rest, "++") && !strings.HasPrefix(rest, "--") {
		return false
	}
	tail := strings.TrimLeft(rest[2:], " \t")
	return tail == "" || strings.ContainsRune(";#}", rune(tail[0]))
}

func matchBracket(s string) int {
	depth := 0
	for i, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// goIfHeader: inside a Go block an `if` whose line carries a standalone "{"
// word and no `then` word is the Go statement; anything else is the shell
// `if` (sh/syntax bashppIf).
func goIfHeader(rest string) bool {
	brace := false
	for _, w := range lineWords(rest) {
		w = strings.TrimSuffix(w, ";")
		switch w {
		case "then":
			return false
		case "{":
			brace = true
		}
	}
	return brace
}

// goForHeader: the shell `for NAME in …`, `for NAME; do`, `for ((…))` stay
// shell; every other `for … {` is the Go statement.
func goForHeader(rest string) bool {
	if strings.HasPrefix(rest, "((") {
		return false
	}
	words := lineWords(rest)
	if len(words) == 0 {
		return false
	}
	if validShellName(strings.TrimSuffix(words[0], ";")) && (len(words) == 1 || words[1] == "in" || words[1] == "do" || strings.HasSuffix(words[0], ";")) {
		return false
	}
	for _, w := range words {
		if w == "{" || strings.HasPrefix(w, "{") {
			return true
		}
	}
	return false
}

func validShellName(s string) bool {
	if s == "" || !(s[0] == '_' || isASCIILetter(s[0])) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if c := s[i]; !(c == '_' || isASCIILetter(c) || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// lineWords splits a line on blanks, keeping quoted text together.
func lineWords(s string) []string {
	var words []string
	var cur strings.Builder
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			cur.WriteByte(c)
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
			cur.WriteByte(c)
		case c == ' ' || c == '\t':
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		case c == '#' && cur.Len() == 0:
			i = len(s)
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

const (
	decoNo = iota
	decoYes
	decoNextLine
)

// decoratorHere: a column-one `@name(` commits to a decorator unless it is
// `@name()` followed by a compound command, which is Bash's own definition of
// a function named @name (the at_function production). With nothing but a
// comment after `@name()`, the next code line decides.
func decoratorHere(s string) int {
	sel := leadingSelector(s[1:])
	if sel == "" {
		return decoNo
	}
	rest := s[1+len(sel):]
	if !strings.HasPrefix(rest, "(") {
		return decoNo
	}
	inner := strings.TrimLeft(rest[1:], " \t")
	if !strings.HasPrefix(inner, ")") {
		return decoYes
	}
	tail := strings.TrimLeft(inner[1:], " \t")
	if tail == "" || strings.HasPrefix(tail, "#") {
		return decoNextLine
	}
	if compoundStart(tail) {
		return decoNo
	}
	return decoYes
}

// nextCodeLine returns the first line at or after LA(k) that is neither
// blank nor a comment, without its leading blanks.
func (l *BashSharpLexer) nextCodeLine(k int) string {
	in := l.GetInputStream()
	for {
		var line []rune
		for ; ; k++ {
			c := in.LA(k)
			if c == antlr.TokenEOF {
				return strings.TrimLeft(string(line), " \t")
			}
			if c == '\n' {
				k++
				break
			}
			line = append(line, rune(c))
		}
		t := strings.TrimLeft(string(line), " \t")
		if t != "" && !strings.HasPrefix(t, "#") {
			return t
		}
	}
}

// compoundStart: the text starts a compound command.
func compoundStart(s string) bool {
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "(") || strings.HasPrefix(s, "[[") {
		return true
	}
	w := s
	if i := strings.IndexAny(s, " \t;"); i >= 0 {
		w = s[:i]
	}
	switch w {
	case "if", "for", "while", "until", "case", "select":
		return true
	}
	return false
}

// isAssignmentWord: NAME=…, NAME+=…, NAME[…]=….
func isAssignmentWord(w string) bool {
	i := 0
	for i < len(w) && (w[i] == '_' || isASCIILetter(w[i]) || i > 0 && w[i] >= '0' && w[i] <= '9') {
		i++
	}
	if i == 0 {
		return false
	}
	if i < len(w) && w[i] == '[' {
		end := strings.IndexByte(w[i:], ']')
		if end < 0 {
			return false
		}
		i += end + 1
	}
	if i < len(w) && w[i] == '+' {
		i++
	}
	return i < len(w) && w[i] == '='
}
