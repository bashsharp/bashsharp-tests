package grammaroracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"bashpp-tests/tools/grammar-oracle/oracle"
)

const mixedResultsPath = "../../docs/grammar-oracle/results/mixed-oracle.jsonl"

// mixedRecord is the oracle half of a CONTRACT.md result record for the mixed
// lane; the comparison driver adds the reference half and the outcome.
type mixedRecord struct {
	FixtureID         string              `json:"fixture_id"`
	Lane              string              `json:"lane"`
	Dialect           string              `json:"dialect"`
	Policy            string              `json:"policy"`
	InputSHA256       string              `json:"input_sha256"`
	Class             string              `json:"class"`
	Expected          string              `json:"expected"`
	VerdictOracle     string              `json:"verdict_oracle"`
	BoundaryOracle    []oracle.Site       `json:"boundary_oracle"`
	DiagnosticsOracle []oracle.Diagnostic `json:"diagnostics_oracle"`
	Transform         *struct{}           `json:"transform"`
}

// mixedAgrees compares an oracle result with the authored expectation: the
// verdict, and on an accept the outermost boundary kind (CONTRACT.md PASS
// rule 3; `shell` means no extension site, `-` asserts no kind).
func mixedAgrees(e entry, r oracle.Result) bool {
	if r.Verdict != e.expected {
		return false
	}
	if r.Verdict != oracle.Accept || e.boundary == "-" {
		return true
	}
	if e.boundary == "shell" {
		return len(r.Boundary) == 0
	}
	return len(r.Boundary) > 0 && r.Boundary[0].Kind == e.boundary
}

