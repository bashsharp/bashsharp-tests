// Package grammaroracle is the Sprint 384 (#1586) grammar-oracle test tool.
// Checkpoint 1 freezes its inputs: docs/grammar-oracle/{pin,manifest}.tsv and
// CONTRACT.md. These tests re-derive every manifest input independently of
// inventory.sh and require the generator to reproduce the committed manifest.
package grammaroracle

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	manifestPath = "../../docs/grammar-oracle/manifest.tsv"
	pinPath      = "../../docs/grammar-oracle/pin.tsv"
	dhntDefault  = "/Users/qiangli/projects/poc/dhnt"
	s6SHA256     = "1cbc58250667661eeac6d5f99c94c8b5afb9d34f3b1c3026e03d30ecc16a3f71"
)

var (
	lanes    = set("mixed", "unit", "island", "derived")
	classes  = set("required-positive", "required-negative", "contextual", "planned", "not-applicable")
	dialects = map[string]map[string]bool{ // dialect -> legal reference policies
		"bash":      set("sh-syntax"),
		"bashsharp": set("sh-syntax"),
		"go-unit":   set("gcsyntax", "goparser"),
	}
	hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func set(xs ...string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

type entry struct {
	id, lane, source, sha, dialect, policy, class, expected, boundary, reason string
	line                                                                      int
}

func readManifest(t *testing.T) []entry {
	t.Helper()
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var out []entry
	for i, l := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Split(l, "\t")
		if len(f) != 10 {
			t.Fatalf("manifest line %d: %d fields, want 10", i+1, len(f))
		}
		out = append(out, entry{f[0], f[1], f[2], f[3], f[4], f[5], f[6], f[7], f[8], f[9], i + 1})
	}
	return out
}

// repoRoot resolves a repo qualifier exactly as inventory.sh does.
func repoRoot(t *testing.T, repo string) string {
	t.Helper()
	tests, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"sh": "SH_ROOT", "bashsharp": "BASHSHARP_ROOT"}
	switch repo {
	case "tests":
		return tests
	case "sh", "bashsharp":
		if v := os.Getenv(env[repo]); v != "" {
			return v
		}
		if st, err := os.Stat(filepath.Join(tests, "..", repo)); err == nil && st.IsDir() {
			return filepath.Join(tests, "..", repo)
		}
		return filepath.Join(dhntDefault, repo)
	}
	t.Fatalf("unknown repo qualifier %q", repo)
	return ""
}

// input returns the exact bytes a manifest source names; ok is false for an
// inventory key that has no committed bytes.
func input(t *testing.T, source string) (data []byte, ok bool, err error) {
	repo, rest, found := strings.Cut(source, ":")
	if !found || strings.HasPrefix(rest, "/") {
		return nil, false, fmt.Errorf("source %q is not repo-qualified and relative", source)
	}
	path, sel, hasSel := strings.Cut(rest, "#")
	file, err := os.ReadFile(filepath.Join(repoRoot(t, repo), path))
	if err != nil || !hasSel {
		return file, err == nil, err
	}
	switch {
	case path == "tools/polyglot-gate.sh":
		n, err := strconv.Atoi(strings.TrimPrefix(sel, "L"))
		if err != nil {
			return nil, false, err
		}
		b, err := heredoc(string(file), n)
		return b, true, err
	case path == "grammar/grammar_test.go":
		b, err := divergence(string(file), sel)
		return b, true, err
	case path == "tools/startsites/shapes.tsv":
		b, err := shape(string(file), sel)
		return b, true, err
	case path == "tests/tour/inventory.tsv", strings.HasSuffix(path, "/pin.tsv"):
		for _, l := range strings.Split(string(file), "\n") {
			if !strings.HasPrefix(l, "#") && strings.Split(l, "\t")[0] == sel {
				return nil, false, nil
			}
		}
		return nil, false, fmt.Errorf("%s has no row %q", path, sel)
	}
	return nil, false, fmt.Errorf("no extractor for %s", source)
}

var heredocHead = regexp.MustCompile(`^cat\s+>.*\.bpp["']?\s+<<'([A-Za-z_][A-Za-z_0-9]*)'\s*$`)

// heredoc: the lines after line n up to its quoted marker, as cat writes them.
func heredoc(script string, n int) ([]byte, error) {
	lines := strings.Split(script, "\n")
	if n < 1 || n > len(lines) {
		return nil, fmt.Errorf("line %d out of range", n)
	}
	m := heredocHead.FindStringSubmatch(lines[n-1])
	if m == nil {
		return nil, fmt.Errorf("line %d is not a .bpp heredoc", n)
	}
	var b bytes.Buffer
	for _, l := range lines[n:] {
		if l == m[1] {
			return b.Bytes(), nil
		}
		b.WriteString(l + "\n")
	}
	return nil, fmt.Errorf("heredoc at line %d is unterminated", n)
}

var divergenceCase = regexp.MustCompile(`^\s*\{("(?:[^"\\]|\\.)*"), (true|false), (true|false)\},\s*$`)

func divergence(src, sel string) ([]byte, error) {
	i, err := strconv.Atoi(strings.TrimPrefix(sel, "TestKnownDivergences/"))
	if err != nil {
		return nil, err
	}
	_, body, ok := strings.Cut(src, "\nfunc TestKnownDivergences(")
	if !ok {
		return nil, fmt.Errorf("TestKnownDivergences not found")
	}
	body, _, _ = strings.Cut(body, "\n}\n")
	var cases []string
	for _, l := range strings.Split(body, "\n") {
		if m := divergenceCase.FindStringSubmatch(l); m != nil {
			s, err := strconv.Unquote(m[1])
			if err != nil {
				return nil, err
			}
			cases = append(cases, s)
		}
	}
	if len(cases) != 4 || i < 0 || i >= len(cases) {
		return nil, fmt.Errorf("TestKnownDivergences: %d cases, index %d", len(cases), i)
	}
	return []byte(cases[i]), nil
}

