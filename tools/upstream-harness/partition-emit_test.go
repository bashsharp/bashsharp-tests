// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #151; Story: #58; Story-ID: fd3a390ec1f2
// Sprint: #154; Story: S154.0; Story-ID: 4877afd3a207
// Sprint: #155; Story: S155.11; Story-ID: 5004b3c3
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestEmitPartitions(t *testing.T) {
	interpreted, compiled, out := t.TempDir(), t.TempDir(), t.TempDir()

	testdir := map[string]map[string]fixtureVerdict{
		"interpreted": {
			"kept.go":  {action: "fail", output: "Bash++ backend unsupported execute phase: module package kept.dir has non-Go inputs [a.s]"},
			"pass.go":  {action: "pass"},
			"skip.go":  {action: "skip"},
			"s151.go":  {action: "fail", output: "../../work/goroot/test/s151.go:4: gosource: unsupported LabeledStmt"},
			"s152.go":  {action: "fail", output: "LOWER-ETYPE: cannot lower expression"},
			"mixed.go": {action: "fail", output: "panic: interpreted failure"},
		},
		"compiled": {
			"kept.go":  {action: "pass"},
			"pass.go":  {action: "pass"},
			"skip.go":  {action: "skip"},
			"s151.go":  {action: "fail", output: "gosource: unsupported RangeStmt"},
			"s152.go":  {action: "fail", output: "# bashpp_fixture\nLOWER-ETYPE: compiled failure"},
			"mixed.go": {action: "fail", output: "BASHPP-EEXPR: compiled failure"},
		},
	}
	types := map[string]map[string]fixtureVerdict{
		"interpreted": {
			"TestCheck/runtime.go":          {action: "fail", output: "panic: runtime error: boom"},
			"TestCheck/escape.go":           {action: "fail", output: "escape.go:8: missing error \"x does not escape\""},
			"TestCheck/no-backend.go":       {action: "fail"},
			"TestUnit/skipped":              {action: "skip"},
			"TestObjectString/with-backend": {action: "pass"},
		},
		"compiled": {
			"TestCheck/runtime.go":          {action: "fail", output: "runtime error: boom"},
			"TestCheck/escape.go":           {action: "fail", output: "escape.go:8: wrong error"},
			"TestCheck/no-backend.go":       {action: "fail"},
			"TestUnit/skipped":              {action: "skip"},
			"TestObjectString/with-backend": {action: "pass"},
		},
	}
	packages := map[string]map[string]fixtureVerdict{
		"interpreted": {"example/unclassified": {action: "fail", output: "mystery product limitation"}},
		"compiled":    {"example/unclassified": {action: "pass"}},
	}

	for _, lane := range []struct{ mode, dir string }{{"interpreted", interpreted}, {"compiled", compiled}} {
		writeTestdirFixture(t, filepath.Join(lane.dir, "testdir.go-test.json"), testdir[lane.mode])
		writeTypesFixture(t, filepath.Join(lane.dir, "types.go-test.json"), types[lane.mode])
		// The two upstream checker implementations share conceptual root IDs;
		// an empty, valid stream proves it is still mandatory input.
		writeRecords(t, filepath.Join(lane.dir, "types2.go-test.json"))
		writePackageFixture(t, filepath.Join(lane.dir, "package.go-test.json"), packages[lane.mode])
		writeRecords(t, filepath.Join(lane.dir, "types.events.jsonl"),
			partitionEventRecord{Kind: "types-backend", Test: "TestCheck/runtime.go"},
			partitionEventRecord{Kind: "types-backend", Test: "TestCheck/escape.go"},
			partitionEventRecord{Kind: "types-backend", Test: "TestObjectString/with-backend"})
		writeRecords(t, filepath.Join(lane.dir, "backend.events.jsonl"), partitionEventRecord{Kind: "terminal", Mode: lane.mode})
	}

	var stdout bytes.Buffer
	hasFailures, err := emitPartitions(interpreted, compiled, out, &stdout, "")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFailures {
		t.Fatal("emitPartitions reported no failures")
	}

	noVerdict := "missing=0;wording=0;extra=0;class=-"
	wantFiles := map[string]string{
		"active-151-manifest.tsv": "root\tmode\tfirst_line\tverdict\n" +
			"testdir:s151.go\tinterpreted\ttest/s151.go:4: gosource: unsupported LabeledStmt\t" + noVerdict + "\n" +
			"testdir:s151.go\tcompiled\tgosource: unsupported RangeStmt\t" + noVerdict + "\n",
		"active-152-manifest.tsv": "root\tmode\tfirst_line\tverdict\n" +
			"testdir:mixed.go\tinterpreted\tpanic: interpreted failure\t" + noVerdict + "\n" +
			"testdir:mixed.go\tcompiled\tBASHPP-EEXPR: compiled failure\t" + noVerdict + "\n" +
			"testdir:s152.go\tinterpreted\tLOWER-ETYPE: cannot lower expression\t" + noVerdict + "\n" +
			"testdir:s152.go\tcompiled\tLOWER-ETYPE: compiled failure\t" + noVerdict + "\n",
		"active-153-manifest.tsv": "root\tmode\tfirst_line\tverdict\n" +
			"typechecker:go/types/TestCheck/runtime.go\tinterpreted\tpanic: runtime error: boom\t" + noVerdict + "\n" +
			"typechecker:go/types/TestCheck/runtime.go\tcompiled\truntime error: boom\t" + noVerdict + "\n",
		"active-154-manifest.tsv": "root\tmode\tfirst_line\tverdict\n" +
			"typechecker:go/types/TestCheck/escape.go\tinterpreted\tescape.go:8: missing error \"x does not escape\"\tmissing=1;wording=0;extra=0;class=missing\n" +
			"typechecker:go/types/TestCheck/escape.go\tcompiled\tescape.go:8: wrong error\t" + noVerdict + "\n",
		"active-package-manifest.tsv": "root\tmode\tfirst_line\tverdict\n" +
			"package:example/unclassified\tinterpreted\tmystery product limitation\t" + noVerdict + "\n",
		"active-unclassified.tsv": "root\tmode\tfirst_line\tverdict\n",
		"active-retained-manifest.tsv": "root\tmode\tfirst_line\tverdict\n" +
			"testdir:kept.go\tinterpreted\tBash++ backend unsupported execute phase: module package kept.dir has non-Go inputs [a.s]\t" + noVerdict + "\n",
		"native-only-typechecker.tsv": "root\tclass\tcredit\n" +
			"typechecker:go/types/TestCheck/no-backend.go\tnative-only\t0\n",
		"active-summary.tsv": "runner\tPASS\tFAIL\tSKIP\ttotal\tnative-only\n" +
			"testdir\t1\t4\t1\t6\t0\n" +
			"typechecker\t1\t2\t1\t4\t1\n" +
			"package\t0\t1\t0\t1\t0\n" +
			"total\t2\t7\t2\t11\t1\n\n" +
			"product\troots\tnative-applicable\tSKIP\tnative-only\n" +
			"total\t11\t9\t2\t1\n\n" +
			"owner\tcount\n151\t1\n152\t2\n153\t1\n154\t1\npackage\t1\nunclassified\t0\nretained\t1\ntotal\t7\n",
	}
	for name, want := range wantFiles {
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s:\n%s\nwant:\n%s", name, got, want)
		}
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 8 {
		t.Fatalf("stdout has %d report lines, want 8:\n%s", len(lines), stdout.String())
	}
	// Family names deliberately point the wrong way: TestCheck has no seam
	// record and is native-only, while TestObjectString has seam records and
	// receives product credit. Classification therefore cannot be name-based.
	if lines[0] != "native_only\ttypechecker:go/types/TestCheck/no-backend.go\tcredit=0" {
		t.Fatalf("native-only report = %q", lines[0])
	}
}

