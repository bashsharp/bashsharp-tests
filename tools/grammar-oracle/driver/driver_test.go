package driver

import (
	"context"
	"fmt"
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
