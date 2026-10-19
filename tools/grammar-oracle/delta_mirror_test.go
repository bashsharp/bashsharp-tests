package grammaroracle

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The R5 mirror (plan "keep one normative source per production"): the
// delta-owned productions of the combined grammar must match
// sh/grammar/delta.ebnf, in the style of sh/grammar TestEBNFMirrorsGBNF. Every
// delta.ebnf production is classified exactly once:
//
//   - mirrored: a parser rule of the same name in BashSharpParser.g4. Its
//     references must equal the EBNF references, each EBNF reference mapped
//     through deltaLexical/deltaComposed, except for the documented
//     deltaWidened entries.
//   - lexical: implemented by named lexer tokens or primitives (README P1-P12).
//   - composed: replaced by named base-shell or imported Go productions.
//
// A new or renamed delta production, a changed reference, or a dropped rule
// fails the test.
var deltaMirrored = []string{
	"fence", "embed_line", "decorator_head", "decorator", "decorator_line", "decorated_decl",
	"function_decl", "at_function", "agentic_decl", "agentic_block", "agentic_shell_func",
	"agentic_typed_func", "agentic_typed_method", "agentic_command", "typed_func", "typed_method",
	"shell_function", "block",
}

// deltaLexical maps a lexical delta production to the grammar symbols that
// implement it; an empty list means hidden text (blanks, comments).
var deltaLexical = map[string][]string{
	"letter": {"WORD", "IDENTIFIER"}, "digit": {"WORD", "IDENTIFIER"},
	"name":            {"IDENTIFIER", "fname", "AT_WORD"},
	"selector":        {"decorator_name"},
	"format_name":     {"WORD"},
	"runner_name":     {"WORD"},
	"space":           {},
	"maybe_space":     {},
	"any_but_newline": {},
	"not_tilde_line":  {"FENCE_CLOSE"},
	"comment":         {},
	"skip_line":       {"linebreak"},
	"header_end":      {}, // newlines and comment lines before a body: linebreak, or hidden after a Go signature (README P4)
	"dq_string":       {"WORD"},
	"sq_string":       {"WORD"},
	"plain_word":      {"word_not_lbrace", "cmd_suffix"},
	"fence_mark":      {"FENCE_OPEN"},
	"fence_tail":      {"word", "FENCE_OPEN"},
	"fence_head":      {"FENCE_OPEN"},
	"fence_open":      {"FENCE_OPEN"},
	"fence_lexeme":    {"FENCE_OPEN", "FENCE_CLOSE"},
	"embed_path":      {"WORD"},
}

// deltaComposed maps a delta production to the base-shell or imported Go
// productions that replace it.
var deltaComposed = map[string][]string{
	"root":                 {"script"},
	"item":                 {"list_body", "term_list", "compound_list"},
	"bash_fragment":        {"and_or"},
	"indented_at_function": {"at_function"},
	"func_decl":            {"go_statement"},
	"func_command":         {"simple_command"},
	"paren_text":           {"arguments"},
	"bracket_text":         {"typeParameters"},
	"receiver":             {"receiver"},
	"params":               {"signature"},
	"type_params":          {"typeParameters"},
	"results":              {"signature"},
	"result_list":          {"result"},
	"result_item":          {"result"},
	"go_type":              {"type_"},
	"type_prefix":          {"type_"},
	"type_core":            {"type_"},
	"type_args":            {"typeArgs"},
}

// deltaWidened: where the combined grammar is deliberately wider than the
// delta, the EBNF reference -> the grammar reference that replaces it.
var deltaWidened = map[string]map[string]string{
	// Bash admits any compound command as a function body; sh/syntax accepts
	// `@n()` + `(( 1 ))` (tests/decorators/compound-bodies.bpp).
	"at_function":    {"block": "compound_command"},
	"shell_function": {"block": "compound_command"},
}

var (
	ebnfComment = regexp.MustCompile(`(?s)\(\*.*?\*\)`)
	ebnfProd    = regexp.MustCompile(`(?s)([a-z_]+)\s*=(.*?);\s*(?:\n|$)`)
	ebnfDrop    = regexp.MustCompile(`(?s)"[^"]*"|'[^']*'|\?[^?]*\?`)
	identRe     = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*`)
	g4Drop      = regexp.MustCompile(`(?s)/\*.*?\*/|//[^\n]*|'(?:[^'\\]|\\.)*'|\{[^{}]*\}\??`)
	lexRule     = regexp.MustCompile(`(?m)^([A-Z][A-Z_0-9]*)\s*:`)
	g4Rule      = regexp.MustCompile(`(?m)^([a-z][A-Za-z_0-9]*)\s*\n?\s*:((?s).*?)\n\s*;`)
)

