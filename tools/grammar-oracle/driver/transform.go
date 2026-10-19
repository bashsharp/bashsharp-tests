package driver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"bashpp-tests/tools/grammar-oracle/oracle"
)

// Transform maps recognizer input bytes back to the original source file.
// Unmapped bytes (a package prefix or an inserted mutant token) use the
// nearest mapped byte as their anchor.
type Transform struct {
	Name string   `json:"name"`
	Map  [][3]int `json:"map"`
}

func derivedTransform(reason string) (*Transform, error) {
	_, encoded, ok := strings.Cut(reason, "transform ")
	if !ok {
		return nil, fmt.Errorf("derived row has no transform")
	}
	var tr Transform
	if err := json.Unmarshal([]byte(encoded), &tr); err != nil {
		return nil, err
	}
	if tr.Name == "" || len(tr.Map) == 0 {
		return nil, fmt.Errorf("empty derived transform")
	}
	return &tr, nil
}

func oracleDiagnostics(ds []oracle.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(ds))
	for _, d := range ds {
		out = append(out, Diagnostic{Offset: d.Offset, Line: d.Line, Col: d.Col, Char: d.Char, Msg: d.Msg})
	}
	return out
}

func mapOffset(tr *Transform, off int, end bool) (int, bool) {
	if tr == nil {
		return off, false
	}
	for i, seg := range tr.Map {
		if off >= seg[0] && off < seg[0]+seg[2] {
			if end && off == seg[0] && i > 0 {
				prev := tr.Map[i-1]
				if prev[0]+prev[2] == off {
					// A span end at a discontinuity belongs to the preceding segment.
					return prev[1] + prev[2], false
				}
			}
			return seg[1] + off - seg[0], false
		}
	}
	for i := len(tr.Map) - 1; i >= 0; i-- {
		seg := tr.Map[i]
		if off == seg[0]+seg[2] {
			return seg[1] + seg[2], false
		}
	}
	// Anchor synthetic bytes at the closest mapped edge.
	bestDistance := int(^uint(0) >> 1)
	anchor := 0
	for _, seg := range tr.Map {
		for _, edge := range [][2]int{{seg[0], seg[1]}, {seg[0] + seg[2], seg[1] + seg[2]}} {
			d := off - edge[0]
			if d < 0 {
				d = -d
			}
			if d < bestDistance {
				bestDistance, anchor = d, edge[1]
			}
		}
	}
	return anchor, true
}

func bytePosition(src []byte, off int) (line, col, char int) {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	line, col, char = 1, 1, 1
	start := 0
	for i, b := range src[:off] {
		if b == '\n' {
			line++
			start = i + 1
		}
	}
	col = off - start + 1
	for p := start; p < off; {
		_, size := utf8.DecodeRune(src[p:off])
		p += size
		char++
	}
	return
}

func mapRecord(r *Record, f Fixture) {
	if f.Transform == nil {
		return
	}
	original := f.Original
	if original == nil {
		original = f.Bytes
	}
	mapDiags := func(ds []Diagnostic) {
		for i := range ds {
			off, synthetic := mapOffset(f.Transform, ds[i].Offset, false)
			ds[i].Offset = off
			ds[i].Line, ds[i].Col, ds[i].Char = bytePosition(original, off)
			ds[i].Synthetic = synthetic
		}
	}
	mapSites := func(ss []oracle.Site) {
		for i := range ss {
			ss[i].Start, _ = mapOffset(f.Transform, ss[i].Start, false)
			ss[i].End, _ = mapOffset(f.Transform, ss[i].End, true)
		}
	}
	mapDiags(r.DiagnosticsOracle)
	mapDiags(r.DiagnosticsReference)
	mapSites(r.BoundaryOracle)
	mapSites(r.BoundaryReference)
}

// composeExtract maps a contiguous slice of a parent's recognizer input to
// the parent's original source, including any mutation or probe transform.
func composeExtract(parent Fixture, start, length int) [][3]int {
	if parent.Transform == nil {
		return [][3]int{{0, start, length}}
	}
	var out [][3]int
	for _, seg := range parent.Transform.Map {
		lo := max(start, seg[0])
		hi := min(start+length, seg[0]+seg[2])
		if lo < hi {
			out = append(out, [3]int{lo - start, seg[1] + lo - seg[0], hi - lo})
		}
	}
	return out
}

func mapPrefix(m [][3]int, prefix int) [][3]int {
	out := make([][3]int, len(m))
	for i, seg := range m {
		out[i] = [3]int{seg[0] + prefix, seg[1], seg[2]}
	}
	return out
}

func stripCR(b []byte) []byte { return bytes.TrimSuffix(b, []byte{'\r'}) }
