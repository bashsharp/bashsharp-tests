package driver

import (
	"bytes"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"bashpp-tests/tools/grammar-oracle/oracle"
)

var embedGo = regexp.MustCompile(`^embed[ \t]+go[ \t]+"([^"\r\n]+)"(?:[ \t]+as[ \t]+[A-Za-z_][A-Za-z_0-9]*)?[ \t]*\r?$`)

func payloads(parent Fixture, sites []oracle.Site, tmp string) ([]Fixture, error) {
	var out []Fixture
	for _, site := range sites {
		if site.Start < 0 || site.End > len(parent.Bytes) || site.Start >= site.End {
			return nil, fmt.Errorf("%s: invalid boundary span", parent.ID)
		}
		var body, original []byte
		var source, path, kind string
		var mapping [][3]int
		switch site.Kind {
		case "fence":
			b, start, ok := goFenceBody(parent.Bytes, site)
			if !ok {
				continue // a non-Go fence
			}
			body = append([]byte(nil), b...)
			original = parent.Original
			if original == nil {
				original = parent.Bytes
			}
			source = sourceFor(parent)
			mapping = composeExtract(parent, start, len(body))
			kind = "go fence body"
		case "embed-line":
			m := embedGo.FindSubmatch(stripCR(bytes.TrimSuffix(parent.Bytes[site.Start:site.End], []byte{'\n'})))
			if m == nil {
				continue
			}
			repo, rest, ok := strings.Cut(parent.Source, ":")
			if !ok || strings.Contains(rest, "#") {
				return nil, fmt.Errorf("%s: cannot resolve embed source", parent.ID)
			}
			name := string(m[1])
			if filepath.IsAbs(name) {
				return nil, fmt.Errorf("%s: absolute embed path", parent.ID)
			}
			path = filepath.Clean(filepath.Join(filepath.Dir(parent.Path), name))
			source = repo + ":" + filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(rest), name)))
			var err error
			body, err = os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("%s: read Go embed target: %w", parent.ID, err)
			}
			original = body
			mapping = [][3]int{{0, 0, len(body)}}
			kind = "embed go target"
		default:
			continue
		}
		prefix := []byte(nil)
		if !hasPackageClause(body) {
			prefix = []byte("package payload\n")
		}
		input := append(append([]byte(nil), prefix...), body...)
		if path == "" || len(prefix) != 0 {
			f, err := os.CreateTemp(tmp, "go-payload-*.go")
			if err != nil {
				return nil, err
			}
			if _, err = f.Write(input); err != nil {
				f.Close()
				return nil, err
			}
			if err = f.Close(); err != nil {
				return nil, err
			}
			path = f.Name()
		}
		class, expected := parent.Class, parent.Expected
		if class == "required-negative" {
			// A rejected parent cannot supply accepted boundary spans.
			class, expected = "contextual", "-"
		}
		if class == "contextual" {
			expected = "-"
		}
		tr := &Transform{Name: kind, Map: mapPrefix(mapping, len(prefix))}
		out = append(out, Fixture{
			ID: fmt.Sprintf("%s#payload-%d", parent.ID, len(out)+1), Lane: "island-payload",
			Source: source, SHA: digest(input), Dialect: "go-unit", Policy: "gcsyntax",
			Class: class, Expected: expected, Boundary: "go-unit", Reason: "Go payload of " + parent.ID,
			Bytes: input, Original: original, OriginalSource: source, Transform: tr, Path: path,
		})
	}
	return out, nil
}

func hasPackageClause(body []byte) bool {
	fset := token.NewFileSet()
	var s scanner.Scanner
	s.Init(fset.AddFile("", fset.Base(), len(body)), body, nil, 0)
	_, first, _ := s.Scan()
	return first == token.PACKAGE
}

func goFenceBody(src []byte, site oracle.Site) ([]byte, int, bool) {
	b := src[site.Start:site.End]
	headerEnd := bytes.IndexByte(b, '\n')
	if headerEnd < 0 {
		return nil, 0, false
	}
	header := stripCR(b[:headerEnd])
	width := 0
	for width < len(header) && header[width] == '~' {
		width++
	}
	if width < 3 || !bytes.HasPrefix(header[width:], []byte("go")) {
		return nil, 0, false
	}
	if tail := header[width+2:]; len(tail) > 0 && tail[0] != ' ' && tail[0] != '\t' {
		return nil, 0, false
	}
	start := headerEnd + 1
	for p := start; p < len(b); {
		end := bytes.IndexByte(b[p:], '\n')
		if end < 0 {
			end = len(b)
		} else {
			end += p
		}
		line := stripCR(b[p:end])
		if len(line) == width && len(bytes.Trim(line, "~")) == 0 {
			return b[start:p], site.Start + start, true
		}
		p = end + 1
	}
	return nil, 0, false
}