func ebnfProductions(t *testing.T, src string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	src = ebnfComment.ReplaceAllString(src, "")
	for _, m := range ebnfProd.FindAllStringSubmatch(src, -1) {
		refs := map[string]bool{}
		for _, id := range identRe.FindAllString(ebnfDrop.ReplaceAllString(m[2], ""), -1) {
			refs[id] = true
		}
		if _, dup := out[m[1]]; dup {
			t.Errorf("delta.ebnf: %s defined twice", m[1])
		}
		out[m[1]] = refs
	}
	return out
}

func g4Rules(src string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	src = g4Drop.ReplaceAllString(src, "")
	for _, m := range g4Rule.FindAllStringSubmatch(src, -1) {
		refs := map[string]bool{}
		for _, id := range identRe.FindAllString(m[2], -1) {
			refs[id] = true
		}
		out[m[1]] = refs
	}
	return out
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestDeltaMirror(t *testing.T) {
	ebnfSrc, err := os.ReadFile(filepath.Join(repoRoot(t, "sh"), "grammar/delta.ebnf"))
	if err != nil {
		t.Fatal(err)
	}
	delta := ebnfProductions(t, string(ebnfSrc))
	if len(delta) < 50 {
		t.Fatalf("delta.ebnf: parsed %d productions", len(delta))
	}
	parserSrc, err := os.ReadFile("grammar/bashsharp/BashSharpParser.g4")
	if err != nil {
		t.Fatal(err)
	}
	rules := g4Rules(string(parserSrc))
	goSrc, err := os.ReadFile("grammar/go-target/GoParser.g4")
	if err != nil {
		t.Fatal(err)
	}
	goRules := g4Rules(string(goSrc))
	lexSrc, err := os.ReadFile("grammar/bashsharp/BashSharpLexer.g4")
	if err != nil {
		t.Fatal(err)
	}
	goLexSrc, err := os.ReadFile("grammar/go-target/GoLexer.g4")
	if err != nil {
		t.Fatal(err)
	}
	tokens := map[string]bool{}
	if _, block, ok := strings.Cut(string(lexSrc), "tokens {"); ok {
		block, _, _ = strings.Cut(block, "}")
		for _, id := range identRe.FindAllString(block, -1) {
			tokens[id] = true
		}
	}
	for _, m := range lexRule.FindAllStringSubmatch(string(lexSrc)+"\n"+string(goLexSrc), -1) {
		tokens[m[1]] = true
	}
	known := func(sym string) bool { return rules[sym] != nil || goRules[sym] != nil || tokens[sym] }

	mirrored := map[string]bool{}
	for _, p := range deltaMirrored {
		mirrored[p] = true
	}
	// 1. Every delta production is classified exactly once.
	for p := range delta {
		n := 0
		if mirrored[p] {
			n++
		}
		if _, ok := deltaLexical[p]; ok {
			n++
		}
		if _, ok := deltaComposed[p]; ok {
			n++
		}
		if n != 1 {
			t.Errorf("delta production %s is classified %d times, want once", p, n)
		}
	}
	for _, table := range []map[string][]string{deltaLexical, deltaComposed} {
		for p, syms := range table {
			if delta[p] == nil {
				t.Errorf("table names %s, which delta.ebnf does not define", p)
			}
			for _, s := range syms {
				if !known(s) {
					t.Errorf("%s maps to %s, which the grammar does not define", p, s)
				}
			}
		}
	}
	// 2-3. Mirrored productions: same name, same references.
	for _, p := range deltaMirrored {
		want := delta[p]
		got := rules[p]
		if want == nil || got == nil {
			t.Errorf("mirrored production %s: delta.ebnf %v, grammar %v", p, want != nil, got != nil)
			continue
		}
		for _, r := range keys(want) {
			var syms []string
			switch {
			case deltaWidened[p][r] != "":
				syms = []string{deltaWidened[p][r]}
			case mirrored[r]:
				syms = []string{r}
			case deltaLexical[r] != nil:
				syms = deltaLexical[r]
				if len(syms) == 0 {
					continue
				}
			default:
				syms = deltaComposed[r]
			}
			found := false
			for _, s := range syms {
				found = found || got[s]
			}
			if !found {
				t.Errorf("%s: delta.ebnf references %s; the grammar rule references none of %v (it has %v)", p, r, syms, keys(got))
			}
		}
		for _, r := range keys(got) {
			if !mirrored[r] || want[r] {
				continue
			}
			widened := false
			for _, w := range deltaWidened[p] {
				widened = widened || w == r
			}
			if !widened {
				t.Errorf("%s: the grammar rule references the delta production %s, which delta.ebnf does not", p, r)
			}
		}
	}
}