func TestPartitionRuleChanges(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		mode      string
		runner    string
		wantLine  string
		wantOwner string
	}{
		{
			name:      "build package header",
			lines:     []string{"# bashpp_s1572", `escape.go:8: missing error "x does not escape"`},
			mode:      "compiled",
			wantLine:  `escape.go:8: missing error "x does not escape"`,
			wantOwner: "154",
		},
		{
			name:      "interpreted bare asmcheck target",
			lines:     []string{"linux/amd64/v1"},
			mode:      "interpreted",
			wantLine:  "linux/amd64/v1",
			wantOwner: "retained",
		},
		{
			name:      "compiled missing opcode",
			lines:     []string{"codegen/x.go:15: linux/amd64/v1: opcode not found: `^ADDQ`"},
			mode:      "compiled",
			wantLine:  "codegen/x.go:15: linux/amd64/v1: opcode not found: `^ADDQ`",
			wantOwner: "152",
		},
		{
			name:      "non-Go inputs",
			lines:     []string{"Bash++ gotest backend: non-Go inputs [a.s]"},
			mode:      "compiled",
			wantLine:  "Bash++ gotest backend: non-Go inputs [a.s]",
			wantOwner: "152",
		},
		{
			name:      "emitter line directive with column 0",
			lines:     []string{"# bashpp_s1572", "./fixedbugs/issue18149.go:39:36: invalid column number: 0"},
			mode:      "compiled",
			wantLine:  "./fixedbugs/issue18149.go:39:36: invalid column number: 0",
			wantOwner: "152",
		},
		{
			name:      "user line directive not passed through",
			lines:     []string{"src/reflect/value.go:369; want /foo/bar.go:123 (or suffix /foo/bar.go)"},
			mode:      "interpreted",
			wantLine:  "src/reflect/value.go:369; want /foo/bar.go:123 (or suffix /foo/bar.go)",
			wantOwner: "152",
		},
		{
			name:      "asm listing logged before the verdict",
			lines:     []string{"    testdir_test.go:1857: main.Append1<1> STEXT size=117 align=0x0 args=0x8 locals=0x50 funcid=0x0", "    testdir_test.go:1857: \t0x0000 00000 (codegen/append.go:12)\tTEXT\tmain.Append1(SB), ABIInternal, $80-8", "        codegen/append.go:18: linux/amd64/v1: opcode not found: `^.*moveSliceNoCapNoScan\\b`"},
			mode:      "compiled",
			wantLine:  "codegen/append.go:18: linux/amd64/v1: opcode not found: `^.*moveSliceNoCapNoScan\\b`",
			wantOwner: "152",
		},
		{
			name:      "bodyless assembly declaration",
			lines:     []string{"LOWER-EUNSUPPORTED: function declaration without body"},
			mode:      "compiled",
			wantLine:  "LOWER-EUNSUPPORTED: function declaration without body",
			wantOwner: "152",
		},
		{
			name:      "compiled source phase non-Go input",
			lines:     []string{"Bash++ backend unsupported generate phase: compile input a.s is not a Go source file"},
			mode:      "compiled",
			wantLine:  "Bash++ backend unsupported generate phase: compile input a.s is not a Go source file",
			wantOwner: "152",
		},
		{
			name:      "cgo requires interpreted",
			lines:     []string{"package requires cgo, which this pure-Go shell does not provide"},
			mode:      "interpreted",
			wantLine:  "package requires cgo, which this pure-Go shell does not provide",
			wantOwner: "retained",
		},
		{
			name:      "cgo requires compiled",
			lines:     []string{"package requires cgo, which this pure-Go shell does not provide"},
			mode:      "compiled",
			wantLine:  "package requires cgo, which this pure-Go shell does not provide",
			wantOwner: "152",
		},
		{
			name:      "unknown import C interpreted",
			lines:     []string{`unknown import path "C"`},
			mode:      "interpreted",
			wantLine:  `unknown import path "C"`,
			wantOwner: "retained",
		},
		{
			name:      "unknown import C json-escaped (typechecker go-list stderr)",
			lines:     []string{`bin/go (GOROOT=/x, found via GOROOT, meets go1.27.0): exit status 1\nunknown import path \"C\": internal error: module loader did not resolve import\n)"`},
			mode:      "compiled",
			wantLine:  `bin/go (GOROOT=/x, found via GOROOT, meets go1.27.0): exit status 1\nunknown import path \"C\": internal error: module loader did not resolve import\n)"`,
			wantOwner: "152",
		},
		{
			name:      "could not import C keeps the diagnostic past a quoted goroot path",
			lines:     []string{"testdir_test.go:153: exit status 2", "/srv/x/goroot/test/fixedbugs/issue34968.go:12:8: could not import C (go list failed using go SDK go1.27.0 at /srv/x/goroot/bin/go (GOROOT=/srv/x/goroot, found via GOROOT, meets go1.27.0): exit status 1"},
			mode:      "compiled",
			wantLine:  "test/fixedbugs/issue34968.go:12:8: could not import C (go list failed using go SDK go1.27.0 at /srv/x/goroot/bin/go (GOROOT=/srv/x/goroot, found via GOROOT, meets go1.27.0): exit status 1",
			wantOwner: "152",
		},
		{
			name:      "bridge writeback refusal is a 153 runtime row",
			lines:     []string{"gosource: invalid native slice writeback: gosource: native callback field belongs to another dependency session"},
			mode:      "interpreted",
			wantLine:  "gosource: invalid native slice writeback: gosource: native callback field belongs to another dependency session",
			wantOwner: "153",
		},
		{
			name:      "bridge mutation refusal is a 153 runtime row",
			lines:     []string{"typeparam/double.go:46:6: gosource: dependency mutation of interpreter-owned references is unsupported for reflect.DeepEqual"},
			mode:      "interpreted",
			wantLine:  "typeparam/double.go:46:6: gosource: dependency mutation of interpreter-owned references is unsupported for reflect.DeepEqual",
			wantOwner: "153",
		},
		{
			name:      "interpreter recursion exhausting the Go stack is a 153 runtime row",
			lines:     []string{"runtime: goroutine stack exceeds 1000000000-byte limit"},
			mode:      "interpreted",
			wantLine:  "runtime: goroutine stack exceeds 1000000000-byte limit",
			wantOwner: "153",
		},
		{
			name:      "unknown field is a checker verdict (151)",
			lines:     []string{"unknown field _ of main.T"},
			mode:      "interpreted",
			wantLine:  "unknown field _ of main.T",
			wantOwner: "151",
		},
		{
			name:      "unknown import C compiled",
			lines:     []string{`unknown import path "C"`},
			mode:      "compiled",
			wantLine:  `unknown import path "C"`,
			wantOwner: "152",
		},
		{
			name:      "package root uses package owner",
			lines:     []string{"src/cmd/compile/main.go:8:2: could not import internal package"},
			mode:      "compiled",
			runner:    "package",
			wantLine:  "src/cmd/compile/main.go:8:2: could not import internal package",
			wantOwner: "package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, diagnostic := firstLine(tt.lines)
			if line != tt.wantLine {
				t.Fatalf("firstLine = %q, want %q", line, tt.wantLine)
			}
			runner := tt.runner
			if runner == "" {
				runner = "testdir"
			}
			owner := classify(line, tt.mode, diagnostic, runner, recipeEvidence{}, errorCheckVerdictOf(tt.lines).class, false)
			if owner != tt.wantOwner {
				t.Fatalf("classify(%q, %q) = %q, want %q", line, tt.mode, owner, tt.wantOwner)
			}
		})
	}
}

