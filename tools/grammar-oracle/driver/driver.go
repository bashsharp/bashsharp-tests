// Package driver compares the independent grammar oracle with the sh adapters.
package driver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"bashpp-tests/tools/grammar-oracle/mutate"
	"bashpp-tests/tools/grammar-oracle/oracle"
)

type Fixture struct {
	ID, Lane, Source, SHA, Dialect, Policy, Class, Expected, Boundary, Reason string
	OriginalSource                                                            string
	Bytes                                                                     []byte
	Original                                                                  []byte
	Transform                                                                 *Transform
	Path                                                                      string
}
type Diagnostic struct {
	Offset    int    `json:"offset"`
	Line      int    `json:"line"`
	Col       int    `json:"col"`
	Char      int    `json:"char"`
	Msg       string `json:"msg"`
	Synthetic bool   `json:"synthetic,omitempty"`
	Logical   *struct {
		File string `json:"file"`
		Line int    `json:"line"`
		Col  int    `json:"col"`
	} `json:"logical,omitempty"`
}
type Reference struct {
	ID          string        `json:"fixture_id"`
	SHA         string        `json:"input_sha256"`
	Verdict     string        `json:"verdict"`
	Boundary    []oracle.Site `json:"boundary"`
	Diagnostics []Diagnostic  `json:"diagnostics"`
	Production  *string       `json:"production_verdict"`
	Phase       string        `json:"phase"`
	Reason      string        `json:"reason,omitempty"`
	Done        bool          `json:"done,omitempty"`
	Count       int           `json:"count,omitempty"`
}
type Record struct {
	ID                   string        `json:"fixture_id"`
	Lane                 string        `json:"lane"`
	Dialect              string        `json:"dialect"`
	Policy               string        `json:"policy"`
	SHA                  string        `json:"input_sha256"`
	Class                string        `json:"class"`
	Expected             string        `json:"expected"`
	Oracle               string        `json:"verdict_oracle"`
	Reference            string        `json:"verdict_reference"`
	BoundaryOracle       []oracle.Site `json:"boundary_oracle"`
	BoundaryReference    []oracle.Site `json:"boundary_reference"`
	DiagnosticsOracle    []Diagnostic  `json:"diagnostics_oracle"`
	DiagnosticsReference []Diagnostic  `json:"diagnostics_reference"`
	Production           *string       `json:"production_verdict"`
	Transform            *Transform    `json:"transform"`
	Outcome              string        `json:"outcome"`
	Category             string        `json:"category"`
}
type Options struct {
	Root, SHRoot, Toolchain               string
	DerivedRoot                           string // tests root for derived.tsv sources; default Root
	Timeout                               time.Duration
	ComparePath, ReportPath, FindingsPath string
}
type Summary struct {
	Records   []Record
	Pass      bool
	Issues    []string
	Toolchain string
	Tally     Tally
}

func readRows(path string) ([]Fixture, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []Fixture
	seen := map[string]bool{}
	for n, l := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Split(l, "\t")
		if len(f) != 10 {
			return nil, fmt.Errorf("%s line %d: want 10 columns", filepath.Base(path), n+1)
		}
		if seen[f[0]] {
			return nil, fmt.Errorf("duplicate fixture %s", f[0])
		}
		seen[f[0]] = true
		out = append(out, Fixture{ID: f[0], Lane: f[1], Source: f[2], SHA: f[3], Dialect: f[4], Policy: f[5], Class: f[6], Expected: f[7], Boundary: f[8], Reason: f[9]})
	}
	return out, nil
}