// shape: the probe classify.sh parses, printf '%b\n' of column 5 or else 4.
func shape(tsv, id string) ([]byte, error) {
	for _, l := range strings.Split(tsv, "\n") {
		f := strings.Split(l, "\t")
		if strings.HasPrefix(l, "#") || f[0] != id || len(f) < 4 {
			continue
		}
		probe := f[3]
		if len(f) > 4 && f[4] != "" {
			probe = f[4]
		}
		var b strings.Builder
		for i := 0; i < len(probe); i++ {
			if probe[i] != '\\' || i+1 == len(probe) {
				b.WriteByte(probe[i])
				continue
			}
			i++
			switch probe[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(probe[i])
			}
		}
		return []byte(b.String() + "\n"), nil
	}
	return nil, fmt.Errorf("no shape %q", id)
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestManifestRows(t *testing.T) {
	rows := readManifest(t)
	if len(rows) == 0 {
		t.Fatal("empty manifest")
	}
	seen := map[string]int{}
	for _, e := range rows {
		where := fmt.Sprintf("line %d %s", e.line, e.id)
		if prev, dup := seen[e.id]; dup {
			t.Errorf("%s: duplicate fixture_id (first at line %d)", where, prev)
		}
		seen[e.id] = e.line
		if !lanes[e.lane] || !classes[e.class] {
			t.Errorf("%s: illegal lane %q or class %q", where, e.lane, e.class)
		}
		if e.reason == "" || strings.Contains(e.source, "/Users/") {
			t.Errorf("%s: missing reason or absolute source", where)
		}
		want := map[string]string{"required-positive": "accept", "required-negative": "reject", "not-applicable": "-"}
		if w, ok := want[e.class]; ok && e.expected != w {
			t.Errorf("%s: class %s needs expected %s, got %s", where, e.class, w, e.expected)
		}
		data, ok, err := input(t, e.source)
		switch {
		case err != nil:
			t.Errorf("%s: source %s: %v", where, e.source, err)
			continue
		case !ok:
			if e.sha != "-" || e.class != "not-applicable" {
				t.Errorf("%s: source has no bytes, so the row must be not-applicable with sha256 -", where)
			}
			continue
		case sum(data) != e.sha:
			t.Errorf("%s: sha256 %s, source %s has %s", where, e.sha, e.source, sum(data))
		}
		if e.class == "not-applicable" {
			if e.dialect != "-" || e.policy != "-" {
				t.Errorf("%s: not-applicable rows carry no dialect/policy", where)
			}
			continue
		}
		if !dialects[e.dialect][e.policy] || !hex64.MatchString(e.sha) {
			t.Errorf("%s: illegal dialect %q / policy %q", where, e.dialect, e.policy)
		}
		if e.expected != "accept" && e.expected != "reject" && e.expected != "-" {
			t.Errorf("%s: illegal expected %q", where, e.expected)
		}
	}
	counts := map[string]int{}
	for _, e := range rows {
		counts[e.lane+" "+e.class]++
	}
	t.Logf("%d rows: %v", len(rows), counts)
}

func TestInventoryRegenerates(t *testing.T) {
	cmd := exec.Command("sh", "inventory.sh")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	got, err := cmd.Output()
	if err != nil {
		t.Fatalf("inventory.sh: %v\n%s", err, stderr.String())
	}
	want, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("inventory.sh output differs from %s; regenerate with tools/grammar-oracle/inventory.sh > docs/grammar-oracle/manifest.tsv", manifestPath)
	}
}

// Pinned local copies: grammar files under grammar/upstream, the golang
// license headers, and the S6 table.
func TestPinnedFiles(t *testing.T) {
	f, err := os.Open(pinPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	checked := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		if strings.HasPrefix(l, "#") || l == "" {
			continue
		}
		p := strings.Split(l, "\t")
		if len(p) != 7 {
			t.Fatalf("pin row %q: %d fields, want 7", l, len(p))
		}
		id, kind, sha := p[0], p[1], p[4]
		var data []byte
		switch kind {
		case "grammar-file":
			data, err = os.ReadFile(filepath.Join("grammar/upstream", strings.TrimPrefix(id, "grammars-v4:golang/")))
		case "license-header":
			name := strings.TrimSuffix(strings.TrimPrefix(id, "grammars-v4:golang/"), "#license-header")
			data, err = os.ReadFile(filepath.Join("grammar/upstream", name))
			if end := bytes.Index(data, []byte("*/")); err == nil && end >= 0 {
				nl := bytes.IndexByte(data[end:], '\n')
				data = data[:end+nl+1]
			}
		case "s6-table":
			data, err = os.ReadFile(filepath.Join(repoRoot(t, "bashsharp"), "godelta/go-delta.tsv"))
			if sha != s6SHA256 {
				t.Errorf("S6 pin %s, frozen %s", sha, s6SHA256)
			}
		default:
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		if sum(data) != sha {
			t.Errorf("%s: pinned sha256 %s, file has %s", id, sha, sum(data))
		}
		checked++
	}
	if checked != 8 {
		t.Errorf("checked %d pinned local files, want 8 (5 grammar files, 2 license headers, S6 table)", checked)
	}
}