// TestMixedLaneOracle runs every mixed-lane row that is compared (required,
// contextual and planned) through the combined grammar, and checks the
// committed oracle results in docs/grammar-oracle/results/mixed-oracle.jsonl
// (-update rewrites it). A disagreement with the authored expectation on a
// row that has one is a finding: it must be an open findings.tsv row, at the
// oracle's first error offset when the oracle rejects. The ledger never
// waives it (CONTRACT.md PASS rule).
func TestMixedLaneOracle(t *testing.T) {
	known := findings(t)
	var buf bytes.Buffer
	type tally struct{ rows, accept, reject, expAccept, expReject, agree int }
	counts := map[string]*tally{}
	for _, e := range readManifest(t) {
		if e.lane != "mixed" || e.class == "not-applicable" {
			continue
		}
		src, _, err := input(t, e.source)
		if err != nil || sum(src) != e.sha {
			t.Fatalf("%s: input %s: %v (sha256 mismatch?)", e.id, e.source, err)
		}
		r := oracle.Recognize("mixed", e.dialect, src)
		rec := mixedRecord{e.id, e.lane, e.dialect, e.policy, e.sha, e.class, e.expected, r.Verdict, r.Boundary, r.Diagnostics, nil}
		if rec.BoundaryOracle == nil {
			rec.BoundaryOracle = []oracle.Site{}
		}
		key := e.dialect + " " + e.class
		c := counts[key]
		if c == nil {
			c = &tally{}
			counts[key] = c
		}
		c.rows++
		switch r.Verdict {
		case oracle.Accept:
			c.accept++
		case oracle.Reject:
			c.reject++
		}
		switch e.expected {
		case oracle.Accept:
			c.expAccept++
		case oracle.Reject:
			c.expReject++
		}
		if e.expected == oracle.Accept || e.expected == oracle.Reject {
			if mixedAgrees(e, r) {
				c.agree++
			}
			want := e.expected
			got := r
			if r.Verdict == want && !mixedAgrees(e, r) {
				got.Verdict = "boundary"
			}
			checkLedger(t, known, e.id, want, got)
		}
		line, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(append(line, '\n'))
	}
	checkLedgerSeen(t, known, laneIDs(t, "mixed"))
	for _, key := range []string{"bash required-positive", "bash required-negative", "bash planned",
		"bashsharp required-positive", "bashsharp required-negative", "bashsharp planned", "bashsharp contextual"} {
		if c := counts[key]; c != nil {
			t.Logf("mixed %s: %d rows, oracle accept %d reject %d, expected accept %d reject %d, agree %d",
				key, c.rows, c.accept, c.reject, c.expAccept, c.expReject, c.agree)
		}
	}
	if *update {
		if err := os.WriteFile(mixedResultsPath, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(mixedResultsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("oracle results differ from %s; rerun with -update and review the diff", mixedResultsPath)
	}
}

// Recognizer behaviour pinned on small inputs, one per enumerated primitive
// of grammar/bashsharp/README.md.
func TestMixedPrimitives(t *testing.T) {
	for _, c := range []struct {
		dialect, src, verdict string
		sites                 string // fmt of the boundary
	}{
		// P1 words, P3 positions: reserved words only at a command position.
		{"bash", "echo if then fi\n", "accept", "[]"},
		{"bash", "then echo\n", "reject", ""},
		{"bash", "if true; then echo; fi\n", "accept", "[]"},
		// P2 comments start a word; Bash# adds //.
		{"bash", "echo a#b # c\n", "accept", "[]"},
		{"bashsharp", "echo a //b\nx := 1 // c\n", "accept", "[]"},
		// P5/P6 a Go region ends at its inserted semicolon.
		{"bashsharp", "x := f(1)\necho $x\n", "accept", "[]"},
		{"bash", "x := f(1)\n", "reject", ""},
		// P4/P5 a Go block's body is shell text; a standalone } closes it.
		{"bashsharp", "func f() { echo a }\n", "accept", "[{typed-func 0 19}]"},
		{"bashsharp", "func f() { echo a }}\n", "reject", ""},
		{"bash", "{ echo a }\n", "reject", ""},
		// P7 fences close at the first exact closer.
		{"bashsharp", "~~~python\nx\n~~~\n", "accept", "[{fence 0 15}]"},
		{"bashsharp", "~~~python\nx\n~~~~\n", "reject", ""},
		// An unaliased fence declares its functions: a later `f()` is a call.
		{"bashsharp", "~~~cpp\nvoid f() {}\n~~~\nf()\n", "accept", "[{fence 0 22}]"},
		{"bashsharp", "~~~cpp as x\nvoid f() {}\n~~~\nf()\n", "reject", ""},
		// P8 decorators and @name() functions.
		{"bashsharp", "@n()\nfunc f() {}\n", "accept", "[{decorator 0 4} {typed-func 5 16}]"},
		{"bashsharp", "@n()\n{ :; }\n", "accept", "[{at-function 0 11}]"},
		// P9 here-documents are opaque to the grammar.
		{"bash", "cat <<'E'\n}{ (\nE\necho\n", "accept", "[]"},
		// An unclosed here-document is a reject.
		{"bash", "cat <<-EOF\n\thi\nEEOF\n", "reject", ""},
		// NAME[ where an assignment may stand reads a matched subscript.
		{"bash", "f[[(]]\n", "accept", "[]"},
		{"bash", "a[0=1\n", "reject", ""},
		{"bash", "echo a[b\n", "accept", "[]"},
		{"bash", "f[`]\n", "reject", ""},
		// A # at a word start is a comment even holding a quote.
		{"bash", "#\" x\necho \"y\"\n", "accept", "[]"},
		// type commits on a complete type shape only; := is its own word.
		{"bashsharp", "type foobar\necho after\n", "accept", "[]"},
		{"bashsharp", "type T struct { A int\nfunc f(j T) int { return 1 }\n", "accept", "[{typed-func 22 50}]"},
		{"bashsharp", "ech:=o a; { echo; }\n", "accept", "[]"},
		// A } glued to a word does not close a Go block.
		{"bashsharp", "func f(c int) {\n  c = \"x\"}\n", "reject", ""},
		// Generic typed functions; a union constraint (S6 M11) is not the site.
		{"bashsharp", "func F[T any](x T) T {\n  return x\n}\n", "accept", "[{typed-func 0 35}]"},
		{"bashsharp", "func F[T int | float64](x T) T {\n  return x\n}\n", "reject", ""},
		// P10 typed operands.
		{"bashsharp", "func f(n int) { f($((n - 1))) }\n", "accept", "[{typed-func 0 31}]"},
		// P11 substitution bodies are scripts.
		{"bash", "x=$(if)\n", "reject", ""},
		{"bash", "x=\"$(echo \")\")\"\n", "accept", "[]"},
		{"bash", "echo ${foo\n", "reject", ""},
		{"bash", "echo $$(foo)\n", "reject", ""},
		// Bash: closers right after a compound command, brace loop bodies,
		// coproc, time and ! alone, extended globs, [[ =~ ]] operands.
		{"bash", "{ if a; then b; fi }\n", "accept", "[]"},
		{"bash", "while (a) do b; done\n", "accept", "[]"},
		{"bash", "for i in 1 2; { echo $i; }\n", "accept", "[]"},
		{"bash", "coproc name { foo; }\n", "accept", "[]"},
		{"bash", "time\n! true\n", "accept", "[]"},
		{"bash", "echo @(a|b) !(c)\n", "accept", "[]"},
		{"bash", "[[ a =~ ^(x |y)$ ]]\n", "accept", "[]"},
		{"bash", "[[ -z ]]\n", "reject", ""},
		{"bash", "case i in 3) foo; 4) bar; esac\n", "reject", ""},
		{"bash", "wh\\\nile a; do b; done\n", "accept", "[]"},
		// Bash#: a function body may start on a later line (delta header_end).
		{"bashsharp", "func f()\n# c\n{\n  echo hi\n}\n", "accept", "[{typed-func 0 26}]"},
		{"bashsharp", "const (\n\tA = iota\n)\n", "accept", "[]"},
		// Enums, keyword arguments and default parameters.
		{"bashsharp", "type Color enum {\n\tRed\n\tGreen\n}\n", "accept", "[]"},
		{"bashsharp", "func f(a int, b int = 3) {}\nf(1, b: 2)\n", "accept", "[{typed-func 0 27}]"},
	} {
		r := oracle.Recognize("mixed", c.dialect, []byte(c.src))
		if r.Verdict != c.verdict || c.sites != "" && fmt.Sprint(r.Boundary) != c.sites {
			t.Errorf("%s %q: got %s %v %v, want %s %s", c.dialect, c.src, r.Verdict, r.Boundary, r.Diagnostics, c.verdict, c.sites)
		}
	}
	if r := oracle.Recognize("mixed", "posix", []byte("echo\n")); r.Verdict != oracle.Error || !strings.Contains(r.Diagnostics[0].Msg, "unknown") {
		t.Errorf("unknown dialect: %+v", r)
	}
}

// TestMixedFences holds the required P7 fence cases of the Sprint 384
// independent review (finding 1) independently of the manifest: opener widths
// 3, 7 and 64, the first exact closer, CRLF line ends and a missing final
// newline. Each pins the oracle and the sh/syntax reference verdict and
// boundary; where they differ, the case names an open findings.tsv row.
func TestMixedFences(t *testing.T) {
	known := findings(t)
	t64 := strings.Repeat("~", 64)
	for _, c := range []struct {
		id, src    string
		oracle, sh string // fmt of the boundary of an accept
		finding    string // findings.tsv row when oracle != sh
	}{
		{"fence/width-3", "~~~python\nprint(1)\n~~~\n", "[{fence 0 22}]", "[{fence 0 22}]", ""},
		{"fence/width-7", "~~~~~~~python\nprint(1)\n~~~~~~~\n", "[{fence 0 30}]", "[{fence 0 30}]", ""},
		// README P7 bounds no width; sh/syntax agrees since de35c153b
		// (findings.tsv production/fence-width-64, fixed).
		{"fence/width-64", t64 + "python\nprint(1)\n" + t64 + "\n", "[{fence 0 144}]", "[{fence 0 144}]", "production/fence-width-64"},
		{"fence/first-exact-closer", "~~~python\nx\n~~~~\n~~~\necho after\n", "[{fence 0 20}]", "[{fence 0 20}]", ""},
		{"fence/crlf", "~~~python\r\nprint(1)\r\n~~~\r\n", "[{fence 0 24}]", "[{fence 0 24}]", ""},
		{"fence/crlf-declares", "~~~python\r\ndef g():\r\n  pass\r\n~~~\r\ng()\n", "[{fence 0 32}]", "[{fence 0 32}]", ""},
		{"fence/no-final-newline", "~~~python\nprint(1)\n~~~", "[{fence 0 22}]", "[{fence 0 22}]", ""},
	} {
		r := oracle.Recognize("mixed", "bashsharp", []byte(c.src))
		if r.Verdict != oracle.Accept || fmt.Sprint(r.Boundary) != c.oracle {
			t.Errorf("%s: got %s %v %v, want accept %s", c.id, r.Verdict, r.Boundary, r.Diagnostics, c.oracle)
		}
		if f := known[c.finding]; (c.oracle != c.sh) != (f != nil && f.status == "open") {
			t.Errorf("%s: oracle %s, sh/syntax %s, findings.tsv row %q: want an open row exactly when they differ", c.id, c.oracle, c.sh, c.finding)
		}
	}
	// A "\r" not followed by "\n" is an ordinary byte: this closer does not
	// close, and the fence is unclosed (sh/syntax: unclosed source block).
	if r := oracle.Recognize("mixed", "bashsharp", []byte("~~~python\r\nprint(1)\r\n~~~\r")); r.Verdict != oracle.Reject {
		t.Errorf("fence/crlf-lone-cr: got %s %v, want reject", r.Verdict, r.Boundary)
	}
}
