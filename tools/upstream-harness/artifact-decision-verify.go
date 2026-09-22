// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #249; Story: #716; Story-ID: 05dfa5ec639c
//
// artifact-decision-verify is a read-only observer of the S249 compiler-
// artifact decision ledger. The product decision (sh@08743660,
// docs/bashpp-compiler-artifact-contracts.md) names five upstream roots whose
// gc-owned artifact assertions Bash# does not implement; the replacement
// contracts are pinned by TestS249CompilerArtifactReplacementConformance in
// the published sh tree. This verifier only authenticates the recording: each
// classified root stays in the denominator in both requested modes with its
// leaf verdict FAIL. It never relabels a verdict, never synthesizes an
// expected diagnostic, and rejects any row that names a native execution
// mode for the tested source.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// The trust anchor: changing any of these is a reviewed commit to this file,
// not an edit to the ledger it authenticates.
const (
	pinSchema          = "bashpp-tests/s249-artifact-decision/v1"
	pinDecision        = "S249-C2"
	pinShCommit        = "08743660f6e03a1de75072ff1f111f27da6b4be4"
	pinContractDoc     = "docs/bashpp-compiler-artifact-contracts.md"
	pinConformanceTest = "TestS249CompilerArtifactReplacementConformance"
	pinModes           = "go,interpreted,compiled"
	pinLeaf            = "S249-cand1-r2"
	pinTestedExecution = "never-native"
)

// The classified roots and the replacement subtest bound to each, copied from
// the contract table at sh@08743660. A ledger root outside this set is not
// covered by the product decision; a missing root is a silent exclusion.
var classifiedRoots = map[string]string{
	"testdir:closure3.go":       "closure-capture-and-call",
	"testdir:codegen/switch.go": "switch-selection-and-fallthrough",
	"testdir:live_regabi.go":    "select-tuple-receive-dereference",
	"testdir:nilptr3.go":        "nil-pointer-panic-is-observable",
	"testdir:prove.go":          "bounds-guard-controls-indexing",
}

// Both requested corpus modes must stay in the denominator for every root.
var requestedModes = []string{"interpreted", "compiled"}

var decisionColumns = []string{"root", "mode", "leaf_status", "decision", "replacement_subtest", "ground"}

// Markers of a synthesized expected diagnostic. The ledger records a
// decision about an assertion; it must never carry the assertion's expected
// text, which only gc can truthfully produce.
var syntheticDiagnosticMarkers = []string{`missing error "`, `// ERROR`, `errorcheck:`}

type decisionConfig struct {
	decisionsPath string
	pinPath       string
}

func main() {
	cfg := decisionConfig{}
	flag.StringVar(&cfg.decisionsPath, "decisions", "docs/upstream-harness/decisions/S249-compiler-artifacts.tsv", "decision ledger TSV")
	flag.StringVar(&cfg.pinPath, "pin", "docs/upstream-harness/decisions/S249-evidence-pin.tsv", "evidence pin TSV")
	flag.Parse()
	os.Exit(verifyArtifactDecisions(cfg, os.Stdout, os.Stderr))
}

func verifyArtifactDecisions(cfg decisionConfig, stdout, stderr io.Writer) int {
	violations := 0
	fail := func(format string, args ...interface{}) {
		fmt.Fprintf(stderr, "VIOLATION: "+format+"\n", args...)
		violations++
	}

	pin, err := readKeyValueTSV(cfg.pinPath)
	if err != nil {
		fail("evidence pin: %v", err)
		return 1
	}
	for key, want := range map[string]string{
		"schema":                  pinSchema,
		"decision":                pinDecision,
		"sh_commit":               pinShCommit,
		"contract_doc":            pinContractDoc,
		"conformance_test":        pinConformanceTest,
		"conformance_modes":       pinModes,
		"leaf":                    pinLeaf,
		"tested_source_execution": pinTestedExecution,
	} {
		got, ok := pin[key]
		if !ok {
			fail("evidence pin: missing key %q", key)
			continue
		}
		if got != want {
			fail("evidence pin: %s is %q, pinned value is %q", key, got, want)
		}
	}

	rows, err := readDecisionRows(cfg.decisionsPath)
	if err != nil {
		fail("decision ledger: %v", err)
		return 1
	}

	seen := map[string]map[string]bool{}
	for i, row := range rows {
		line := i + 2 // 1-based, after the header
		root, mode, status, decision, subtest := row[0], row[1], row[2], row[3], row[4]

		wantSubtest, classified := classifiedRoots[root]
		if !classified {
			fail("line %d: root %q is not classified by the S249 product decision; the decision covers exactly the contract-doc table", line, root)
			continue
		}
		if mode != "interpreted" && mode != "compiled" {
			fail("line %d: mode %q — only the two requested corpus modes are accountable; native tested-source execution is never permitted", line, mode)
			continue
		}
		if status != "FAIL" {
			fail("line %d: leaf_status %q — the upstream artifact assertion stays FAIL; a product decision never relabels it", line, status)
		}
		if decision != pinDecision {
			fail("line %d: decision %q, pinned decision is %q", line, decision, pinDecision)
		}
		if subtest != wantSubtest {
			fail("line %d: replacement_subtest %q, contract table binds %q to %s", line, subtest, wantSubtest, root)
		}
		for _, cell := range row {
			for _, marker := range syntheticDiagnosticMarkers {
				if strings.Contains(cell, marker) {
					fail("line %d: cell carries synthesized expected-diagnostic text (%q)", line, marker)
				}
			}
		}
		if seen[root] == nil {
			seen[root] = map[string]bool{}
		}
		if seen[root][mode] {
			fail("line %d: duplicate row for %s %s", line, root, mode)
		}
		seen[root][mode] = true
	}

	roots := make([]string, 0, len(classifiedRoots))
	for root := range classifiedRoots {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	for _, root := range roots {
		for _, mode := range requestedModes {
			if !seen[root][mode] {
				fail("%s %s: absent from the ledger — every classified root stays in the denominator in both requested modes; exclusion is never silent", root, mode)
			}
		}
	}

	if violations > 0 {
		fmt.Fprintf(stdout, "artifact-decision-verify: %d violation(s)\n", violations)
		return 1
	}
	fmt.Fprintf(stdout, "artifact-decision-verify: OK — %d roots x %d modes retained as FAIL under decision %s, evidence sh@%s %s (%s), leaf %s\n",
		len(classifiedRoots), len(requestedModes), pinDecision, pinShCommit[:8], pinConformanceTest, pinContractDoc, pinLeaf)
	return 0
}

func readKeyValueTSV(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s:%d: want 2 tab-separated fields, got %d", path, lineNo, len(parts))
		}
		if lineNo == 1 {
			if parts[0] != "key" || parts[1] != "value" {
				return nil, fmt.Errorf("%s: header must be key\tvalue", path)
			}
			continue
		}
		if _, dup := out[parts[0]]; dup {
			return nil, fmt.Errorf("%s:%d: duplicate key %q", path, lineNo, parts[0])
		}
		out[parts[0]] = parts[1]
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func readDecisionRows(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows [][]string
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if lineNo == 1 {
			if strings.Join(parts, "\t") != strings.Join(decisionColumns, "\t") {
				return nil, fmt.Errorf("%s: header must be exactly %q — no column may carry synthesized diagnostics", path, strings.Join(decisionColumns, "\t"))
			}
			continue
		}
		if len(parts) != len(decisionColumns) {
			return nil, fmt.Errorf("%s:%d: want %d fields, got %d", path, lineNo, len(decisionColumns), len(parts))
		}
		rows = append(rows, parts)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s: no decision rows", path)
	}
	return rows, nil
}