// Reconcile fails closed on every adapter protocol fault. A corrupted batch
// cannot provide a trustworthy result for any of its requested fixtures.
func Reconcile(data []byte, requested []Fixture, processErr error) (map[string]Reference, error) {
	want := map[string]string{}
	for _, f := range requested {
		want[f.ID] = f.SHA
	}
	got := map[string]Reference{}
	done := false
	count := 0
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Buffer(make([]byte, 1<<20), 64<<20)
	var faults []string
	for s.Scan() {
		if done {
			faults = append(faults, "data after done")
			continue
		}
		var r Reference
		if err := json.Unmarshal(s.Bytes(), &r); err != nil {
			faults = append(faults, "malformed JSON")
			continue
		}
		if r.Done {
			done = true
			if r.Count != len(requested) {
				faults = append(faults, "done count mismatch")
			}
			continue
		}
		count++
		sha, ok := want[r.ID]
		if !ok {
			faults = append(faults, "unknown ID "+r.ID)
			continue
		}
		if _, ok := got[r.ID]; ok {
			faults = append(faults, "duplicate ID "+r.ID)
			continue
		}
		if r.SHA != sha {
			faults = append(faults, "hash mismatch "+r.ID)
		}
		if r.Verdict != "accept" && r.Verdict != "reject" && r.Verdict != "error" {
			faults = append(faults, "invalid verdict "+r.ID)
		}
		got[r.ID] = r
	}
	if err := s.Err(); err != nil {
		faults = append(faults, "read error")
	}
	if !done {
		faults = append(faults, "missing done")
	}
	if count != len(requested) {
		faults = append(faults, "short or excess output")
	}
	if len(got) != len(requested) {
		faults = append(faults, "ID set mismatch")
	}
	if processErr != nil {
		faults = append(faults, "adapter failure: "+processErr.Error())
	}
	if len(faults) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(faults, "; "))
	}
	return got, nil
}

