package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bashpp-tests/tools/grammar-oracle/oracle"
)

func TestGateFailureClasses(t *testing.T) {
	f := Fixture{ID: "a", Lane: "mixed", SHA: strings.Repeat("a", 64), Dialect: "bash", Policy: "sh-syntax", Class: "required-positive", Expected: "accept", Boundary: "shell"}
	good := classify(f, oracle.Result{Verdict: "accept"}, &Reference{ID: f.ID, SHA: f.SHA, Verdict: "accept"})
	if err := Evaluate([]Fixture{f}, []Record{good}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		records []Record
	}{
		{"missing result", nil},
		{"duplicate result", []Record{good, good}},
		{"hash mismatch", []Record{func() Record { r := good; r.SHA = "wrong"; return r }()}},
		{"disagreement", []Record{func() Record { r := good; r.Reference = "reject"; r.Outcome = "disagree"; return r }()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Evaluate([]Fixture{f}, tt.records); err == nil {
				t.Fatal("gate passed")
			}
		})
	}
	t.Run("timeout", func(t *testing.T) {
		data := []byte(fmt.Sprintf("{\"fixture_id\":\"a\",\"input_sha256\":%q,\"verdict\":\"accept\"}\n{\"done\":true,\"count\":1}\n", f.SHA))
		refs, err := Reconcile(data, []Fixture{f}, context.DeadlineExceeded)
		if err == nil {
			t.Fatal("timed out adapter passed")
		}
		r, ok := refs[f.ID]
		if ok {
			t.Fatalf("timed out adapter produced trusted record: %+v", r)
		}
		missing := classify(f, oracle.Result{Verdict: "accept"}, nil)
		if err := Evaluate([]Fixture{f}, []Record{missing}); err == nil {
			t.Fatal("gate passed timed out adapter")
		}
	})
}

func TestDerivedProbeMapsToOriginal(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	original, err := os.ReadFile(filepath.Join(root, "tour", "_content", "tour", "basics", "functions.bsh"))
	if err != nil {
		t.Fatal(err)
	}
	f := Fixture{ID: "probe/tour/basics/functions.bsh/1", Lane: "derived", SHA: "probe", Dialect: "bashsharp", Policy: "sh-syntax", Class: "contextual", Expected: "-", Boundary: "-", Original: original,
		Transform: &Transform{Name: "extract func declaration, lines 7-9", Map: [][3]int{{0, 45, 45}}}}
	site := oracle.Site{Kind: "typed-func", Start: 0, End: 44}
	r := classify(f, oracle.Result{Verdict: "accept", Boundary: []oracle.Site{site}}, &Reference{Verdict: "accept", Boundary: []oracle.Site{site}})
	if len(r.BoundaryOracle) != 1 || r.BoundaryOracle[0].Start != 45 || r.BoundaryOracle[0].End != 89 || r.BoundaryReference[0] != r.BoundaryOracle[0] {
		t.Fatalf("mapped boundary = %+v / %+v, want [45,89)", r.BoundaryOracle, r.BoundaryReference)
	}
}

