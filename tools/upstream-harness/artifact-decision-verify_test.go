// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #249; Story: #716; Story-ID: 05dfa5ec639c
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type decisionFixture struct {
	t   *testing.T
	dir string
}

func newDecisionFixture(t *testing.T) *decisionFixture {
	t.Helper()
	f := &decisionFixture{t: t, dir: t.TempDir()}
	f.writePin(validPinLines())
	f.writeDecisions(validDecisionLines())
	return f
}

func validPinLines() []string {
	return []string{
		"key\tvalue",
		"schema\tbashpp-tests/s249-artifact-decision/v1",
		"decision\tS249-C2",
		"sh_commit\t08743660f6e03a1de75072ff1f111f27da6b4be4",
		"contract_doc\tdocs/bashpp-compiler-artifact-contracts.md",
		"conformance_test\tTestS249CompilerArtifactReplacementConformance",
		"conformance_modes\tgo,interpreted,compiled",
		"leaf\tS249-cand1-r2",
		"tested_source_execution\tnever-native",
	}
}

func validDecisionLines() []string {
	lines := []string{"root\tmode\tleaf_status\tdecision\treplacement_subtest\tground"}
	for root, subtest := range classifiedRoots {
		for _, mode := range requestedModes {
			lines = append(lines, root+"\t"+mode+"\tFAIL\tS249-C2\t"+subtest+"\tcompiler artifact, not a source observable")
		}
	}
	return lines
}

func (f *decisionFixture) writePin(lines []string) {
	f.t.Helper()
	f.write("pin.tsv", lines)
}

func (f *decisionFixture) writeDecisions(lines []string) {
	f.t.Helper()
	f.write("decisions.tsv", lines)
}

func (f *decisionFixture) write(name string, lines []string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *decisionFixture) verify() (int, string, string) {
	f.t.Helper()
	var stdout, stderr bytes.Buffer
	code := verifyArtifactDecisions(decisionConfig{
		decisionsPath: filepath.Join(f.dir, "decisions.tsv"),
		pinPath:       filepath.Join(f.dir, "pin.tsv"),
	}, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func requireViolation(t *testing.T, code int, stderr, want string) {
	t.Helper()
	if code == 0 {
		t.Fatalf("verifier accepted a tampered ledger; want violation containing %q", want)
	}
	if !strings.Contains(stderr, want) {
		t.Fatalf("stderr %q does not name the violation %q", stderr, want)
	}
}

func TestValidLedgerVerifies(t *testing.T) {
	f := newDecisionFixture(t)
	code, stdout, stderr := f.verify()
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	for _, want := range []string{"5 roots x 2 modes retained as FAIL", "sh@08743660", "S249-cand1-r2"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q missing %q", stdout, want)
		}
	}
}

func TestCommittedRepoLedgerVerifies(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := verifyArtifactDecisions(decisionConfig{
		decisionsPath: "../../docs/upstream-harness/decisions/S249-compiler-artifacts.tsv",
		pinPath:       "../../docs/upstream-harness/decisions/S249-evidence-pin.tsv",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("committed ledger rejected, exit %d, stderr: %s", code, stderr.String())
	}
}

func TestPassStatusRejected(t *testing.T) {
	f := newDecisionFixture(t)
	lines := validDecisionLines()
	lines[1] = strings.Replace(lines[1], "\tFAIL\t", "\tPASS\t", 1)
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "never relabels")
}

func TestMissingModeRejected(t *testing.T) {
	f := newDecisionFixture(t)
	var lines []string
	for _, line := range validDecisionLines() {
		if strings.HasPrefix(line, "testdir:nilptr3.go\tcompiled\t") {
			continue
		}
		lines = append(lines, line)
	}
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "testdir:nilptr3.go compiled: absent from the ledger")
}

func TestMissingRootRejected(t *testing.T) {
	f := newDecisionFixture(t)
	var lines []string
	for _, line := range validDecisionLines() {
		if strings.HasPrefix(line, "testdir:prove.go\t") {
			continue
		}
		lines = append(lines, line)
	}
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "testdir:prove.go interpreted: absent from the ledger")
}

func TestNativeModeRejected(t *testing.T) {
	f := newDecisionFixture(t)
	lines := append(validDecisionLines(),
		"testdir:prove.go\tnative\tFAIL\tS249-C2\tbounds-guard-controls-indexing\tcompiler artifact")
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "native tested-source execution is never permitted")
}

func TestUnclassifiedRootRejected(t *testing.T) {
	f := newDecisionFixture(t)
	lines := append(validDecisionLines(),
		"testdir:escape2.go\tinterpreted\tFAIL\tS249-C2\tclosure-capture-and-call\tnot in the contract table")
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "not classified by the S249 product decision")
}

func TestWrongSubtestBindingRejected(t *testing.T) {
	f := newDecisionFixture(t)
	lines := validDecisionLines()
	for i, line := range lines {
		if strings.HasPrefix(line, "testdir:nilptr3.go\t") {
			lines[i] = strings.Replace(line, "nil-pointer-panic-is-observable", "closure-capture-and-call", 1)
		}
	}
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "contract table binds")
}

func TestSynthesizedDiagnosticRejected(t *testing.T) {
	f := newDecisionFixture(t)
	lines := validDecisionLines()
	lines[1] += ` expected: missing error "generated nil check"`
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "synthesized expected-diagnostic text")
}

func TestExtraColumnRejected(t *testing.T) {
	f := newDecisionFixture(t)
	lines := validDecisionLines()
	lines[0] += "\texpected_diagnostic"
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "no column may carry synthesized diagnostics")
}

func TestDuplicateRowRejected(t *testing.T) {
	f := newDecisionFixture(t)
	lines := validDecisionLines()
	lines = append(lines, lines[1])
	f.writeDecisions(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "duplicate row")
}

func TestTamperedPinCommitRejected(t *testing.T) {
	f := newDecisionFixture(t)
	var lines []string
	for _, line := range validPinLines() {
		if strings.HasPrefix(line, "sh_commit\t") {
			line = "sh_commit\tdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
		}
		lines = append(lines, line)
	}
	f.writePin(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "pinned value")
}

func TestMissingPinKeyRejected(t *testing.T) {
	f := newDecisionFixture(t)
	var lines []string
	for _, line := range validPinLines() {
		if strings.HasPrefix(line, "tested_source_execution\t") {
			continue
		}
		lines = append(lines, line)
	}
	f.writePin(lines)
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, `missing key "tested_source_execution"`)
}

func TestMissingPinFileRejected(t *testing.T) {
	f := newDecisionFixture(t)
	if err := os.Remove(filepath.Join(f.dir, "pin.tsv")); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := f.verify()
	requireViolation(t, code, stderr, "evidence pin")
}