func sameSites(a, b []oracle.Site) bool {
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
func outer(s []oracle.Site, expected string) bool {
	if expected == "-" {
		return true
	}
	if expected == "shell" {
		return len(s) == 0
	}
	if expected == "go-unit" {
		return len(s) == 1 && s[0].Kind == "go-unit"
	}
	return len(s) > 0 && s[0].Kind == expected
}
func classify(f Fixture, o oracle.Result, r *Reference) Record {
	rec := Record{ID: f.ID, Lane: f.Lane, Dialect: f.Dialect, Policy: f.Policy, SHA: f.SHA, Class: f.Class, Expected: f.Expected, Oracle: o.Verdict, BoundaryOracle: o.Boundary, DiagnosticsOracle: oracleDiagnostics(o.Diagnostics), BoundaryReference: []oracle.Site{}, DiagnosticsReference: []Diagnostic{}, Transform: f.Transform, Category: "syntax"}
	if rec.BoundaryOracle == nil {
		rec.BoundaryOracle = []oracle.Site{}
	}
	if rec.DiagnosticsOracle == nil {
		rec.DiagnosticsOracle = []Diagnostic{}
	}
	if f.Class == "not-applicable" {
		rec.Oracle = ""
		rec.Outcome = "agree"
		return rec
	}
	if r == nil {
		rec.Outcome = "missing"
		mapRecord(&rec, f)
		return rec
	}
	rec.Reference = r.Verdict
	rec.BoundaryReference = r.Boundary
	rec.DiagnosticsReference = r.Diagnostics
	rec.Production = r.Production
	if rec.BoundaryReference == nil {
		rec.BoundaryReference = []oracle.Site{}
	}
	if rec.DiagnosticsReference == nil {
		rec.DiagnosticsReference = []Diagnostic{}
	}
	if o.Verdict == "error" || r.Verdict == "error" {
		rec.Outcome = "error"
		mapRecord(&rec, f)
		return rec
	}
	if r.Phase == "context" && (r.Verdict == "reject" || r.Production != nil && *r.Production == "reject" && r.Verdict == "accept") {
		rec.Category = "semantic"
		rec.Outcome = "agree"
		mapRecord(&rec, f)
		return rec
	}
	// gcsyntax accepts what the oracle refuses, and the full production
	// pipeline (Load: go/types and lowering) refuses it too: the refusal is
	// semantic, decided on reference-side evidence alone.
	if f.Dialect == "go-unit" && r.Verdict == "accept" && o.Verdict == "reject" && r.Production != nil && *r.Production == "reject" {
		rec.Category = "semantic"
		rec.Outcome = "agree"
		mapRecord(&rec, f)
		return rec
	}
	if o.Verdict != r.Verdict {
		rec.Outcome = "disagree"
		mapRecord(&rec, f)
		return rec
	}
	if o.Verdict == "accept" && !sameSites(o.Boundary, r.Boundary) {
		rec.Category = "boundary"
		rec.Outcome = "disagree"
		mapRecord(&rec, f)
		return rec
	}
	rec.Outcome = "agree"
	mapRecord(&rec, f)
	return rec
}

// knownDivergence is the exact observed pair on a mutant where the oracle
// follows bash -n and sh/syntax does not, as the sh/grammar
// TestKnownDivergences cases are. The sh static parser assumes an array index
// is an arithmetic expression (sh README "Caveats"); bash -n accepts these
// subscripts. A change on either side fails the gate, so a fix or a
// regression is visible.
type knownDivergence struct {
	Oracle, Reference string
	Offset            int // the reference's first error offset
}

var knownDivergences = map[string]knownDivergence{
	"mutant/startsite/generic-inst-bare-ptr/4": {"accept", "reject", 2}, // f[**T]
	"mutant/startsite/subscript-dash/0":        {"accept", "reject", 2}, // f[-]x
	"mutant/startsite/subscript-empty/1":       {"accept", "reject", 1}, // f[ LF ]
}

func (k knownDivergence) holds(r Record) bool {
	return r.Oracle == k.Oracle && r.Reference == k.Reference && r.Category == "syntax" &&
		len(r.DiagnosticsReference) > 0 && r.DiagnosticsReference[0].Offset == k.Offset
}

// Kind is how a fixture counts toward the gate: "required" (manifest rows
// that are neither planned nor not-applicable, and the frozen mutant set),
// "probe" (derived probes: non-blocking, CONTRACT PASS rules 1 and 3 cover
// neither), "known-divergence", "planned" or "not-applicable".
func Kind(f Fixture) string {
	derived := f.Lane == "derived" || strings.HasPrefix(f.ID, "probe/") || strings.HasPrefix(f.ID, "mutant/")
	switch {
	case f.Class == "not-applicable" || f.Class == "planned":
		return f.Class
	case derived && strings.HasPrefix(f.ID, "probe/"):
		return "probe"
	case derived:
		if _, ok := knownDivergences[f.ID]; ok {
			return "known-divergence"
		}
	}
	return "required"
}

func valid(f Fixture, r Record) bool {
	if f.Class == "not-applicable" || f.Class == "planned" {
		return true
	}
	if r.Outcome == "missing" || r.Outcome == "error" || r.Oracle == "error" || r.Reference == "error" {
		return false
	}
	switch Kind(f) {
	case "probe":
		return true
	case "known-divergence":
		return knownDivergences[f.ID].holds(r)
	}
	if f.Lane == "derived" || strings.HasPrefix(f.ID, "mutant/") {
		// A mutant has no authored expectation: the two sides agree, with
		// equal site lists on accept, or the difference is semantic.
		return r.Outcome == "agree"
	}
	if f.Class == "contextual" {
		return r.Reference == f.Expected && r.Outcome != "disagree"
	}
	if r.Outcome != "agree" || r.Oracle != r.Reference {
		return false
	}
	if r.Oracle == "accept" && !sameSites(r.BoundaryOracle, r.BoundaryReference) {
		return false
	}
	if f.Class == "required-positive" || f.Class == "required-negative" {
		if r.Oracle != f.Expected || r.Reference != f.Expected {
			return false
		}
	}
	if r.Oracle == "accept" && (!outer(r.BoundaryOracle, f.Boundary) || !outer(r.BoundaryReference, f.Boundary)) {
		return false
	}
	return true
}

// Tally counts the comparison by how each record counts toward the gate.
// Required counts every record that fails the PASS rule other than a missing
// or error one; Probes, Planned and Known count reported, non-blocking
// differences (a known divergence counts only while its pair holds).
type Tally struct{ Required, Probes, Planned, Known, Missing int }

func Count(rows []Fixture, records []Record) Tally {
	fixtures := map[string]Fixture{}
	for _, f := range rows {
		fixtures[f.ID] = f
	}
	var t Tally
	for _, r := range records {
		f := fixtures[r.ID]
		switch {
		case r.Outcome == "missing" || r.Outcome == "error" || r.Oracle == "error" || r.Reference == "error":
			if f.Class != "planned" {
				t.Missing++
			}
		case !valid(f, r):
			t.Required++
		case r.Outcome != "disagree":
		case Kind(f) == "probe":
			t.Probes++
		case Kind(f) == "planned":
			t.Planned++
		case Kind(f) == "known-divergence":
			t.Known++
		}
	}
	return t
}

func adapter(ctx context.Context, sh, tool, dir, kind string, requested []Fixture) (map[string]Reference, error) {
	if len(requested) == 0 {
		return map[string]Reference{}, nil
	}
	manifest := filepath.Join(dir, kind+".tsv")
	out := filepath.Join(dir, kind+".jsonl")
	var b strings.Builder
	b.WriteString("fixture_id\tdialect\tpolicy\tinput_sha256\tpath\n")
	for _, f := range requested {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", f.ID, f.Dialect, f.Policy, f.SHA, f.Path)
	}
	if err := os.WriteFile(manifest, []byte(b.String()), 0600); err != nil {
		return nil, err
	}
	runctx, cancel := context.WithTimeout(ctx, 600*time.Second)
	defer cancel()
	goBinary := os.Getenv("GRAMMAR_ORACLE_GO")
	if goBinary == "" {
		goBinary = "go"
	}
	cmd := exec.CommandContext(runctx, goBinary, "test", "./"+kind, "-run", "^TestGrammarOracleReference$", "-count=1", "-timeout", "300s", "-args", "-manifest", manifest, "-out", out)
	cmd.Dir = sh
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN="+tool)
	output, err := cmd.CombinedOutput()
	if runctx.Err() != nil {
		err = fmt.Errorf("600 s wall timeout")
	}
	data, readErr := os.ReadFile(out)
	if readErr != nil && err == nil {
		err = fmt.Errorf("adapter output absent")
	}
	refs, recErr := Reconcile(data, requested, err)
	if recErr != nil {
		return nil, fmt.Errorf("%s: %w; process output: %s", kind, recErr, strings.TrimSpace(string(output)))
	}
	return refs, nil
}
func Run(opt Options) (Summary, error) {
	if opt.Root == "" {
		return Summary{}, fmt.Errorf("root required")
	}
	root, err := filepath.Abs(opt.Root)
	if err != nil {
		return Summary{}, err
	}
	sh := opt.SHRoot
	if sh == "" {
		sh, err = roots("sh", root)
		if err != nil {
			return Summary{}, err
		}
	}
	tool := opt.Toolchain
	if tool == "" {
		tool = "go1.27.2"
	}
	rows, err := readRows(filepath.Join(root, "docs/grammar-oracle/manifest.tsv"))
	if err != nil {
		return Summary{}, err
	}
	derivedRoot := root
	if opt.DerivedRoot != "" {
		if derivedRoot, err = filepath.Abs(opt.DerivedRoot); err != nil {
			return Summary{}, err
		}
	}
	isDerived := map[string]bool{}
	derived := filepath.Join(root, "docs/grammar-oracle/derived.tsv")
	if _, err := os.Stat(derived); err == nil {
		extra, e := readRows(derived)
		if e != nil {
			return Summary{}, e
		}
		for i := range extra {
			isDerived[extra[i].ID] = true
		}
		rows = append(rows, extra...)
	}
	seen := map[string]bool{}
	manifest := map[string]Fixture{}
	for _, f := range rows {
		if seen[f.ID] {
			return Summary{}, fmt.Errorf("duplicate fixture %s", f.ID)
		}
		seen[f.ID] = true
		if !isDerived[f.ID] {
			manifest[f.ID] = f
		}
	}
	tmp, err := os.MkdirTemp("", "grammar-oracle-")
	if err != nil {
		return Summary{}, err
	}
	defer os.RemoveAll(tmp)
	var syntax, gosource []Fixture
	issues := []string{}
	for i := range rows {
		f := &rows[i]
		if f.Class == "not-applicable" {
			continue
		}
		base := root
		if isDerived[f.ID] {
			base = derivedRoot
		}
		f.Bytes, f.Path, err = input(f.Source, base, tmp)
		if err != nil {
			issues = append(issues, f.ID+": input unavailable")
			continue
		}
		if digest(f.Bytes) != f.SHA {
			issues = append(issues, f.ID+": input hash mismatch")
			f.Path = ""
			continue
		}
		if isDerived[f.ID] {
			f.Transform, err = derivedTransform(f.Reason)
			if err != nil {
				issues = append(issues, f.ID+": "+err.Error())
				continue
			}
			parentID := strings.TrimPrefix(strings.TrimPrefix(f.ID, "mutant/"), "probe/")
			parentID = parentID[:strings.LastIndex(parentID, "/")]
			parent, ok := manifest[parentID]
			if !ok {
				issues = append(issues, f.ID+": parent absent")
				continue
			}
			f.Original, _, err = input(parent.Source, root, tmp)
			if err != nil || digest(f.Original) != parent.SHA {
				issues = append(issues, f.ID+": parent unavailable or hash mismatch")
				continue
			}
			f.OriginalSource = parent.Source
		} else {
			f.Original = f.Bytes
			f.OriginalSource = f.Source
		}
		if f.Dialect == "go-unit" {
			gosource = append(gosource, *f)
		} else {
			syntax = append(syntax, *f)
		}
	}
	// Accepted mixed boundaries define the Go island extents. Each island is
	// independently sent through both unit recognizers, including embed files.
	preparsed := map[string]oracle.Result{}
	parentCount := len(rows)
	for i := 0; i < parentCount; i++ {
		f := rows[i]
		if f.Path == "" || f.Dialect == "go-unit" {
			continue
		}
		o := oracle.Recognize("mixed", f.Dialect, f.Bytes)
		preparsed[f.ID] = o
		if o.Verdict != "accept" {
			continue
		}
		ps, e := payloads(f, o.Boundary, tmp)
		if e != nil {
			issues = append(issues, e.Error())
			continue
		}
		for _, p := range ps {
			if seen[p.ID] {
				issues = append(issues, p.ID+": duplicate payload ID")
				continue
			}
			seen[p.ID] = true
			rows = append(rows, p)
			gosource = append(gosource, p)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	a, e := adapter(ctx, sh, tool, tmp, "syntax", syntax)
	if e != nil {
		issues = append(issues, e.Error())
	}
	g, e := adapter(ctx, sh, tool, tmp, "gosource", gosource)
	if e != nil {
		issues = append(issues, e.Error())
	}
	refs := map[string]Reference{}
	for k, v := range a {
		refs[k] = v
	}
	for k, v := range g {
		refs[k] = v
	}
	for k, v := range refs {
		for i := range v.Diagnostics {
			v.Diagnostics[i].Msg = portable(v.Diagnostics[i].Msg, root, sh, tmp)
			if v.Diagnostics[i].Logical != nil {
				v.Diagnostics[i].Logical.File = portable(v.Diagnostics[i].Logical.File, root, sh, tmp)
			}
		}
		refs[k] = v
	}
	records := make([]Record, 0, len(rows))
	pass := len(issues) == 0
	for _, f := range rows {
		if f.Class == "not-applicable" {
			continue
		}
		o := oracle.Result{}
		if f.Path != "" && f.Class != "not-applicable" {
			lane, entry := "mixed", f.Dialect
			if f.Dialect == "go-unit" {
				lane, entry = "unit", "sourceFile"
			}
			if cached, ok := preparsed[f.ID]; ok {
				o = cached
			} else {
				o = oracle.Recognize(lane, entry, f.Bytes)
			}
			if o.Verdict == "accept" && f.Dialect == "go-unit" {
				o.Boundary = []oracle.Site{{Kind: "go-unit", Start: 0, End: len(f.Bytes)}}
			}
		}
		r, ok := refs[f.ID]
		var rp *Reference
		if ok {
			rp = &r
		}
		rec := classify(f, o, rp)
		records = append(records, rec)
		if f.Class != "not-applicable" && (rec.Outcome == "missing" || rec.Outcome == "error" || rec.Oracle == "error" || rec.Reference == "error") {
			pass = false
		}
		if !valid(f, rec) {
			pass = false
		}
	}
	label := tool
	if tool == "local" {
		label = "local go1.27.2 (cached toolchain; host review status unverified)"
	}
	summary := Summary{Records: records, Pass: pass, Issues: issues, Toolchain: label, Tally: Count(rows, records)}
	if err := writeResults(opt, rows, summary); err != nil {
		return summary, err
	}
	return summary, nil
}
func writeResults(opt Options, rows []Fixture, s Summary) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, r := range s.Records {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	if err := os.WriteFile(opt.ComparePath, b.Bytes(), 0644); err != nil {
		return err
	}
	return writeReport(opt, rows, s)
}
func writeReport(opt Options, rows []Fixture, s Summary) error {
	type count struct{ Total, Agree, Disagree, Missing, Error int }
	by := map[string]*count{}
	fixtures := map[string]Fixture{}
	for _, f := range rows {
		fixtures[f.ID] = f
		if f.Class == "not-applicable" {
			key := f.Lane + " / " + f.Class
			c := by[key]
			if c == nil {
				c = &count{}
				by[key] = c
			}
			c.Total++
		}
	}
	for _, r := range s.Records {
		key := r.Lane + " / " + r.Class
		c := by[key]
		if c == nil {
			c = &count{}
			by[key] = c
		}
		c.Total++
		switch r.Outcome {
		case "agree":
			c.Agree++
		case "disagree":
			c.Disagree++
		case "missing":
			c.Missing++
		case "error":
			c.Error++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Grammar oracle comparison\n\nToolchain: %s. Reference: sh adapter.\n\nGate: **%s**. Fixtures: %d.\n\n", s.Toolchain, map[bool]string{true: "PASS", false: "FAIL"}[s.Pass], len(s.Records))
	b.WriteString("| Lane / class | Total | Agree | Disagree | Missing | Error |\n|---|---:|---:|---:|---:|---:|\n")
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c := by[k]
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d |\n", k, c.Total, c.Agree, c.Disagree, c.Missing, c.Error)
	}
	cat := map[string]int{}
	for _, r := range s.Records {
		cat[r.Category+" / "+r.Outcome]++
	}
	b.WriteString("\n## Category totals\n\n| Category / outcome | Count |\n|---|---:|\n")
	keys = keys[:0]
	for k := range cat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "| %s | %d |\n", k, cat[k])
	}
	b.WriteString("\n## Blocking disagreements and incomplete results\n\n")
	ledger, err := os.ReadFile(opt.FindingsPath)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for _, l := range strings.Split(string(ledger), "\n") {
		if l != "" && !strings.HasPrefix(l, "#") {
			existing[strings.Split(l, "\t")[0]] = true
		}
	}
	var additions strings.Builder
	line := func(f Fixture, r Record) {
		off, line, col := position(f, r)
		fmt.Fprintf(&b, "- %s: %s at byte %d (%d:%d); oracle %s %s boundary %v; reference %s %s boundary %v; outcome %s/%s.\n", f.ID, sourceFor(f), off, line, col, r.Oracle, diagOracle(r.DiagnosticsOracle), r.BoundaryOracle, r.Reference, diagReference(r.DiagnosticsReference), r.BoundaryReference, r.Category, r.Outcome)
	}
	blocking := 0
	for _, r := range s.Records {
		if f := fixtures[r.ID]; !valid(f, r) {
			line(f, r)
			blocking++
		}
	}
	if blocking == 0 {
		b.WriteString("None.\n")
	}
	for _, r := range s.Records {
		f := fixtures[r.ID]
		if r.Outcome != "disagree" {
			continue
		}
		off, line, col := position(f, r)
		ledgerID := "reference/" + f.ID
		if r.Outcome == "disagree" && !existing[ledgerID] {
			fmt.Fprintf(&additions, "%s\t%s\t%d\t%d:%d\topen\t%s\t%s\t%s\treference comparison: %s\n", ledgerID, sourceFor(f), off, line, col, r.Oracle, r.Reference, cleanTSV(diagOracle(r.DiagnosticsOracle)+" "+fmt.Sprint(r.BoundaryOracle)+" | "+diagReference(r.DiagnosticsReference)+" "+fmt.Sprint(r.BoundaryReference)), r.Category)
			existing[ledgerID] = true
		}
	}
	if additions.Len() > 0 {
		f, e := os.OpenFile(opt.FindingsPath, os.O_APPEND|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		_, e = f.WriteString(additions.String())
		f.Close()
		if e != nil {
			return e
		}
	}
	if err := rewriteDerivedFindings(opt.FindingsPath, fixtures, s.Records); err != nil {
		return err
	}
	t := Count(rows, s.Records)
	probes := 0
	for _, r := range s.Records {
		if Kind(fixtures[r.ID]) == "probe" {
			probes++
		}
	}
	fmt.Fprintf(&b, "\n## Probes (non-blocking)\n\nCONTRACT PASS rules 1 and 3 cover the manifest rows and the frozen mutant set; a probe is neither. Probes: %d, disagreements: %d, missing/errors are counted above. Each disagreement stays open in findings.tsv and routes to the S6-gap decision (todo a964f1a3).\n\n", probes, t.Probes)
	for _, r := range s.Records {
		if f := fixtures[r.ID]; Kind(f) == "probe" && r.Outcome == "disagree" {
			line(f, r)
		}
	}
	fmt.Fprintf(&b, "\n## Known divergences\n\nThe sh static parser assumes an array index is arithmetic (sh README \"Caveats\"); the oracle follows bash -n. Each entry pins the exact observed pair; a change on either side fails the gate. Holding: %d of %d.\n\n", t.Known, len(knownDivergences))
	for _, r := range s.Records {
		if f := fixtures[r.ID]; Kind(f) == "known-divergence" {
			k := knownDivergences[f.ID]
			fmt.Fprintf(&b, "- %s: pinned oracle %s, reference %s at byte %d; observed holds=%t.\n", f.ID, k.Oracle, k.Reference, k.Offset, k.holds(r))
		}
	}
	b.WriteString("\n## Production findings\n\n```tsv\n")
	for _, entry := range strings.Split(string(ledger), "\n") {
		if strings.HasPrefix(entry, "production/") {
			b.WriteString(entry)
			b.WriteByte('\n')
		}
	}
	b.WriteString("```\n")
	b.WriteString("\n## Semantic cases\n\n")
	for _, r := range s.Records {
		if r.Category != "semantic" {
			continue
		}
		f := fixtures[r.ID]
		off, line, col := position(f, r)
		production := "-"
		if r.Production != nil {
			production = *r.Production
		}
		fmt.Fprintf(&b, "- %s: %s at byte %d (%d:%d); oracle %s %s; reference %s %s; production %s.\n", f.ID, sourceFor(f), off, line, col, r.Oracle, diagOracle(r.DiagnosticsOracle), r.Reference, diagReference(r.DiagnosticsReference), production)
	}
	if len(s.Issues) > 0 {
		b.WriteString("\n## Accounting errors\n\n")
		for _, issue := range s.Issues {
			fmt.Fprintf(&b, "- %s\n", sanitize(issue))
		}
	}
	records := map[string]Record{}
	for _, r := range s.Records {
		records[r.ID] = r
	}
	fmt.Fprintf(&b, "\n## Planned and not applicable\n\nPlanned disagreements: %d.\n\n", t.Planned)
	for _, f := range rows {
		if f.Class == "planned" || f.Class == "not-applicable" {
			outcome := ""
			if r, ok := records[f.ID]; ok && f.Class == "planned" {
				outcome = fmt.Sprintf(" Outcome %s/%s (oracle %s, reference %s).", r.Category, r.Outcome, r.Oracle, r.Reference)
			}
			fmt.Fprintf(&b, "- %s (%s): %s%s\n", f.ID, f.Class, f.Reason, outcome)
		}
	}
	return os.WriteFile(opt.ReportPath, []byte(b.String()), 0644)
}

func rewriteDerivedFindings(path string, fixtures map[string]Fixture, records []Record) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	byID := map[string]Record{}
	for _, r := range records {
		byID[r.ID] = r
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "reference/") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 9 {
			continue
		}
		id := strings.TrimPrefix(cols[0], "reference/")
		f, ok := fixtures[id]
		r, hasRecord := byID[id]
		if !ok || !hasRecord || f.Lane != "derived" || r.Outcome != "disagree" {
			continue
		}
		off, lineNo, col := position(f, r)
		cols[1], cols[2], cols[3] = sourceFor(f), strconv.Itoa(off), fmt.Sprintf("%d:%d", lineNo, col)
		lines[i] = strings.Join(cols, "\t")
	}
	updated := []byte(strings.Join(lines, "\n"))
	if bytes.Equal(data, updated) {
		return nil
	}
	return os.WriteFile(path, updated, 0644)
}
func sourceFor(f Fixture) string {
	if f.OriginalSource != "" {
		return f.OriginalSource
	}
	return f.Source
}
func position(f Fixture, r Record) (int, int, int) {
	if len(r.DiagnosticsOracle) > 0 {
		d := r.DiagnosticsOracle[0]
		return d.Offset, d.Line, d.Col
	}
	if len(r.DiagnosticsReference) > 0 {
		d := r.DiagnosticsReference[0]
		return d.Offset, d.Line, d.Col
	}
	off := 0
	for i := 0; i < len(r.BoundaryOracle) && i < len(r.BoundaryReference); i++ {
		if r.BoundaryOracle[i] != r.BoundaryReference[i] {
			off = r.BoundaryOracle[i].Start
			break
		}
	}
	if len(r.BoundaryOracle) > len(r.BoundaryReference) {
		off = r.BoundaryOracle[len(r.BoundaryReference)].Start
	}
	if len(r.BoundaryReference) > len(r.BoundaryOracle) {
		off = r.BoundaryReference[len(r.BoundaryOracle)].Start
	}
	src := f.Original
	if src == nil {
		src = f.Bytes
	}
	line, col, _ := bytePosition(src, off)
	return off, line, col
}
func diagOracle(ds []Diagnostic) string {
	if len(ds) == 0 {
		return "[]"
	}
	return strconv.Quote(ds[0].Msg)
}
func diagReference(ds []Diagnostic) string {
	if len(ds) == 0 {
		return "[]"
	}
	return strconv.Quote(ds[0].Msg)
}
func cleanTSV(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
}
func sanitize(s string) string {
	for _, v := range []string{os.Getenv("DHNT_ROOT"), os.Getenv("SH_ROOT"), os.Getenv("BASHSHARP_ROOT"), os.Getenv("HOME")} {
		if v != "" {
			s = strings.ReplaceAll(s, v, "<env>")
		}
	}
	return s
}
func portable(s, root, sh, tmp string) string {
	for _, p := range []struct{ path, label string }{{sh, "sh:"}, {root, "tests:"}, {tmp, "<fragment>"}} {
		if p.path != "" {
			s = strings.ReplaceAll(s, p.path+"/", p.label)
		}
	}
	return sanitize(s)
}

// Evaluate rechecks the emitted evidence, including its exact fixture ID set.
func Evaluate(rows []Fixture, records []Record) error {
	expected := map[string]Fixture{}
	for _, f := range rows {
		if f.Class == "not-applicable" {
			continue
		}
		if _, ok := expected[f.ID]; ok {
			return fmt.Errorf("duplicate manifest ID %s", f.ID)
		}
		expected[f.ID] = f
	}
	seen := map[string]bool{}
	for _, r := range records {
		f, ok := expected[r.ID]
		if !ok {
			return fmt.Errorf("unknown result ID %s", r.ID)
		}
		if seen[r.ID] {
			return fmt.Errorf("duplicate result ID %s", r.ID)
		}
		seen[r.ID] = true
		if r.SHA != f.SHA {
			return fmt.Errorf("result hash mismatch %s", r.ID)
		}
		if r.Lane != f.Lane || r.Class != f.Class || r.Dialect != f.Dialect || r.Policy != f.Policy || r.Expected != f.Expected {
			return fmt.Errorf("result metadata mismatch %s", r.ID)
		}
		if f.Class != "not-applicable" && (r.Outcome == "missing" || r.Outcome == "error" || r.Oracle == "" || r.Reference == "" || r.Oracle == "error" || r.Reference == "error") {
			return fmt.Errorf("incomplete result %s", r.ID)
		}
		if !valid(f, r) {
			return fmt.Errorf("disagreement %s", r.ID)
		}
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("missing result IDs: got %d, want %d", len(seen), len(expected))
	}
	return nil
}

func Verify(root, path string) error {
	rows, err := readRows(filepath.Join(root, "docs/grammar-oracle/manifest.tsv"))
	if err != nil {
		return err
	}
	derived := filepath.Join(root, "docs/grammar-oracle/derived.tsv")
	tmp, err := os.MkdirTemp("", "grammar-oracle-verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	derivedIDs := map[string]bool{}
	if _, err := os.Stat(derived); err == nil {
		extra, e := readRows(derived)
		if e != nil {
			return e
		}
		parents, e := mutate.Parents(root)
		if e != nil {
			return e
		}
		generated, e := mutate.Generate(parents)
		if e != nil {
			return e
		}
		if b, e := os.ReadFile(derived); e != nil || !bytes.Equal(mutate.Render(generated), b) {
			return fmt.Errorf("derived.tsv is stale")
		}
		if e := mutate.Materialize(tmp, generated); e != nil {
			return e
		}
		for _, f := range extra {
			derivedIDs[f.ID] = true
		}
		rows = append(rows, extra...)
	}
	parentCount := len(rows)
	for i := 0; i < parentCount; i++ {
		f := &rows[i]
		if f.Class == "not-applicable" {
			continue
		}
		base := root
		if derivedIDs[f.ID] {
			base = tmp
		}
		f.Bytes, f.Path, err = input(f.Source, base, tmp)
		if err != nil || digest(f.Bytes) != f.SHA {
			return fmt.Errorf("%s: input unavailable or hash mismatch", f.ID)
		}
		if f.Dialect == "go-unit" {
			continue
		}
		o := oracle.Recognize("mixed", f.Dialect, f.Bytes)
		if o.Verdict != "accept" {
			continue
		}
		ps, e := payloads(*f, o.Boundary, tmp)
		if e != nil {
			return e
		}
		rows = append(rows, ps...)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var records []Record
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Buffer(make([]byte, 1<<20), 64<<20)
	for s.Scan() {
		var r Record
		if err := json.Unmarshal(s.Bytes(), &r); err != nil {
			return err
		}
		records = append(records, r)
	}
	if err := s.Err(); err != nil {
		return err
	}
	return Evaluate(rows, records)
}
