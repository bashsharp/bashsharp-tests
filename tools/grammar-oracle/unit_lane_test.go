package grammaroracle

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"bashpp-tests/tools/grammar-oracle/oracle"
)

const (
	unitResultsPath = "../../docs/grammar-oracle/results/unit-oracle.jsonl"
	findingsPath    = "../../docs/grammar-oracle/findings.tsv"
)

var update = flag.Bool("update", false, "rewrite docs/grammar-oracle/results/unit-oracle.jsonl")

// oracleRecord is the oracle half of a CONTRACT.md result record. The
// reference half (verdict_reference, boundary_reference, …) and outcome are
// filled in by the comparison driver.
type oracleRecord struct {
	FixtureID         string              `json:"fixture_id"`
	Lane              string              `json:"lane"`
	Dialect           string              `json:"dialect"`
	Policy            string              `json:"policy"`
	InputSHA256       string              `json:"input_sha256"`
	Class             string              `json:"class"`
	Expected          string              `json:"expected"`
	VerdictOracle     string              `json:"verdict_oracle"`
	BoundaryOracle    []span              `json:"boundary_oracle"`
	DiagnosticsOracle []oracle.Diagnostic `json:"diagnostics_oracle"`
	Transform         *struct{}           `json:"transform"`
}

type span struct {
	Kind  string `json:"kind"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type finding struct {
	offset int
	status string // open | fixed:<patch> | fixed:bashsharpparser/<file> | fixed:sh@<commit>
	seen   bool
}

// findings reads docs/grammar-oracle/findings.tsv, keyed by fixture_id.
func findings(t *testing.T) map[string]*finding {
	t.Helper()
	data, err := os.ReadFile(findingsPath)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*finding{}
	for _, l := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Split(l, "\t")
		if len(f) != 9 {
			t.Fatalf("findings.tsv: %d fields, want 9: %q", len(f), l)
		}
		fd := &finding{status: f[4]}
		if _, err := fmt.Sscan(f[2], &fd.offset); err != nil {
			t.Fatalf("findings.tsv: offset %q: %v", f[2], err)
		}
		if p, ok := strings.CutPrefix(fd.status, "fixed:sh@"); ok {
			// A production defect repaired in sh, at a commit.
			if len(p) < 7 || strings.Trim(p, "0123456789abcdef") != "" {
				t.Errorf("findings.tsv %s: status %q: want an sh commit", f[0], fd.status)
			}
		} else if p, ok := strings.CutPrefix(fd.status, "fixed:"); ok {
			// A grammar patch, or a hand-written lexer or parser file.
			path := "grammar/go-target/patches/" + p
			if strings.HasPrefix(p, "bashsharpparser/") {
				path = p
			}
			if _, err := os.Stat(path); err != nil {
				t.Errorf("findings.tsv %s: %v", f[0], err)
			}
		} else if fd.status != "open" {
			t.Errorf("findings.tsv %s: status %q", f[0], fd.status)
		}
		out[f[0]] = fd
	}
	return out
}

// checkLedger: an oracle verdict that differs from want must be an open
// finding (with the same first-error offset when the oracle rejects); an open
// finding must still disagree; a fixed one must agree.
func checkLedger(t *testing.T, known map[string]*finding, id, want string, r oracle.Result) {
	t.Helper()
	f := known[id]
	if f != nil {
		f.seen = true
	}
	switch open := f != nil && f.status == "open"; {
	case r.Verdict == want && open:
		t.Errorf("%s: oracle agrees now (%s); close its findings.tsv row", id, want)
	case r.Verdict == want:
	case !open:
		msg := ""
		if len(r.Diagnostics) > 0 {
			d := r.Diagnostics[0]
			msg = fmt.Sprintf(" at offset %d (%d:%d) %s", d.Offset, d.Line, d.Col, d.Msg)
		}
		t.Errorf("%s: oracle %s, want %s%s: unrecorded (or regressed) finding", id, r.Verdict, want, msg)
	case r.Verdict == oracle.Reject && r.Diagnostics[0].Offset != f.offset:
		t.Errorf("%s: first error now at offset %d, findings.tsv says %d", id, r.Diagnostics[0].Offset, f.offset)
	}
}

// checkLedgerSeen fails on the findings.tsv rows a test owns (mine) that it
// did not visit.
func checkLedgerSeen(t *testing.T, known map[string]*finding, mine func(id string) bool) {
	t.Helper()
	for id, f := range known {
		if mine(id) && !f.seen {
			t.Errorf("findings.tsv: %s matches no input", id)
		}
	}
}

// laneIDs is the set of manifest fixture ids in lane.
func laneIDs(t *testing.T, lane string) func(string) bool {
	ids := map[string]bool{}
	for _, e := range readManifest(t) {
		if e.lane == lane {
			ids[e.id] = true
		}
	}
	return func(id string) bool { return ids[id] }
}

// TestUnitLaneOracle runs every unit-lane required-positive row through the
// independent recognizer and checks the committed oracle results. A reject is
// a finding: it must be an open findings.tsv row with its first error offset.
// The ledger never waives it (CONTRACT.md PASS rule); it only keeps this gate
// honest about what is known.
func TestUnitLaneOracle(t *testing.T) {
	known := findings(t)
	var buf bytes.Buffer
	accept, reject := 0, 0
	for _, e := range readManifest(t) {
		if e.lane != "unit" || e.class != "required-positive" {
			continue
		}
		src, _, err := input(t, e.source)
		if err != nil || sum(src) != e.sha {
			t.Fatalf("%s: input %s: %v (sha256 mismatch?)", e.id, e.source, err)
		}
		r := oracle.Recognize("unit", "sourceFile", src)
		rec := oracleRecord{e.id, e.lane, e.dialect, e.policy, e.sha, e.class, e.expected, r.Verdict, []span{}, r.Diagnostics, nil}
		if r.Verdict == oracle.Accept {
			accept++
			rec.BoundaryOracle = []span{{"go-unit", 0, len(src)}}
		} else {
			reject++
		}
		checkLedger(t, known, e.id, e.expected, r)
		line, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(append(line, '\n'))
	}
	checkLedgerSeen(t, known, laneIDs(t, "unit"))
	t.Logf("unit lane required-positive: %d rows, oracle accept %d, reject %d (expected accept %d)", accept+reject, accept, reject, accept+reject)
	if *update {
		if err := os.WriteFile(unitResultsPath, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(unitResultsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("oracle results differ from %s; rerun with -update and review the diff", unitResultsPath)
	}
}
