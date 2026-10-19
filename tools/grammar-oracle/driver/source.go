package driver

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func roots(repo, tests string) (string, error) {
	if repo == "tests" {
		return tests, nil
	}
	if repo != "sh" && repo != "bashsharp" {
		return "", fmt.Errorf("unknown source repository %q", repo)
	}
	if v := os.Getenv(map[string]string{"sh": "SH_ROOT", "bashsharp": "BASHSHARP_ROOT"}[repo]); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("DHNT_ROOT"); v != "" {
		return filepath.Abs(filepath.Join(v, repo))
	}
	return "", fmt.Errorf("set %s or DHNT_ROOT", map[string]string{"sh": "SH_ROOT", "bashsharp": "BASHSHARP_ROOT"}[repo])
}

func input(source, tests, temp string) ([]byte, string, error) {
	repo, rest, ok := strings.Cut(source, ":")
	if !ok || strings.HasPrefix(rest, "/") {
		return nil, "", fmt.Errorf("invalid source %q", source)
	}
	root, err := roots(repo, tests)
	if err != nil {
		return nil, "", err
	}
	name, sel, extracted := strings.Cut(rest, "#")
	path := filepath.Join(root, name)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if !extracted {
		return b, path, nil
	}
	switch {
	case name == "tools/polyglot-gate.sh":
		b, err = heredoc(string(b), sel)
	case name == "grammar/grammar_test.go":
		b, err = divergence(string(b), sel)
	case name == "tools/startsites/shapes.tsv":
		b, err = shape(string(b), sel)
	default:
		err = fmt.Errorf("no extractor for %s", source)
	}
	if err != nil {
		return nil, "", err
	}
	f, err := os.CreateTemp(temp, "fragment-*.src")
	if err != nil {
		return nil, "", err
	}
	if _, err = f.Write(b); err != nil {
		f.Close()
		return nil, "", err
	}
	if err = f.Close(); err != nil {
		return nil, "", err
	}
	return b, f.Name(), nil
}

var heredocHead = regexp.MustCompile(`^cat\s+>.*\.bpp["']?\s+<<'([A-Za-z_][A-Za-z_0-9]*)'\s*$`)

func heredoc(script, sel string) ([]byte, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(sel, "L"))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(script, "\n")
	if n < 1 || n > len(lines) {
		return nil, fmt.Errorf("heredoc line out of range")
	}
	m := heredocHead.FindStringSubmatch(lines[n-1])
	if m == nil {
		return nil, fmt.Errorf("not a heredoc")
	}
	var b bytes.Buffer
	for _, l := range lines[n:] {
		if l == m[1] {
			return b.Bytes(), nil
		}
		b.WriteString(l + "\n")
	}
	return nil, fmt.Errorf("unterminated heredoc")
}

var divergenceCase = regexp.MustCompile(`^\s*\{("(?:[^"\\]|\\.)*"), (true|false), (true|false)\},\s*$`)

func divergence(src, sel string) ([]byte, error) {
	i, err := strconv.Atoi(strings.TrimPrefix(sel, "TestKnownDivergences/"))
	if err != nil {
		return nil, err
	}
	_, body, ok := strings.Cut(src, "\nfunc TestKnownDivergences(")
	if !ok {
		return nil, fmt.Errorf("divergence function absent")
	}
	body, _, _ = strings.Cut(body, "\n}\n")
	var cases []string
	for _, l := range strings.Split(body, "\n") {
		if m := divergenceCase.FindStringSubmatch(l); m != nil {
			v, e := strconv.Unquote(m[1])
			if e != nil {
				return nil, e
			}
			cases = append(cases, v)
		}
	}
	if len(cases) != 4 || i < 0 || i >= len(cases) {
		return nil, fmt.Errorf("divergence index out of range")
	}
	return []byte(cases[i]), nil
}
func shape(tsv, id string) ([]byte, error) {
	for _, l := range strings.Split(tsv, "\n") {
		f := strings.Split(l, "\t")
		if strings.HasPrefix(l, "#") || len(f) < 4 || f[0] != id {
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
	return nil, fmt.Errorf("shape %q absent", id)
}