// TestCgoRootWithCrossModeRuntimeFailure verifies that a root with a cgo
// diagnostic (retained) in one mode and a real runtime failure in the other
// mode is assigned to the runtime owner, not retained. ownerRank ensures
// retained never absorbs a real failure in the other mode.
func TestCgoRootWithCrossModeRuntimeFailure(t *testing.T) {
	interpreted, compiled, out := t.TempDir(), t.TempDir(), t.TempDir()

	testdir := map[string]map[string]fixtureVerdict{
		"interpreted": {
			"cgo_cross.go": {action: "fail", output: `unknown import path "C"`},
		},
		"compiled": {
			"cgo_cross.go": {action: "fail", output: "panic: runtime error: nil pointer dereference"},
		},
	}
	for _, lane := range []struct{ mode, dir string }{{"interpreted", interpreted}, {"compiled", compiled}} {
		writeTestdirFixture(t, filepath.Join(lane.dir, "testdir.go-test.json"), testdir[lane.mode])
		writeRecords(t, filepath.Join(lane.dir, "types.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "types2.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "package.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "backend.events.jsonl"), partitionEventRecord{Kind: "terminal", Mode: lane.mode})
	}

	var stdout bytes.Buffer
	hasFailures, err := emitPartitions(interpreted, compiled, out, &stdout, "")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFailures {
		t.Fatal("emitPartitions reported no failures")
	}

	// The root must land in 153 (runtime), not retained.
	got, err := os.ReadFile(filepath.Join(out, "active-153-manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "testdir:cgo_cross.go") {
		t.Fatalf("cgo_cross.go not in 153 manifest:\n%s", got)
	}
	retained, err := os.ReadFile(filepath.Join(out, "active-retained-manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(retained), "testdir:cgo_cross.go") {
		t.Fatalf("cgo_cross.go should not be in retained manifest:\n%s", retained)
	}
}

// TestClassifyRecipeAndVerdictRules covers the S154.0 rules that key on the
// recipe flags carried by the backend events and on the verdict shape, never
// on expected strings: D1 (optimizer diagnostics), the run-family runtime
// rows, D2 (typechecker multiplicity and missing test builtins), and the
// verdict-carrying unclassified rows.
func TestClassifyRecipeAndVerdictRules(t *testing.T) {
	errorcheck := func(action string, flags ...string) recipeEvidence {
		return recipeEvidence{action: action, flags: flags}
	}
	tests := []struct {
		name         string
		line         string
		mode         string
		runner       string
		recipe       recipeEvidence
		verdictClass string
		want         string
	}{
		// D1: interpreted errorcheck-family rows with optimizer flags.
		{
			name:         "D1 interpreted errorcheck -m",
			line:         `esc.go:10: missing error "escapes to heap"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheck", "-0", "-m", "-l"),
			verdictClass: "missing",
			want:         "retained",
		},
		{
			name:         "D1 interpreted errorcheckwithauto -m=2 form",
			line:         `inline.go:20: missing error "can inline f"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckwithauto", "-0", "-m=2"),
			verdictClass: "missing",
			want:         "retained",
		},
		{
			name:         "D1 interpreted errorcheckandrundir nested gcflags -m",
			line:         `linkname1.go:3: missing error "xs does not escape"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckandrundir", "-gcflags", "-m -l"),
			verdictClass: "missing",
			want:         "retained",
		},
		{
			name:         "D1 interpreted errorcheckdir -live",
			line:         `live.go:15: missing error "live at entry"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckdir", "-0", "-live"),
			verdictClass: "missing",
			want:         "retained",
		},
		{
			name:         "D1 interpreted errorcheckandrundir -race -m",
			line:         `race.go:9: missing error "write barrier"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckandrundir", "-race", "-m"),
			verdictClass: "missing",
			want:         "retained",
		},
		{
			// -race alone asks for no optimizer note (issue15091, issue17449
			// pass interpreted); only -m under -race does.
			name:         "D1 interpreted errorcheck -race alone stays 154",
			line:         `race.go:9: missing error "cannot use"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheck", "-0", "-race"),
			verdictClass: "missing",
			want:         "154",
		},
		{
			name:         "D1 interpreted errorcheck -d= debug flag",
			line:         `dbg.go:3: missing error "cannot use"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheck", "-0", "-d=wb", "-d=ssa/check/on"),
			verdictClass: "missing",
			want:         "retained",
		},
		{
			name:         "D1 compiled -d= is retained (dropped as evidence only)",
			line:         `dbg.go:3: missing error "cannot use"`,
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("errorcheck", "-0", "-d=nil", "-d=ssa/check/on"),
			verdictClass: "missing",
			want:         "152",
		},
		{
			name:         "D1 compiled -m is a lowering row (152)",
			line:         `esc.go:10: missing error "escapes to heap"`,
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("errorcheck", "-0", "-m", "-l"),
			verdictClass: "missing",
			want:         "152",
		},
		{
			name:         "D1 compiled -live is a lowering row (152)",
			line:         `live.go:15: missing error "live at entry"`,
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckdir", "-0", "-live"),
			verdictClass: "missing",
			want:         "152",
		},
		{
			name:         "D1 compiled -race -m is a lowering row (152)",
			line:         `race.go:9: missing error "write barrier"`,
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckandrundir", "-race", "-m"),
			verdictClass: "missing",
			want:         "152",
		},
		{
			name:         "compiled -m with no verdict (transpile failed) is left to the line rules",
			line:         `esc.go:12:1: LOWER-EUNSUPPORTED: function declaration without body`,
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("errorcheck", "-0", "-m", "-l"),
			verdictClass: "-",
			want:         "152",
		},
		{
			name:         "run root program output with a wording substring is not a diagnostic row",
			line:         `unexpected offset 4 not 6`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("run"),
			verdictClass: "-",
			want:         "153",
		},
		{
			name:         "D1 does not fire without optimizer flags",
			line:         `plain.go:4: missing error "undefined"`,
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("errorcheck", "-0", "-l"),
			verdictClass: "missing",
			want:         "154",
		},
		{
			name:         "D1 keys on the errorcheck family, not a run recipe with -race",
			line:         "panic: runtime error: invalid memory address",
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("run", "-race"),
			verdictClass: "-",
			want:         "153",
		},
		// Run-family rows with no errorCheck verdict and a runtime shape.
		{
			name:         "run output mismatch is 153, not the 'expected ' 154 substring",
			line:         "output does not match expected in run.out. Instead saw",
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("run"),
			verdictClass: "-",
			want:         "153",
		},
		{
			name:         "errorcheckoutput generator panic is 153",
			line:         "panic: runtime error: index out of range [3]",
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckoutput"),
			verdictClass: "-",
			want:         "153",
		},
		{
			name:         "run unexpected fault is 153",
			line:         "unexpected fault address 0xb01dfacedebac1e",
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("run"),
			verdictClass: "-",
			want:         "153",
		},
		{
			name:         "run argument-count runtime shape is 153",
			line:         "expected 2 arguments; got 1",
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       errorcheck("run"),
			verdictClass: "-",
			want:         "153",
		},
		{
			name:         "a run row with a real errorCheck verdict is not rerouted",
			line:         `gen.go:5: missing error "undefined"`,
			mode:         "compiled",
			runner:       "testdir",
			recipe:       errorcheck("errorcheckoutput"),
			verdictClass: "missing",
			want:         "154",
		},
		// D2: typechecker multiplicity and missing test builtins.
		{
			name:         "D2 tab-continuation of a multi-part types.Error",
			line:         `check_test.go:256: testdata/check/cycles0.go:14:2: no error expected: "\tT3 refers to T4"`,
			mode:         "interpreted",
			runner:       "typechecker",
			recipe:       recipeEvidence{},
			verdictClass: "-",
			want:         "154",
		},
		{
			name:         "D2 undefined assert builtin",
			line:         `check_test.go:256: testdata/check/builtins0.go:91:2: no error expected: "undefined: assert"`,
			mode:         "compiled",
			runner:       "typechecker",
			recipe:       recipeEvidence{},
			verdictClass: "-",
			want:         "154",
		},
		{
			name:         "D2 undefined trace builtin",
			line:         `check_test.go:256: testdata/check/builtins0.go:100:2: no error expected: "undefined: trace"`,
			mode:         "interpreted",
			runner:       "typechecker",
			recipe:       recipeEvidence{},
			verdictClass: "-",
			want:         "154",
		},
		{
			name:         "other no-error-expected rows stay 151",
			line:         `check_test.go:256: testdata/check/decls0.go:12:2: no error expected: "undeclared name: x"`,
			mode:         "compiled",
			runner:       "typechecker",
			recipe:       recipeEvidence{},
			verdictClass: "-",
			want:         "151",
		},
		{
			name:         "could not import C still falls to retained",
			line:         "test/fixedbugs/issue34968.go:12:8: could not import C (go list failed using go SDK go1.27.0)",
			mode:         "interpreted",
			runner:       "typechecker",
			recipe:       recipeEvidence{},
			verdictClass: "-",
			want:         "retained",
		},
		// Verdict-carrying unclassified rows.
		{
			name:         "unclassified with a wording verdict is 154",
			line:         "mystery product limitation",
			mode:         "interpreted",
			runner:       "testdir",
			recipe:       recipeEvidence{},
			verdictClass: "wording",
			want:         "154",
		},
		{
			name:         "unclassified with an extra verdict is 154 in compiled mode too",
			line:         "mystery product limitation",
			mode:         "compiled",
			runner:       "testdir",
			recipe:       recipeEvidence{},
			verdictClass: "extra",
			want:         "154",
		},
		{
			name:         "unclassified without a verdict stays unclassified",
			line:         "mystery product limitation",
			mode:         "compiled",
			runner:       "testdir",
			recipe:       recipeEvidence{},
			verdictClass: "-",
			want:         "unclassified",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner := classify(tt.line, tt.mode, false, tt.runner, tt.recipe, tt.verdictClass, false)
			if owner != tt.want {
				t.Fatalf("classify(%q, %q, %v, %q) = %q, want %q", tt.line, tt.mode, tt.recipe, tt.verdictClass, owner, tt.want)
			}
		})
	}
}

// TestRecipeFlagRootsAcrossModes proves D1 through the full emit path: an
// interpreted errorcheck -m row is retained, but a real compiled failure of
// the same root still wins the owner (ownerRank); a -d= root is retained in
// both modes. The recipe flags arrive only through the backend event lane.
func TestRecipeFlagRootsAcrossModes(t *testing.T) {
	interpreted, compiled, out := t.TempDir(), t.TempDir(), t.TempDir()

	testdir := map[string]map[string]fixtureVerdict{
		"interpreted": {
			"dflag.go":     {action: "fail", output: "dflag.go:3: missing error \"cannot use\""},
			"mflag.go":     {action: "fail", output: "mflag.go:10: missing error \"escapes to heap\""},
			"mflagreal.go": {action: "fail", output: "mflagreal.go:10: missing error \"can inline f\""},
		},
		"compiled": {
			"dflag.go":     {action: "fail", output: "dflag.go:3: missing error \"cannot use\""},
			"mflag.go":     {action: "pass"},
			"mflagreal.go": {action: "fail", output: "mflagreal.go:10: missing error \"can inline f\""},
		},
	}
	for _, lane := range []struct{ mode, dir string }{{"interpreted", interpreted}, {"compiled", compiled}} {
		writeTestdirFixture(t, filepath.Join(lane.dir, "testdir.go-test.json"), testdir[lane.mode])
		writeRecords(t, filepath.Join(lane.dir, "types.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "types2.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "package.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "backend.events.jsonl"),
			partitionEventRecord{Kind: "backend", Test: "dflag.go", Mode: lane.mode, Action: "errorcheck", Phase: "compile", RecipeFlags: []string{"-0", "-d=wb", "-d=ssa/check/on"}},
			partitionEventRecord{Kind: "backend", Test: "mflag.go", Mode: lane.mode, Action: "errorcheck", Phase: "compile", RecipeFlags: []string{"-0", "-m", "-l"}},
			partitionEventRecord{Kind: "backend", Test: "mflagreal.go", Mode: lane.mode, Action: "errorcheck", Phase: "compile", RecipeFlags: []string{"-0", "-m"}},
			partitionEventRecord{Kind: "terminal", Mode: lane.mode})
	}

	var stdout bytes.Buffer
	if _, err := emitPartitions(interpreted, compiled, out, &stdout, ""); err != nil {
		t.Fatal(err)
	}
	retained, err := os.ReadFile(filepath.Join(out, "active-retained-manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"testdir:mflag.go"} {
		if !strings.Contains(string(retained), root) {
			t.Errorf("%s not in retained manifest:\n%s", root, retained)
		}
	}
	if strings.Contains(string(retained), "testdir:mflagreal.go") {
		t.Errorf("mixed root mflagreal.go must not be retained:\n%s", retained)
	}
	// The real compiled -m failure is the generated module's optimizer notes
	// (a lowering row): the mixed root follows it to 152, never to retained
	// and never to 154.
	lowering, err := os.ReadFile(filepath.Join(out, "active-152-manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lowering), "testdir:mflagreal.go") {
		t.Errorf("mixed root mflagreal.go must stay with the real compiled failure in 152:\n%s", lowering)
	}
	if !strings.Contains(string(lowering), "testdir:dflag.go") {
		t.Errorf("compiled -d= failure must be a 152 lowering row:\n%s", lowering)
	}
	fidelity, err := os.ReadFile(filepath.Join(out, "active-154-manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(fidelity), "testdir:mflag.go\t") || strings.Contains(string(fidelity), "testdir:dflag.go") || strings.Contains(string(fidelity), "testdir:mflagreal.go") {
		t.Errorf("retained or lowering roots leaked into 154:\n%s", fidelity)
	}
}

// TestErrorCheckVerdictColumn drives the verdict column with literal go-test
// Output lines in the three upstream errorCheck shapes (testdir_test.go:1226):
// `missing error "…"`, "no match for `…` in:" with tab-indented got-lines, and
// `Unmatched Errors:` with the extra diagnostics. A tab-prefixed line
// continues the previous error (testdir_test.go:1169) and must never count as
// a separate entry.
func TestErrorCheckVerdictColumn(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{
			name: "missing",
			lines: []string{
				"=== RUN   Test/miss.go",
				"    testdir_test.go:1257: miss.go:8: missing error \"undefined\"",
				"--- FAIL: Test/miss.go (0.01s)",
			},
			want: "missing=1;wording=0;extra=0;class=missing",
		},
		{
			name: "wording",
			lines: []string{
				"=== RUN   Test/word.go",
				"    testdir_test.go:1276: word.go:4: no match for `cannot use` in:",
				"        \tword.go:4:2: BASHPP-ETYPE: value mismatch",
				"--- FAIL: Test/word.go (0.01s)",
			},
			want: "missing=0;wording=1;extra=0;class=wording",
		},
		{
			name: "extra with a tab continuation that is not a second entry",
			lines: []string{
				"=== RUN   Test/extra.go",
				"    testdir_test.go:1300: ",
				"        Unmatched Errors:",
				"        extra.go:9:2: gosource: unexpected declaration",
				"        \textra.go:9:2: continued detail of the same error",
				"--- FAIL: Test/extra.go (0.01s)",
			},
			want: "missing=0;wording=0;extra=1;class=extra",
		},
		{
			name: "position: the missing diagnostic surfaced elsewhere in the same file",
			lines: []string{
				"=== RUN   Test/pos.go",
				"    testdir_test.go:1226: ",
				"        pos.go:4: missing error \"undefined: x\"",
				"        Unmatched Errors:",
				"        pos.go:6:2: undefined: x",
				"--- FAIL: Test/pos.go (0.01s)",
			},
			want: "missing=1;wording=0;extra=1;class=position",
		},
		{
			name: "multiplicity: every extra is on a line that also matched (gc output sibling)",
			lines: []string{
				"=== RUN   Test/multi.go",
				"    testdir_test.go:1233: gc output:",
				"        /work/goroot/test/multi.go:7:2: undefined: y",
				"        /work/goroot/test/multi.go:7:9: BASHPP-EDUP: second diagnostic for y",
				"    testdir_test.go:1300: ",
				"        Unmatched Errors:",
				"        multi.go:7:9: BASHPP-EDUP: second diagnostic for y",
				"--- FAIL: Test/multi.go (0.01s)",
			},
			want: "missing=0;wording=0;extra=1;class=multiplicity",
		},
		{
			name: "no errorCheck verdict at all (runtime failure)",
			lines: []string{
				"=== RUN   Test/crash.go",
				"    testdir_test.go:153: exit status 2",
				"        panic: runtime error: index out of range",
				"--- FAIL: Test/crash.go (0.01s)",
			},
			want: "missing=0;wording=0;extra=0;class=-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorCheckVerdictOf(tt.lines).column(); got != tt.want {
				t.Fatalf("errorCheckVerdictOf = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFirstLineAfterFailAndFallback(t *testing.T) {
	line, diagnostic := firstLine([]string{"=== RUN   Test/example", "--- FAIL: Test/example", "[2 tests, 0 benchmarks] ../../x/goroot/test/example.go: surprising"})
	if line != "test/example.go: surprising" || diagnostic {
		t.Fatalf("firstLine = %q, %v", line, diagnostic)
	}
	line, diagnostic = firstLine([]string{"", "plain fallback"})
	if line != "plain fallback" || diagnostic {
		t.Fatalf("fallback = %q, %v", line, diagnostic)
	}
}

func TestMissingNamedStreamIsInputError(t *testing.T) {
	interpreted, compiled := t.TempDir(), t.TempDir()
	for _, dir := range []string{interpreted, compiled} {
		for _, stream := range evidenceStreams {
			if dir == interpreted && stream.name == "types2.go-test.json" {
				continue
			}
			writeRecords(t, filepath.Join(dir, stream.name))
		}
		writeRecords(t, filepath.Join(dir, "backend.events.jsonl"))
	}
	_, err := emitPartitions(interpreted, compiled, t.TempDir(), &bytes.Buffer{}, "")
	if err == nil || !strings.Contains(err.Error(), "types2.go-test.json") {
		t.Fatalf("error = %v, want missing stream name", err)
	}
}

type fixtureVerdict struct {
	action, output string
}

func writeTestdirFixture(t *testing.T, name string, roots map[string]fixtureVerdict) {
	t.Helper()
	var records []any
	for _, root := range sortedFixtureRoots(roots) {
		v := roots[root]
		if v.output != "" {
			records = append(records, partitionGoRecord{Action: "output", Test: "Test/" + root, Output: v.output + "\n"})
		}
		records = append(records, partitionGoRecord{Action: v.action, Test: "Test/" + root})
	}
	writeRecords(t, name, records...)
}

func writeTypesFixture(t *testing.T, name string, roots map[string]fixtureVerdict) {
	t.Helper()
	var records []any
	for _, root := range sortedFixtureRoots(roots) {
		v := roots[root]
		if v.output != "" {
			records = append(records, partitionGoRecord{Action: "output", Test: root, Package: "go/types", Output: v.output + "\n"})
		}
		records = append(records, partitionGoRecord{Action: v.action, Test: root, Package: "go/types"})
	}
	writeRecords(t, name, records...)
}

func writePackageFixture(t *testing.T, name string, roots map[string]fixtureVerdict) {
	t.Helper()
	var records []any
	for _, root := range sortedFixtureRoots(roots) {
		v := roots[root]
		if v.output != "" {
			records = append(records, partitionGoRecord{Action: "output", Package: root, Output: v.output + "\n"})
		}
		records = append(records, partitionGoRecord{Action: v.action, Package: root})
	}
	writeRecords(t, name, records...)
}

func sortedFixtureRoots(roots map[string]fixtureVerdict) []string {
	out := make([]string, 0, len(roots))
	for root := range roots {
		out = append(out, root)
	}
	sort.Strings(out)
	return out
}

func writeRecords(t *testing.T, name string, records ...any) {
	t.Helper()
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			f.Close()
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// Sprint: #154; Story: S154.0; Story-ID: 4877afd3a207
func TestDebugDiagnosticFlagIgnoresUpstreamAndPanic(t *testing.T) {
	// upstream appends -d=ssa/check/on to every errorcheck compile, and
	// -d=panic carries no expectation: neither makes a plain errorcheck root
	// an optimizer-diagnostic recipe.
	for _, flags := range [][]string{{"-d=ssa/check/on"}, {"-d=panic"}, {"-d=panic", "-d=ssa/check/on"}, {"-e", "-d=panic,ssa/check/on"}} {
		if optimizerDiagnosticFlags(flags) {
			t.Errorf("optimizerDiagnosticFlags(%q) = true, want false", flags)
		}
	}
	for _, flags := range [][]string{{"-d=wb"}, {"-d=nil", "-d=ssa/check/on"}, {"-0", "-d=ssa/prove/debug=1"}, {"-d=append,slice"}, {"-d=escapedebug=1"}, {"-0", "-m", "-d=ssa/check/on"}} {
		if !optimizerDiagnosticFlags(flags) {
			t.Errorf("optimizerDiagnosticFlags(%q) = false, want true", flags)
		}
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// Partition v10.6 (D11): one fixture per moved class, each keyed on mode,
// verdict shape or root kind — never on an expected string.

// TestV106LoweringRowRoutesRoot proves rule (a) through the full emit path:
// a root whose interpreted row is an evaluator diagnostic (151 under v10.5
// by rank) and whose compiled row is a lowering-fidelity shape belongs to
// the lower owner. The five shapes each drive one root; the interpreted
// twin of every shape is left to the rank rules (negative controls).
func TestV106LoweringRowRoutesRoot(t *testing.T) {
	interpreted, compiled, out := t.TempDir(), t.TempDir(), t.TempDir()
	testdir := map[string]map[string]fixtureVerdict{
		"interpreted": {
			"lower.go":     {action: "fail", output: "lower.go:18:10: lower.go:18:10: BASHPP-EEXPR-UNDEFINED: undefined: iota"},
			"mangled.go":   {action: "fail", output: "BASHPP-EBUILTIN-TYPE: cannot use *interp.bashPPBridgeValue value as runtime.Frame"},
			"pragma.go":    {action: "fail", output: "compilation succeeded unexpectedly"},
			"unused.go":    {action: "fail", output: `test/unused.go:14:2: "bufio" imported and not used`},
			"success.go":   {action: "fail", output: "compilation succeeded unexpectedly"},
			"namespace.go": {action: "fail", output: "gosource: package-level name T._ declared twice in the lowered file"},
			"evaluator.go": {action: "fail", output: "evaluator.go:3:1: BASHPP-EEXPR-SHIFT: shift count must be an unsigned integer"},
			"gconly.go":    {action: "fail", output: "compilation succeeded unexpectedly"},
		},
		"compiled": {
			"lower.go":     {action: "fail", output: "lower.go:18:10: LOWER-EUNDEFINED: undefined: iota"},
			"mangled.go":   {action: "fail", output: "2026/09/13 10:18:16 frame 0: got main.(*__gosource_pkg_0_WaitGroup).Add, want test/mysync.(*WaitGroup).Add"},
			"pragma.go":    {action: "fail", output: "/work/Testpragma.go123/002/main.go:6: //go:nowritebarrier only allowed in runtime"},
			"unused.go":    {action: "fail", output: `test/unused.go:14:2: "bufio" imported and not used`},
			"success.go":   {action: "fail", output: "compilation succeeded unexpectedly"},
			"namespace.go": {action: "fail", output: "gosource: package-level name T._ declared twice in the lowered file"},
			// A plain gc diagnostic on the original position with no
			// lowering shape: the rank rules keep the root with 151.
			"evaluator.go": {action: "fail", output: "test/evaluator.go:41:13: cannot use t1(0) (constant 0 of float64 type t1) as I0 value in variable declaration"},
			// Interpreted unexpected success with a passing compiled row is
			// D5's gc-only check, not a lowering row.
			"gconly.go": {action: "pass"},
		},
	}
	for _, lane := range []struct{ mode, dir string }{{"interpreted", interpreted}, {"compiled", compiled}} {
		writeTestdirFixture(t, filepath.Join(lane.dir, "testdir.go-test.json"), testdir[lane.mode])
		writeRecords(t, filepath.Join(lane.dir, "types.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "types2.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "package.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "backend.events.jsonl"), partitionEventRecord{Kind: "terminal", Mode: lane.mode})
	}
	if _, err := emitPartitions(interpreted, compiled, out, &bytes.Buffer{}, ""); err != nil {
		t.Fatal(err)
	}
	lowering := readManifestFile(t, out, "active-152-manifest.tsv")
	evaluator := readManifestFile(t, out, "active-151-manifest.tsv")
	for _, root := range []string{"lower.go", "mangled.go", "pragma.go", "unused.go", "success.go", "namespace.go"} {
		if !strings.Contains(lowering, "testdir:"+root+"\t") {
			t.Errorf("%s must be routed to 152:\n%s", root, lowering)
		}
		if strings.Contains(evaluator, "testdir:"+root+"\t") {
			t.Errorf("%s must leave 151:\n%s", root, evaluator)
		}
	}
	for _, root := range []string{"evaluator.go", "gconly.go"} {
		if !strings.Contains(evaluator, "testdir:"+root+"\t") || strings.Contains(lowering, "testdir:"+root+"\t") {
			t.Errorf("%s has no lowering row and must stay 151:\n151:\n%s\n152:\n%s", root, evaluator, lowering)
		}
	}
	rules := readManifestFile(t, out, "active-rules.tsv")
	for _, want := range []string{
		"testdir:lower.go\tcompiled\tv10.6a lower-diagnostic\t152\n",
		"testdir:mangled.go\tcompiled\tv10.6a mangled-name\t152\n",
		"testdir:pragma.go\tcompiled\tv10.6a pragma-position\t152\n",
		"testdir:unused.go\tcompiled\tv10.6a unused-import\t152\n",
		"testdir:success.go\tcompiled\tv10.6a unexpected-success\t152\n",
		"testdir:namespace.go\tcompiled\tv10.6a lowered-namespace\t152\n",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("active-rules.tsv lacks %q:\n%s", want, rules)
		}
	}
	if strings.Contains(rules, "evaluator.go") || strings.Contains(rules, "gconly.go") {
		t.Errorf("no rule may fire on the negative controls:\n%s", rules)
	}
}

func TestV106LoweringShapeReadsCompiledRowsOnly(t *testing.T) {
	tests := []struct {
		mode, line, want string
	}{
		{"compiled", "chan/x.go:17:1: LOWER-ETYPE: cannot use flag.Int(...) as int value", "lower-diagnostic"},
		{"interpreted", "chan/x.go:17:1: LOWER-ETYPE: cannot use flag.Int(...) as int value", ""},
		{"compiled", "panic: interface conversion: main.X1 is not main.__gosource_pkg_0_I1: missing method Foo", "mangled-name"},
		{"compiled", "test/a.dir/b.go:9:1: LOWER-ETYPE: invalid recursive type: __gosource_pkg_1_T refers to itself", "lower-diagnostic"},
		{"compiled", `/work/002/main.go:6: //go:cgo_ldflag // ERROR "usage: //go:cgo_ldflag" only allowed in cgo-generated code`, "pragma-position"},
		// A missing-error verdict quoting a pragma is the errorCheck verdict
		// shape, not gc's refusal of the directive.
		{"compiled", `x.go:5: missing error "//go:noinline"`, ""},
		{"compiled", `test/x.go:15:2: "fmt" imported and not used`, "unused-import"},
		{"interpreted", `test/x.go:15:2: "fmt" imported and not used`, ""},
		{"compiled", "compilation succeeded unexpectedly", "unexpected-success"},
		{"interpreted", "compilation succeeded unexpectedly", ""},
		{"compiled", "gosource: package-level name T._ declared twice in the lowered file", "lowered-namespace"},
		{"compiled", "gosource: unsupported call target", ""},
		{"compiled", "test/b.go:41:13: cannot use t1(0) (constant 0 of float64 type t1) as I0 value", ""},
	}
	for _, tt := range tests {
		if got := loweringShape(tt.mode, tt.line); got != tt.want {
			t.Errorf("loweringShape(%s, %q) = %q, want %q", tt.mode, tt.line, got, tt.want)
		}
	}
}

// TestV106InternalVisibilityRoutesToPackage proves rule (b): the checker's
// internal-visibility refusal routes the root to the package/D8 owner
// whatever the root kind, over a retained interpreted row (a -m recipe) and
// over an evaluator row alike. A different import failure stays 151.
func TestV106InternalVisibilityRoutesToPackage(t *testing.T) {
	interpreted, compiled, out := t.TempDir(), t.TempDir(), t.TempDir()
	refusal := "test/escape_x.go:12:2: could not import internal/runtime/atomic (use of internal package internal/runtime/atomic not allowed)"
	testdir := map[string]map[string]fixtureVerdict{
		"interpreted": {
			"escape_x.go": {action: "fail", output: "Bash++ backend unsupported: optimizer diagnostics are a compiler artifact; the check interface has no inlining, escape-analysis or SSA meaning"},
			"both.go":     {action: "fail", output: "test/both.dir/main.go:9:4: could not import internal/runtime/sys (use of internal package internal/runtime/sys not allowed)"},
			"other.go":    {action: "fail", output: "test/other.go:3:2: could not import internal/foo (package not found)"},
		},
		"compiled": {
			"escape_x.go": {action: "fail", output: refusal},
			"both.go":     {action: "fail", output: "test/both.dir/main.go:9:4: could not import internal/runtime/sys (use of internal package internal/runtime/sys not allowed)"},
			"other.go":    {action: "fail", output: "test/other.go:3:2: could not import internal/foo (package not found)"},
		},
	}
	for _, lane := range []struct{ mode, dir string }{{"interpreted", interpreted}, {"compiled", compiled}} {
		writeTestdirFixture(t, filepath.Join(lane.dir, "testdir.go-test.json"), testdir[lane.mode])
		writeRecords(t, filepath.Join(lane.dir, "types.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "types2.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "package.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "backend.events.jsonl"),
			partitionEventRecord{Kind: "backend", Test: "escape_x.go", Mode: lane.mode, Action: "errorcheck", Phase: "compile", RecipeFlags: []string{"-0", "-m", "-l"}},
			partitionEventRecord{Kind: "terminal", Mode: lane.mode})
	}
	if _, err := emitPartitions(interpreted, compiled, out, &bytes.Buffer{}, ""); err != nil {
		t.Fatal(err)
	}
	pkg := readManifestFile(t, out, "active-package-manifest.tsv")
	for _, root := range []string{"escape_x.go", "both.go"} {
		if !strings.Contains(pkg, "testdir:"+root+"\t") {
			t.Errorf("%s must be routed to the package owner:\n%s", root, pkg)
		}
	}
	if strings.Contains(pkg, "testdir:other.go") {
		t.Errorf("a different import failure must not reach the package owner:\n%s", pkg)
	}
	if evaluator := readManifestFile(t, out, "active-151-manifest.tsv"); !strings.Contains(evaluator, "testdir:other.go\t") {
		t.Errorf("other.go must stay 151:\n%s", evaluator)
	}
	if retained := readManifestFile(t, out, "active-retained-manifest.tsv"); strings.Contains(retained, "escape_x.go") {
		t.Errorf("the refused root must not be retained:\n%s", retained)
	}
	if rules := readManifestFile(t, out, "active-rules.tsv"); !strings.Contains(rules, "testdir:escape_x.go\tcompiled\tv10.6b internal-visibility\tpackage\n") {
		t.Errorf("active-rules.tsv lacks the (b) record:\n%s", rules)
	}
}

// TestV106MultiplicityVerdictDecidesOwner proves rule (c): a multiplicity
// verdict (only extra diagnostics, each at a matched position) is a 154 row
// by the verdict column alone, whatever the extra diagnostic's wording says;
// the same line under an extra verdict is left to the line rules.
func TestV106MultiplicityVerdictDecidesOwner(t *testing.T) {
	nul1 := []string{
		"=== RUN   Test/nul1.go",
		"    testdir_test.go:1233: gc output:",
		"        /work/Testnul1.go1/001/tmp__.go:4:20: invalid NUL character",
		"        /work/Testnul1.go1/001/tmp__.go:4:20: invalid NUL character",
		"    testdir_test.go:1300: ",
		"        Unmatched Errors:",
		"        /work/Testnul1.go1/001/tmp__.go:4:20: invalid NUL character",
		"--- FAIL: Test/nul1.go (0.01s)",
	}
	verdict := errorCheckVerdictOf(nul1)
	if verdict.class != "multiplicity" || verdict.extra != 1 {
		t.Fatalf("verdict = %+v, want one extra of class multiplicity", verdict)
	}
	line, diagnostic := firstLine(nul1)
	if got := classify(line, "interpreted", diagnostic, "testdir", recipeEvidence{action: "errorcheck"}, verdict.class, false); got != "154" {
		t.Errorf("nul1 shape = %q, want 154", got)
	}
	// The wording of the extra diagnostic would otherwise pick 151.
	wording := "test/x.go:10:5: cannot use y (variable of type int) as string value"
	if got := classify(wording, "compiled", false, "testdir", recipeEvidence{action: "errorcheck"}, "multiplicity", false); got != "154" {
		t.Errorf("multiplicity verdict with an evaluator wording = %q, want 154", got)
	}
	if got := classify(wording, "compiled", false, "testdir", recipeEvidence{action: "errorcheck"}, "extra", false); got != "151" {
		t.Errorf("extra verdict with an evaluator wording = %q, want 151 (line rules)", got)
	}
	// D1 still comes first: an interpreted optimizer-diagnostic recipe with
	// a multiplicity verdict is retained.
	if got := classify(wording, "interpreted", false, "testdir", recipeEvidence{action: "errorcheck", flags: []string{"-0", "-m"}}, "multiplicity", false); got != "retained" {
		t.Errorf("D1 must precede the multiplicity rule, got %q", got)
	}
}

// TestV106CgoRootReadsTheSource proves rule (d): a root whose own source (or
// a companion's) declares cgo is a cgo root — compiled 152, interpreted
// retained — whatever phase refused it; the same first lines on a root
// without a cgo declaration keep their runtime owner. A root the corpus
// lacks is an input error, never a guess.
func TestV106CgoRootReadsTheSource(t *testing.T) {
	corpus := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		name := filepath.Join(corpus, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// issue47185's shape: the constraint on the root, import "C" in a
	// companion package.
	write("cgo_build.go", "// runindir\n\n//go:build cgo\n\npackage ignored\n")
	write("cgo_build.dir/main.go", "package main\n\nimport \"x/bad\"\n\nfunc main() { bad.F() }\n")
	write("cgo_build.dir/bad/bad.go", "package bad\n\n// int f() { return 1; }\nimport \"C\"\n\nfunc F() { _ = C.f() }\n")
	write("cgo_import.go", "// run\n\npackage main\n\n/*\nint f() { return 1; }\n*/\nimport \"C\"\n\nfunc main() { _ = C.f() }\n")
	write("cgo_group.go", "// rundir\n\npackage ignored\n")
	write("cgo_group.dir/bad.go", "package bad\n\nimport (\n\t\"fmt\"\n\t\"C\"\n)\n\nfunc F() { fmt.Println(C.int(1)) }\n")
	// bug514's shape: a run root importing runtime/cgo, refused at run time.
	write("cgo_runtime.go", "// run\n\n//go:build cgo\n\npackage main\n\nimport \"runtime/cgo\"\n\ntype T struct{ _ cgo.Incomplete }\n\nfunc main() {}\n")
	// notinheap's shape: an errorcheck root importing runtime/cgo is
	// checked like any other package (its verdict is D5's, not cgo).
	write("check_runtime.go", "// errorcheck -+\n\n//go:build cgo\n\npackage p\n\nimport \"runtime/cgo\"\n\ntype T struct{ _ cgo.Incomplete }\n")
	// A -race root carries the cgo constraint and never touches cgo.
	write("race.go", "// run -race\n\n//go:build cgo && linux && amd64\n\npackage main\n\nfunc main() {}\n")
	write("plain.go", "// run\n\n//go:build linux || darwin\n\npackage main\n\n// import \"C\" is only mentioned in this comment\nfunc main() {}\n")
	write("plain.dir/x.go", "package x\n")

	for action, want := range map[string]bool{"run": true, "rundir": true, "runindir": true, "runoutput": true, "buildrun": true, "buildrundir": true, "errorcheckandrundir": true, "errorcheckoutput": true, "errorcheck": false, "errorcheckdir": false, "compile": false, "compiledir": false, "build": false, "builddir": false, "asmcheck": false, "": false} {
		if got := recipeExecutes(action); got != want {
			t.Errorf("recipeExecutes(%q) = %v, want %v", action, got, want)
		}
	}
	if got := recipeAction([]byte("// run -race\n\n//go:build cgo\n\npackage main\n")); got != "run" {
		t.Errorf("recipeAction = %q, want run", got)
	}
	if got := recipeAction([]byte("//go:build cgo\n\n// errorcheck -+\n\npackage p\n")); got != "errorcheck" {
		t.Errorf("recipeAction after a constraint = %q, want errorcheck", got)
	}

	c := corpusRoot(corpus)
	for root, want := range map[string]bool{
		"testdir:cgo_build.go":                true,
		"testdir:cgo_import.go":               true,
		"testdir:cgo_group.go":                true,
		"testdir:cgo_runtime.go":              true,
		"testdir:check_runtime.go":            false,
		"testdir:race.go":                     false,
		"testdir:plain.go":                    false,
		"typechecker:go/types/TestCheck/x.go": false,
		"package:cmd/compile/internal/abt":    false,
	} {
		got, err := c.cgoRoot(root)
		if err != nil {
			t.Fatalf("cgoRoot(%s): %v", root, err)
		}
		if got != want {
			t.Errorf("cgoRoot(%s) = %v, want %v", root, got, want)
		}
	}
	if _, err := c.cgoRoot("testdir:missing.go"); err == nil {
		t.Error("a root absent from the corpus must be an input error")
	}
	if got, err := corpusRoot("").cgoRoot("testdir:cgo_build.go"); err != nil || got {
		t.Errorf("without a corpus the rule never fires: got %v, %v", got, err)
	}

	interpreted, compiled, out := t.TempDir(), t.TempDir(), t.TempDir()
	refusal := "Bash++ backend unsupported execute phase: module package cgo_build.dir/bad has non-Go inputs [bad.go]"
	testdir := map[string]map[string]fixtureVerdict{
		"interpreted": {
			"cgo_build.go":  {action: "fail", output: refusal},
			"cgo_import.go": {action: "fail", output: "panic: boom"},
			"plain.go":      {action: "fail", output: "panic: boom"},
		},
		"compiled": {
			"cgo_build.go":  {action: "fail", output: refusal},
			"cgo_import.go": {action: "fail", output: "panic: boom"},
			"plain.go":      {action: "fail", output: "panic: boom"},
		},
	}
	for _, lane := range []struct{ mode, dir string }{{"interpreted", interpreted}, {"compiled", compiled}} {
		writeTestdirFixture(t, filepath.Join(lane.dir, "testdir.go-test.json"), testdir[lane.mode])
		writeRecords(t, filepath.Join(lane.dir, "types.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "types2.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "package.go-test.json"))
		writeRecords(t, filepath.Join(lane.dir, "backend.events.jsonl"), partitionEventRecord{Kind: "terminal", Mode: lane.mode})
	}
	if _, err := emitPartitions(interpreted, compiled, out, &bytes.Buffer{}, c); err != nil {
		t.Fatal(err)
	}
	lowering := readManifestFile(t, out, "active-152-manifest.tsv")
	runtime := readManifestFile(t, out, "active-153-manifest.tsv")
	for _, root := range []string{"cgo_build.go", "cgo_import.go"} {
		if !strings.Contains(lowering, "testdir:"+root+"\tinterpreted\t") || !strings.Contains(lowering, "testdir:"+root+"\tcompiled\t") {
			t.Errorf("cgo root %s must carry both rows under 152 (compiled cgo wins over the retained interpreted row):\n%s", root, lowering)
		}
	}
	if !strings.Contains(runtime, "testdir:plain.go\t") || strings.Contains(lowering, "testdir:plain.go\t") {
		t.Errorf("plain.go declares no cgo and keeps its runtime owner:\n153:\n%s\n152:\n%s", runtime, lowering)
	}
	rules := readManifestFile(t, out, "active-rules.tsv")
	for _, want := range []string{
		"testdir:cgo_build.go\tinterpreted\tv10.6d cgo-root\tretained\n",
		"testdir:cgo_build.go\tcompiled\tv10.6d cgo-root\t152\n",
		"testdir:cgo_import.go\tcompiled\tv10.6d cgo-root\t152\n",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("active-rules.tsv lacks %q:\n%s", want, rules)
		}
	}
	if strings.Contains(rules, "plain.go") {
		t.Errorf("no rule may fire on plain.go:\n%s", rules)
	}
}

// TestReplayPartitionsMovesByRule drives the -manifests replay: recorded
// v10.5 manifests move under the root-level rules (a) and (b) only, the
// runner verdict block of the summary is copied verbatim, the owner block is
// recomputed, and the row-level rules are annotated without moving a root.
func TestReplayPartitionsMovesByRule(t *testing.T) {
	src, out := t.TempDir(), t.TempDir()
	header := "root\tmode\tfirst_line\tverdict\n"
	none := "missing=0;wording=0;extra=0;class=-"
	files := map[string]string{
		"active-151-manifest.tsv": header +
			"testdir:const8.go\tinterpreted\tconst8.go:18:10: BASHPP-EEXPR-UNDEFINED: undefined: iota\t" + none + "\n" +
			"testdir:const8.go\tcompiled\tconst8.go:18:10: LOWER-EUNDEFINED: undefined: iota\t" + none + "\n" +
			"testdir:escape_x.go\tcompiled\ttest/escape_x.go:12:2: could not import internal/runtime/atomic (use of internal package internal/runtime/atomic not allowed)\t" + none + "\n" +
			"testdir:stays.go\tinterpreted\tstays.go:3:1: BASHPP-EEXPR-SHIFT: shift count must be an unsigned integer\t" + none + "\n",
		"active-152-manifest.tsv": header +
			"testdir:intrinsic.go\tinterpreted\ttest/intrinsic.dir/main.go:9:4: could not import internal/runtime/sys (use of internal package internal/runtime/sys not allowed)\t" + none + "\n" +
			"testdir:intrinsic.go\tcompiled\ttest/intrinsic.dir/main.go:9:4: could not import internal/runtime/sys (use of internal package internal/runtime/sys not allowed)\t" + none + "\n" +
			"testdir:lowering.go\tcompiled\tlowering.go:1:1: LOWER-ETYPE: x\t" + none + "\n",
		"active-153-manifest.tsv": header +
			"testdir:embed.go\tinterpreted\tpanic: want 'p.I1', got 'main.__gosource_pkg_0_I1'\t" + none + "\n" +
			"testdir:embed.go\tcompiled\tpanic: interface conversion: main.X1 is not main.__gosource_pkg_0_I1: missing method Foo\t" + none + "\n",
		"active-154-manifest.tsv": header +
			"testdir:nul1.go\tinterpreted\t/work/001/tmp__.go:4:20: invalid NUL character\tmissing=0;wording=0;extra=1;class=multiplicity\n" +
			"testdir:nul1.go\tcompiled\t/work/001/tmp__.go:4:20: invalid NUL character\tmissing=0;wording=0;extra=1;class=multiplicity\n",
		"active-package-manifest.tsv": header +
			"package:cmd/compile/internal/base\tinterpreted\tgosource: duplicate file\t" + none + "\n",
		"active-unclassified.tsv": header +
			"testdir:blank.go\tinterpreted\tgosource: package-level name T._ declared twice in the lowered file\t" + none + "\n" +
			"testdir:blank.go\tcompiled\tgosource: package-level name T._ declared twice in the lowered file\t" + none + "\n" +
			"testdir:odd.go\tinterpreted\tstring for float64\t" + none + "\n",
		"active-retained-manifest.tsv": header +
			"testdir:esc.go\tinterpreted\tesc.go:10: missing error \"escapes to heap\"\tmissing=1;wording=0;extra=0;class=missing\n",
		"active-summary.tsv": "runner\tPASS\tFAIL\tSKIP\ttotal\tnative-only\n" +
			"testdir\t1\t10\t0\t11\t0\ntypechecker\t0\t0\t0\t0\t0\npackage\t0\t1\t0\t1\t0\ntotal\t1\t11\t0\t12\t0\n\n" +
			"product\troots\tnative-applicable\tSKIP\tnative-only\ntotal\t12\t12\t0\t0\n\n" +
			"owner\tcount\n151\t3\n152\t2\n153\t1\n154\t1\npackage\t1\nunclassified\t2\nretained\t1\ntotal\t11\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(src, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout bytes.Buffer
	if err := replayPartitions(src, "", out, &stdout); err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"active-151-manifest.tsv":      {"testdir:stays.go\t"},
		"active-152-manifest.tsv":      {"testdir:const8.go\tinterpreted\t", "testdir:const8.go\tcompiled\t", "testdir:lowering.go\t", "testdir:embed.go\tinterpreted\t", "testdir:embed.go\tcompiled\t", "testdir:blank.go\tinterpreted\t", "testdir:blank.go\tcompiled\t"},
		"active-153-manifest.tsv":      {},
		"active-154-manifest.tsv":      {"testdir:nul1.go\tinterpreted\t", "testdir:nul1.go\tcompiled\t"},
		"active-package-manifest.tsv":  {"package:cmd/compile/internal/base\t", "testdir:escape_x.go\tcompiled\t", "testdir:intrinsic.go\tinterpreted\t", "testdir:intrinsic.go\tcompiled\t"},
		"active-unclassified.tsv":      {"testdir:odd.go\t"},
		"active-retained-manifest.tsv": {"testdir:esc.go\t"},
	}
	for name, rows := range want {
		got := readManifestFile(t, out, name)
		if n := strings.Count(got, "\n") - 1; n != len(rows) {
			t.Errorf("%s has %d rows, want %d:\n%s", name, n, len(rows), got)
		}
		for _, row := range rows {
			if !strings.Contains(got, row) {
				t.Errorf("%s lacks %q:\n%s", name, row, got)
			}
		}
	}
	summary := readManifestFile(t, out, "active-summary.tsv")
	wantSummary := strings.SplitAfter(files["active-summary.tsv"], "owner\tcount\n")[0] +
		"151\t1\n152\t4\n153\t0\n154\t1\npackage\t3\nunclassified\t1\nretained\t1\ntotal\t11\n"
	if summary != wantSummary {
		t.Errorf("summary:\n%s\nwant:\n%s", summary, wantSummary)
	}
	rules := readManifestFile(t, out, "active-rules.tsv")
	for _, want := range []string{
		"testdir:const8.go\tcompiled\tv10.6a lower-diagnostic\t152\n",
		"testdir:embed.go\tcompiled\tv10.6a mangled-name\t152\n",
		"testdir:blank.go\tcompiled\tv10.6a lowered-namespace\t152\n",
		"testdir:escape_x.go\tcompiled\tv10.6b internal-visibility\tpackage\n",
		"testdir:intrinsic.go\tinterpreted\tv10.6b internal-visibility\tpackage\n",
		"testdir:nul1.go\tinterpreted\tv10.6c multiplicity-verdict\t154\n",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("active-rules.tsv lacks %q:\n%s", want, rules)
		}
	}
	// A rule that fires on a root already at its owner is still recorded
	// (the ledger reads the reason); a root no rule touches is not.
	if !strings.Contains(rules, "testdir:lowering.go\tcompiled\tv10.6a lower-diagnostic\t152\n") || strings.Contains(rules, "stays.go") || strings.Contains(rules, "odd.go") {
		t.Errorf("rule records must name exactly the roots a rule fired on:\n%s", rules)
	}
	if !strings.HasPrefix(stdout.String(), "active_151_rootlist\t") {
		t.Errorf("stdout = %q, want the owner rootlist digests", stdout.String())
	}
}

func readManifestFile(t *testing.T, dir, name string) string {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}