func TestCorruptedEmbedPayloadFailsGate(t *testing.T) {
	sh, err := roots("sh", "")
	if err != nil {
		t.Skip("sh reference checkout unavailable")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "tests", "go-delta")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	script := []byte("embed go \"./I03-main.go\"\necho done\n")
	corrupt := []byte("package main\nfunc main( {\n")
	if err := os.WriteFile(filepath.Join(dir, "I03.bsh"), script, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "I03-main.go"), corrupt, 0644); err != nil {
		t.Fatal(err)
	}
	docs := filepath.Join(root, "docs", "grammar-oracle")
	if err := os.MkdirAll(filepath.Join(docs, "results"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "# fixture_id\tlane\tsource\tsha256\tdialect\tpolicy\tclass\texpected\tboundary\treason\n" +
		fmt.Sprintf("s6/I03\tisland\ttests:tests/go-delta/I03.bsh\t%s\tbashsharp\tsh-syntax\trequired-positive\taccept\tembed-line\tfixture\n", digest(script))
	if err := os.WriteFile(filepath.Join(docs, "manifest.tsv"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	findings := filepath.Join(docs, "findings.tsv")
	if err := os.WriteFile(findings, nil, 0644); err != nil {
		t.Fatal(err)
	}
	s, err := Run(Options{Root: root, SHRoot: sh, Toolchain: "go1.27.2", ComparePath: filepath.Join(docs, "results", "compare.jsonl"), ReportPath: filepath.Join(docs, "results", "report.md"), FindingsPath: findings})
	if err != nil {
		t.Fatal(err)
	}
	if s.Pass || len(s.Records) != 2 {
		t.Fatalf("corrupted Go payload passed or was not compared: pass=%t records=%d issues=%v", s.Pass, len(s.Records), s.Issues)
	}
	p := s.Records[1]
	if p.ID != "s6/I03#payload-1" || p.Lane != "island-payload" || p.Oracle != "reject" || p.Reference != "reject" || p.Class != "required-positive" {
		t.Fatalf("payload result = %+v", p)
	}
	if err := Evaluate([]Fixture{{ID: "s6/I03", SHA: digest(script), Lane: "island", Dialect: "bashsharp", Policy: "sh-syntax", Class: "required-positive", Expected: "accept", Boundary: "embed-line"},
		{ID: p.ID, SHA: p.SHA, Lane: p.Lane, Dialect: p.Dialect, Policy: p.Policy, Class: p.Class, Expected: p.Expected, Boundary: "go-unit"}}, s.Records); err == nil {
		t.Fatal("gate accepted corrupted payload")
	}
}

func TestFencePayloadGate(t *testing.T) {
	sh, err := roots("sh", "")
	if err != nil {
		t.Skip("sh reference checkout unavailable")
	}
	for _, tc := range []struct {
		name, body string
		pass       bool
	}{
		{"longer tilde line stays body", "package main\nfunc valid() {}\n~~~~\nfunc main( {\n", false},
		{"raw string with longer tilde line", "package main\nvar text = `\n~~~~\n`\nfunc main() {}\n", true},
		{"comment before package clause", "// package comment\npackage main\nfunc main() {}\n", true},
		{"block comment and blanks before package clause", "\n/* package comment */\n\npackage main\nfunc main() {}\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "tests", "go-delta")
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			script := []byte("~~~go\n" + tc.body + "~~~\n")
			if err := os.WriteFile(filepath.Join(dir, "fence.bsh"), script, 0644); err != nil {
				t.Fatal(err)
			}
			docs := filepath.Join(root, "docs", "grammar-oracle")
			if err := os.MkdirAll(filepath.Join(docs, "results"), 0755); err != nil {
				t.Fatal(err)
			}
			manifest := "# fixture_id\tlane\tsource\tsha256\tdialect\tpolicy\tclass\texpected\tboundary\treason\n" +
				fmt.Sprintf("fence\tisland\ttests:tests/go-delta/fence.bsh\t%s\tbashsharp\tsh-syntax\trequired-positive\taccept\tfence\tfixture\n", digest(script))
			if err := os.WriteFile(filepath.Join(docs, "manifest.tsv"), []byte(manifest), 0644); err != nil {
				t.Fatal(err)
			}
			findings := filepath.Join(docs, "findings.tsv")
			if err := os.WriteFile(findings, nil, 0644); err != nil {
				t.Fatal(err)
			}
			s, err := Run(Options{Root: root, SHRoot: sh, Toolchain: "go1.27.2", ComparePath: filepath.Join(docs, "results", "compare.jsonl"), ReportPath: filepath.Join(docs, "results", "report.md"), FindingsPath: findings})
			if err != nil {
				t.Fatal(err)
			}
			if s.Pass != tc.pass || len(s.Records) != 2 {
				t.Fatalf("gate pass=%t records=%d issues=%v, want pass=%t and two records", s.Pass, len(s.Records), s.Issues, tc.pass)
			}
			if parent := s.Records[0]; parent.Oracle != "accept" || parent.Reference != "accept" {
				t.Fatalf("fence verdicts oracle=%s reference=%s, want accept", parent.Oracle, parent.Reference)
			}
			p := s.Records[1]
			if p.ID != "fence#payload-1" || p.SHA != digest([]byte(tc.body)) || p.Lane != "island-payload" || p.Class != "required-positive" {
				t.Fatalf("payload extraction = %+v, want exact body", p)
			}
			wantVerdict := "accept"
			if !tc.pass {
				wantVerdict = "reject"
			}
			if p.Oracle != wantVerdict || p.Reference != wantVerdict {
				t.Fatalf("payload verdicts oracle=%s reference=%s, want %s", p.Oracle, p.Reference, wantVerdict)
			}
		})
	}
}

func TestAdapterIDFailures(t *testing.T) {
	f := Fixture{ID: "a", SHA: "sha"}
	for _, body := range []string{
		"{\"done\":true,\"count\":1}\n",
		"{\"fixture_id\":\"a\",\"input_sha256\":\"sha\",\"verdict\":\"accept\"}\n{\"fixture_id\":\"a\",\"input_sha256\":\"sha\",\"verdict\":\"accept\"}\n{\"done\":true,\"count\":1}\n",
		"{\"fixture_id\":\"unknown\",\"input_sha256\":\"sha\",\"verdict\":\"accept\"}\n{\"done\":true,\"count\":1}\n",
		"{\"fixture_id\":\"a\",\"input_sha256\":\"bad\",\"verdict\":\"accept\"}\n{\"done\":true,\"count\":1}\n",
	} {
		if _, err := Reconcile([]byte(body), []Fixture{f}, nil); err == nil {
			t.Fatalf("accepted faulty adapter output %q", body)
		}
	}
}

// CONTRACT PASS rules 1 and 3 cover manifest rows and the frozen mutant set:
// a probe disagreement is reported but never fails the gate, a mutant
// disagreement does.
func TestProbeDisagreementDoesNotBlock(t *testing.T) {
	derived := func(id string) Fixture {
		return Fixture{ID: id, Lane: "derived", SHA: strings.Repeat("b", 64), Dialect: "bashsharp", Policy: "sh-syntax", Class: "contextual", Expected: "-", Boundary: "-"}
	}
	ref := func(f Fixture) *Reference {
		return &Reference{ID: f.ID, SHA: f.SHA, Verdict: "reject", Diagnostics: []Diagnostic{{Offset: 0}}, Phase: "syntax"}
	}
	probe, mutant := derived("probe/x/0"), derived("mutant/x/0")
	p := classify(probe, oracle.Result{Verdict: "accept"}, ref(probe))
	m := classify(mutant, oracle.Result{Verdict: "accept"}, ref(mutant))
	if p.Outcome != "disagree" || m.Outcome != "disagree" {
		t.Fatalf("outcomes %s %s, want disagree", p.Outcome, m.Outcome)
	}
	if err := Evaluate([]Fixture{probe}, []Record{p}); err != nil {
		t.Fatalf("probe disagreement failed the gate: %v", err)
	}
	if got := Count([]Fixture{probe}, []Record{p}); got != (Tally{Probes: 1}) {
		t.Fatalf("probe tally %+v", got)
	}
	if err := Evaluate([]Fixture{probe, mutant}, []Record{p, m}); err == nil {
		t.Fatal("mutant disagreement passed the gate")
	}
	if got := Count([]Fixture{probe, mutant}, []Record{p, m}); got != (Tally{Required: 1, Probes: 1}) {
		t.Fatalf("tally %+v", got)
	}
	agree := classify(mutant, oracle.Result{Verdict: "reject"}, ref(mutant))
	if err := Evaluate([]Fixture{mutant}, []Record{agree}); err != nil {
		t.Fatalf("agreeing mutant failed the gate: %v", err)
	}
	// A probe still needs a complete result.
	if err := Evaluate([]Fixture{probe}, []Record{classify(probe, oracle.Result{Verdict: "accept"}, nil)}); err == nil {
		t.Fatal("missing probe result passed the gate")
	}
}

// A known divergence passes only while its exact observed pair holds.
func TestKnownDivergencePinned(t *testing.T) {
	f := Fixture{ID: "mutant/startsite/subscript-dash/0", Lane: "derived", SHA: strings.Repeat("c", 64), Dialect: "bash", Policy: "sh-syntax", Class: "contextual", Expected: "-", Boundary: "-"}
	ref := func(verdict string, off int) *Reference {
		r := &Reference{ID: f.ID, SHA: f.SHA, Verdict: verdict, Phase: "syntax"}
		if verdict == "reject" {
			r.Diagnostics = []Diagnostic{{Offset: off}}
		}
		return r
	}
	held := classify(f, oracle.Result{Verdict: "accept"}, ref("reject", 2))
	if err := Evaluate([]Fixture{f}, []Record{held}); err != nil {
		t.Fatalf("pinned pair failed: %v", err)
	}
	if got := Count([]Fixture{f}, []Record{held}); got != (Tally{Known: 1}) {
		t.Fatalf("tally %+v", got)
	}
	for name, r := range map[string]Record{
		"reference accepts": classify(f, oracle.Result{Verdict: "accept"}, ref("accept", 0)),
		"oracle rejects":    classify(f, oracle.Result{Verdict: "reject", Diagnostics: []oracle.Diagnostic{{Offset: 2}}}, ref("reject", 2)),
		"offset moves":      classify(f, oracle.Result{Verdict: "accept"}, ref("reject", 3)),
	} {
		if err := Evaluate([]Fixture{f}, []Record{r}); err == nil {
			t.Errorf("%s: changed pair passed the gate", name)
		}
	}
}

// gcsyntax accepts, the oracle rejects and the production pipeline rejects:
// semantic. Without the production reject it stays a blocking disagreement.
func TestGoUnitProductionSemantic(t *testing.T) {
	f := Fixture{ID: "mutant/u/0", Lane: "derived", SHA: strings.Repeat("d", 64), Dialect: "go-unit", Policy: "gcsyntax", Class: "contextual", Expected: "-", Boundary: "-"}
	reject, accept := "reject", "accept"
	o := oracle.Result{Verdict: "reject", Diagnostics: []oracle.Diagnostic{{Offset: 5}}}
	ref := func(prod *string) *Reference {
		return &Reference{ID: f.ID, SHA: f.SHA, Verdict: "accept", Boundary: []oracle.Site{{Kind: "go-unit", End: 10}}, Production: prod, Phase: "syntax"}
	}
	r := classify(f, o, ref(&reject))
	if r.Category != "semantic" || r.Outcome != "agree" {
		t.Fatalf("got %s/%s, want semantic/agree", r.Category, r.Outcome)
	}
	if err := Evaluate([]Fixture{f}, []Record{r}); err != nil {
		t.Fatal(err)
	}
	for _, prod := range []*string{&accept, nil} {
		r := classify(f, o, ref(prod))
		if r.Category != "syntax" || r.Outcome != "disagree" {
			t.Errorf("production %v: got %s/%s, want syntax/disagree", prod, r.Category, r.Outcome)
		}
		if err := Evaluate([]Fixture{f}, []Record{r}); err == nil {
			t.Errorf("production %v: disagreement passed the gate", prod)
		}
	}
	// The rule is reference-side only: an oracle accept is never semantic.
	if r := classify(f, oracle.Result{Verdict: "accept"}, &Reference{ID: f.ID, SHA: f.SHA, Verdict: "reject", Production: &reject, Phase: "syntax"}); r.Category != "syntax" {
		t.Errorf("oracle accept, reference reject: got %s", r.Category)
	}
}
