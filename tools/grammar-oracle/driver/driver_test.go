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
