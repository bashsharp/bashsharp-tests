package mutate

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"bashpp-tests/tools/grammar-oracle/oracle"
)

var (
	update      = flag.Bool("update", false, "rewrite docs/grammar-oracle/derived.tsv")
	materialize = flag.Bool("materialize", false, "write the derived input bytes under "+OutDir)
)

const (
	testsRoot = "../../.."
	tsvPath   = testsRoot + "/docs/grammar-oracle/derived.tsv"
	// deadline bounds one oracle recognition; exceeding it is a finding.
	deadline = 30 * time.Second
)

func generate(t *testing.T) ([]Parent, []Row) {
	t.Helper()
	parents, err := Parents(testsRoot)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Generate(parents)
	if err != nil {
		t.Fatal(err)
	}
	return parents, rows
}

func TestEdits(t *testing.T) {
	src := []byte("abcd")
	for _, c := range []struct {
		e    edit
		want string
		m    [][3]int
	}{
		{edit{op: "delete", at: 0}, "bcd", [][3]int{{0, 1, 3}}},
		{edit{op: "duplicate", at: 3}, "abcdd", [][3]int{{0, 0, 4}, {4, 3, 1}}},
		{edit{op: "swap", at: 1}, "acbd", [][3]int{{0, 0, 1}, {1, 2, 1}, {2, 1, 1}, {3, 3, 1}}},
		{edit{op: "insert", at: 2, token: "~~~"}, "ab~~~cd", [][3]int{{0, 0, 2}, {5, 2, 2}}},
	} {
		out, tr := c.e.apply(src)
		if string(out) != c.want || !equalMap(tr.Map, c.m) {
			t.Errorf("%+v: got %q %v, want %q %v", c.e, out, tr.Map, c.want, c.m)
		}
	}
}

func equalMap(a, b [][3]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestDerivedTSV: regeneration is deterministic and byte-identical to the
// committed derived.tsv; IDs are unique and disjoint from the manifest's;
// the CONTRACT bounds hold; every transform maps its bytes back to the
// parent input.
func TestDerivedTSV(t *testing.T) {
	parents, rows := generate(t)
	_, again := generate(t)
	got := Render(rows)
	if !bytes.Equal(got, Render(again)) {
		t.Fatal("two generations differ")
	}
	if *materialize {
		root, err := filepath.Abs(testsRoot)
		if err != nil {
			t.Fatal(err)
		}
		if err := Materialize(root, rows); err != nil {
			t.Fatal(err)
		}
	}
	if *update {
		if err := os.WriteFile(tsvPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	} else if want, err := os.ReadFile(tsvPath); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("%s is stale (%v); rerun with -update and review the diff", tsvPath, err)
	}

	manifest, err := os.ReadFile(testsRoot + "/docs/grammar-oracle/manifest.tsv")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, l := range strings.Split(string(manifest), "\n") {
		if l != "" && !strings.HasPrefix(l, "#") {
			ids[strings.Split(l, "\t")[0]] = true
		}
	}
	byID := map[string]Parent{}
	for _, p := range parents {
		byID[p.ID] = p
	}
	perParent := map[string]int{}
	mutants, probes := 0, 0
	for i, l := range strings.Split(strings.TrimSuffix(string(got[len(Header):]), "\n"), "\n") {
		f := strings.Split(l, "\t")
		if len(f) != 10 || f[1] != "derived" || f[6] != "contextual" || f[7] != "-" {
			t.Fatalf("row %d: bad schema %q", i, l)
		}
		if ids[f[0]] {
			t.Errorf("%s: duplicate or manifest id", f[0])
		}
		ids[f[0]] = true
		r := rows[i]
		p := byID[r.Parent]
		if strings.HasPrefix(r.ID, "mutant/") {
			mutants++
			perParent[r.Parent]++
			if r.ID != "mutant/"+p.ID+"/"+itoa(perParent[r.Parent]-1) || len(p.Bytes) > MaxParentBytes || p.Class != "required-positive" {
				t.Errorf("%s: id or parent outside CONTRACT bounds", r.ID)
			}
		} else {
			probes++
		}
		covered := 0
		for _, s := range r.Transform.Map {
			if !bytes.Equal(r.Bytes[s[0]:s[0]+s[2]], p.Bytes[s[1]:s[1]+s[2]]) {
				t.Errorf("%s: transform segment %v does not map to %s", r.ID, s, p.ID)
			}
			covered += s[2]
		}
		if synthetic := len(r.Bytes) - covered; synthetic < 0 || strings.HasPrefix(r.ID, "probe/") && synthetic != 0 ||
			!strings.Contains(r.Transform.Name, "insert") && synthetic != 0 {
			t.Errorf("%s: %d synthetic bytes for %q", r.ID, synthetic, r.Transform.Name)
		}
	}
	for id, n := range perParent {
		if n > MaxPerFixture {
			t.Errorf("%s: %d mutants", id, n)
		}
	}
	if mutants > MaxMutants || probes > MaxProbes || mutants == 0 || probes == 0 {
		t.Errorf("%d mutants, %d probes", mutants, probes)
	}
	t.Logf("%d parents, %d mutants, %d probes", len(parents), mutants, probes)
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

// TestDerivedOracle runs the oracle over every derived row, each under a
// deadline. An error verdict (panic), a timeout or a hang is a finding to
// fix in the oracle; reject and accept are both legitimate outcomes here.
func TestDerivedOracle(t *testing.T) {
	_, rows := generate(t)
	counts := map[string]int{}
	for _, r := range rows {
		lane, entry := "mixed", r.Dialect
		if r.Dialect == "go-unit" {
			lane, entry = "unit", "sourceFile"
		}
		done := make(chan oracle.Result, 1)
		start := time.Now()
		go func() { done <- oracle.Recognize(lane, entry, r.Bytes) }()
		var res oracle.Result
		select {
		case res = <-done:
		case <-time.After(deadline):
			t.Fatalf("%s: oracle exceeded %s", r.ID, deadline)
		}
		if res.Verdict != oracle.Accept && res.Verdict != oracle.Reject {
			t.Errorf("%s: oracle %s %v", r.ID, res.Verdict, res.Diagnostics)
		}
		if d := time.Since(start); d > deadline/3 {
			t.Logf("%s: slow, %s", r.ID, d)
		}
		kind := strings.SplitN(r.ID, "/", 2)[0]
		counts[kind+" "+r.Dialect+" "+res.Verdict]++
		counts[kind+" "+r.Kind+" "+res.Verdict]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%s: %d", k, counts[k])
	}
}
