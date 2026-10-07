// Sprint 381 / Story #107 / Story-ID 3b50496741fd.
//
// Sealed-lane dispatch tests for the tour-evidence/v2 validator: the committed
// historical ledger authenticates against the seal-time input snapshot, and
// every tamper direction — ledger bytes, seal inventory, seal baseline — fails
// closed. The sealed lane is pure file validation, so these run anywhere.
package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func sealTestRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	if !fileExists(filepath.Join(root, "tests/tour/evidence.jsonl")) {
		t.Fatalf("repo root not found from %s", file)
	}
	return root
}

func copyFileTo(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.WriteFile(dst, readFile(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copySealDir(t *testing.T, root string) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(root, evidenceSealDir)
	for _, name := range []string{"seal.tsv", "inventory.tsv", "results.tsv", "baseline-pin.tsv", "toolchain.tsv"} {
		copyFileTo(t, filepath.Join(src, name), filepath.Join(dir, name))
	}
	return dir
}

func TestSealedHistoricalLedgerValidates(t *testing.T) {
	root := sealTestRoot(t)
	if code := cmdEvidenceValidator(root); code != 0 {
		t.Fatalf("sealed historical ledger must validate against the seal-time snapshot, got exit %d", code)
	}
}

func TestTamperedLedgerBytesLeaveSealedLane(t *testing.T) {
	root := sealTestRoot(t)
	tampered := filepath.Join(t.TempDir(), "evidence.jsonl")
	data := readFile(filepath.Join(root, "tests/tour/evidence.jsonl"))
	// Flip one byte inside a base64 payload: the ledger stays parseable but no
	// longer matches the seal, so it must take the full live lane and fail
	// closed there (this tree is not the ledger's recorded input state).
	mutated := append([]byte(nil), data...)
	for i := range mutated {
		if mutated[i] == 'A' {
			mutated[i] = 'B'
			break
		}
	}
	if err := os.WriteFile(tampered, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOUR_EVIDENCE", tampered)
	if code := cmdEvidenceValidator(root); code == 0 {
		t.Fatal("byte-tampered historical ledger was accepted")
	}
}

func TestTamperedSealInventoryFailsClosed(t *testing.T) {
	root := sealTestRoot(t)
	dir := copySealDir(t, root)
	inv := filepath.Join(dir, "inventory.tsv")
	if err := os.WriteFile(inv, append(readFile(inv), []byte("forged/path.go\tx\tx\tgo_program\tx\tx\t1\tdeadbeef\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOUR_EVIDENCE_SEAL", dir)
	if code := cmdEvidenceValidator(root); code == 0 {
		t.Fatal("tampered seal-time inventory was accepted")
	}
}

func TestJointLedgerAndSealMutationFailsClosed(t *testing.T) {
	root := sealTestRoot(t)
	dir := copySealDir(t, root)
	tampered := filepath.Join(t.TempDir(), "evidence.jsonl")
	mutated := append([]byte(nil), readFile(filepath.Join(root, "tests/tour/evidence.jsonl"))...)
	for i := range mutated {
		if mutated[i] == 'A' {
			mutated[i] = 'B'
			break
		}
	}
	if err := os.WriteFile(tampered, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	// Forge seal.tsv to pin the mutated ledger's digest: the compiled-in
	// legacy digest pin must still reject the seal record itself.
	sealRows := string(readFile(filepath.Join(dir, "seal.tsv")))
	forged := strings.ReplaceAll(sealRows, legacySealedEvidenceSHA256, sha256hex(mutated))
	if forged == sealRows {
		t.Fatal("seal fixture did not contain the legacy digest")
	}
	if err := os.WriteFile(filepath.Join(dir, "seal.tsv"), []byte(forged), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOUR_EVIDENCE", tampered)
	t.Setenv("TOUR_EVIDENCE_SEAL", dir)
	if code := cmdEvidenceValidator(root); code == 0 {
		t.Fatal("joint mutation of ledger and seal record was accepted")
	}
}

func TestTamperedSealBaselineFailsClosed(t *testing.T) {
	root := sealTestRoot(t)
	dir := copySealDir(t, root)
	results := filepath.Join(dir, "results.tsv")
	if err := os.WriteFile(results, append(readFile(results), '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOUR_EVIDENCE_SEAL", dir)
	if code := cmdEvidenceValidator(root); code == 0 {
		t.Fatal("tampered seal-time baseline results were accepted")
	}
}
