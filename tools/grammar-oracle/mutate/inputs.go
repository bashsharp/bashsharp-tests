package mutate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Parents reads the manifest under testsRoot and resolves the input bytes of
// every row Generate uses (required-positive mixed and unit rows), checking
// each against its manifest sha256. Sibling repositories resolve as
// inventory.sh resolves them: SH_ROOT or BASHSHARP_ROOT, else DHNT_ROOT/<repo>.
func Parents(testsRoot string) ([]Parent, error) {
	data, err := os.ReadFile(filepath.Join(testsRoot, "docs/grammar-oracle/manifest.tsv"))
	if err != nil {
		return nil, err
	}
	var out []Parent
	for _, l := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Split(l, "\t")
		if len(f) != 10 {
			return nil, fmt.Errorf("manifest row %q: %d fields", f[0], len(f))
		}
		p := Parent{ID: f[0], Lane: f[1], Source: f[2], SHA: f[3], Dialect: f[4], Policy: f[5], Class: f[6]}
		if p.Class != "required-positive" || (p.Lane != "mixed" && p.Lane != "unit") {
			continue
		}
		if p.Bytes, err = resolve(testsRoot, p.Source); err != nil {
			return nil, fmt.Errorf("%s: %v", p.ID, err)
		}
		if h := sha256.Sum256(p.Bytes); hex.EncodeToString(h[:]) != p.SHA {
			return nil, fmt.Errorf("%s: input sha256 differs from the manifest", p.ID)
		}
		out = append(out, p)
	}
	return out, nil
}

func resolve(testsRoot, source string) ([]byte, error) {
	repo, rest, ok := strings.Cut(source, ":")
	if !ok {
		return nil, fmt.Errorf("source %q is not repo-qualified", source)
	}
	root := testsRoot
	if repo != "tests" {
		env := map[string]string{"sh": "SH_ROOT", "bashsharp": "BASHSHARP_ROOT"}[repo]
		if env == "" {
			return nil, fmt.Errorf("unknown repo qualifier %q", repo)
		}
		if root = os.Getenv(env); root == "" {
			if d := os.Getenv("DHNT_ROOT"); d != "" {
				root = filepath.Join(d, repo)
			} else {
				return nil, fmt.Errorf("set %s or DHNT_ROOT", env)
			}
		}
	}
	path, sel, hasSel := strings.Cut(rest, "#")
	file, err := os.ReadFile(filepath.Join(root, path))
	if err != nil || !hasSel {
		return file, err
	}
	switch path {
	case "tools/startsites/shapes.tsv":
		return shape(string(file), sel)
	case "grammar/grammar_test.go":
		return divergence(string(file), sel)
	}
	return nil, fmt.Errorf("no extractor for %s", source)
}

// shape: the probe classify.sh parses, printf '%b\n' of column 5 or else 4
// (the same extraction as manifest_test.go; the sha256 check guards it).
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

var divergenceCase = regexp.MustCompile(`^\s*\{("(?:[^"\\]|\\.)*"), (true|false), (true|false)\},\s*$`)

// divergence: the decoded Go literal of a TestKnownDivergences case.
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
	if i < 0 || i >= len(cases) {
		return nil, fmt.Errorf("TestKnownDivergences: %d cases, index %d", len(cases), i)
	}
	return []byte(cases[i]), nil
}
