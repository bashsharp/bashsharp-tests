// The adapter is linked with the unchanged tools/go-by-example/gbe package.
// All process supervision and SDK authentication belong to that package.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	if os.Getenv("FENCE_CORPUS") != "1" {
		return
	}
	ROOT = resolveRoot()
	DOCS = ROOT + "/docs/go-by-example"
	if len(os.Args) < 2 {
		fatal("missing fence command")
	}
	var err error
	switch os.Args[1] {
	case "fence-validate":
		if len(os.Args) != 3 {
			fatal("usage: validate-inventory.sh DIR")
		}
		err = fenceInventory(os.Args[2])
		if err == nil {
			fmt.Println("PASS inventory", os.Args[2])
		}
	case "fence-check":
		err = fenceCheck(os.Args[2:])
	default:
		err = fmt.Errorf("unknown fence command")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// Accept the contract's plain header or the established # header convention.
func fenceTable(path, header string) ([][]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if len(lines) == 0 || strings.TrimPrefix(lines[0], "# ") != header {
		return nil, fmt.Errorf("%s: expected header %q", path, header)
	}
	n := len(strings.Split(header, "\t"))
	for i, line := range lines[1:] {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		row := strings.Split(line, "\t")
		if len(row) != n {
			return nil, fmt.Errorf("%s:%d: expected %d fields", path, i+2, n)
		}
		for _, v := range row {
			if v == "" {
				return nil, fmt.Errorf("%s:%d: empty field", path, i+2)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
func fenceCases(dir string) ([][]string, error) {
	rows, err := fenceTable(filepath.Join(dir, "cases.tsv"), "id\tmode\tfile\tneeds\trc\txfail")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if seen[r[0]] {
			return nil, fmt.Errorf("duplicate case id %s", r[0])
		}
		seen[r[0]] = true
		rc, e := strconv.Atoi(r[4])
		if (r[1] != "run" && r[1] != "compiled" && r[1] != "build") || e != nil || rc < 0 || rc > 255 || r[5] != "-" {
			return nil, fmt.Errorf("invalid case/rc/xfail: %s", r[0])
		}
		if filepath.IsAbs(r[2]) || strings.Contains(filepath.ToSlash(r[2]), "../") || !strings.HasSuffix(r[2], ".bsh") {
			return nil, fmt.Errorf("invalid case file: %s", r[2])
		}
		path := filepath.Join(dir, r[2])
		if !isRegularFile(path) {
			return nil, fmt.Errorf("missing case: %s", path)
		}
		b, e := os.ReadFile(strings.TrimSuffix(path, ".bsh") + ".expected")
		if e != nil {
			return nil, e
		}
		header, body, ok := bytes.Cut(b, []byte("\n"))
		if !ok || string(header) != "# rc="+r[4] {
			return nil, fmt.Errorf("invalid expected rc header: %s", path)
		}
		if r[1] == "build" && (rc != 0 || len(body) != 0) {
			return nil, fmt.Errorf("build expected must contain only # rc=0: %s", path)
		}
		if _, e := fenceComparator(path); e != nil {
			return nil, e
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s: no cases", dir)
	}
	return rows, nil
}
func fenceInventory(dir string) error {
	pins, err := fenceTable(filepath.Join(dir, "pin.tsv"), "source\tupstream_repo\tcommit\tcode_license\tprose_license")
	if err != nil {
		return err
	}
	if len(pins) == 0 {
		return fmt.Errorf("empty pin.tsv")
	}
	seenPins := map[string]bool{}
	for _, p := range pins {
		if seenPins[p[0]] || !regexp.MustCompile(`^[0-9a-fA-F]{40}([0-9a-fA-F]{24})?$`).MatchString(p[2]) {
			return fmt.Errorf("invalid or duplicate pin: %s", p[0])
		}
		seenPins[p[0]] = true
	}
	cases, err := fenceCases(dir)
	if err != nil {
		return err
	}
	ids := map[string]bool{}
	mapped := map[string]bool{}
	for _, c := range cases {
		ids[c[0]] = true
	}
	rows, err := fenceTable(filepath.Join(dir, "inventory.tsv"), "source_ref\tdisposition\tcase_id\treason")
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("empty inventory")
	}
	refs := map[string]bool{}
	allowed := regexp.MustCompile(`^(case|diagnostic|prose|fragment|interactive|env-dependent|os-only:[a-zA-Z0-9_-]+|missing-feature:[a-zA-Z0-9_-]+)$`)
	for _, r := range rows {
		if refs[r[0]] {
			return fmt.Errorf("duplicate source_ref: %s", r[0])
		}
		refs[r[0]] = true
		if !allowed.MatchString(r[1]) {
			return fmt.Errorf("invalid disposition: %s", r[1])
		}
		if r[1] == "case" || r[1] == "diagnostic" {
			if !ids[r[2]] {
				return fmt.Errorf("unknown case_id: %s", r[2])
			}
			mapped[r[2]] = true
		} else if r[2] != "-" {
			return fmt.Errorf("non-case disposition must use case_id -")
		}
	}
	for id := range ids {
		if !mapped[id] {
			return fmt.Errorf("case missing from inventory: %s", id)
		}
	}
	return nil
}

func fenceRun(tc *toolchainContext, bashy, toolroot, dir string, row []string, limit float64) (bool, error) {
	work, err := os.MkdirTemp("", "fence-case-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(work)
	// Copy the case directory, including relative input assets, for every case.
	cwd := work + "/work"
	source := filepath.Join(dir, row[2])
	// A copied symlink could redirect writes into the original corpus/HOME.
	if err = filepath.Walk(filepath.Dir(source), func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("case assets must not contain symlinks: %s", path)
		}
		return nil
	}); err != nil {
		return false, err
	}
	if err = copyTree(filepath.Dir(source), cwd); err != nil {
		return false, err
	}
	for _, d := range []string{"home", "tmp", "cache", "build", "bin"} {
		if err = os.MkdirAll(work+"/"+d, 0700); err != nil {
			return false, err
		}
	}
	tc.runGocache = work + "/cache"
	env := fenceEnv(tc, work, toolroot, true)
	env["PWD"] = cwd
	deadline := time.Now().Add(time.Duration(limit * float64(time.Second)))
	stage := 0
	invoke := func(argv []string, at string, e map[string]string) (*Object, error) {
		if err := fenceInterrupted(); err != nil {
			return nil, err
		}
		left := time.Until(deadline).Seconds()
		if left <= 0 {
			return nil, fmt.Errorf("case wall-time limit")
		}
		stage++
		result, err := capture(argv, at, fmt.Sprintf("%s/log/%d", work, stage), e, Flt(left), os.DevNull)
		if interrupted := fenceInterrupted(); interrupted != nil {
			return nil, interrupted
		}
		return result, err
	}
	argv := []string{bashy, "--bashsharp", filepath.Base(source)}
	if row[1] == "build" {
		s, e := invoke([]string{bashy, "check", "--bashsharp", filepath.Base(source)}, cwd, env)
		if e != nil {
			return false, e
		}
		if !success(s) {
			return false, fmt.Errorf("static check failed (%s): %s", s.Str("state"), fenceExcerpt([]byte(fenceRead(s.Obj("stderr").Str("path")))))
		}
	}
	if row[1] == "compiled" || row[1] == "build" {
		module := os.Getenv("FENCE_SH_MODULE")
		if module == "" || moduleName(module) != "mvdan.cc/sh/v3" {
			return false, fmt.Errorf("compiled mode requires FENCE_SH_MODULE naming the candidate lowering runtime")
		}
		module, err = filepath.Abs(module)
		if err != nil {
			return false, err
		}
		generated := work + "/build/main.go"
		s, e := invoke([]string{bashy, "transpile", "--bashsharp", filepath.Base(source), "-o", generated}, cwd, env)
		if e != nil {
			return false, e
		}
		if !success(s) {
			return false, fmt.Errorf("transpile failed (%s): %s", s.Str("state"), fenceExcerpt([]byte(fenceRead(s.Obj("stderr").Str("path")))))
		}
		mod := "module fencelowered\n\ngo 1.27\n\nrequire mvdan.cc/sh/v3 v3.0.0\nreplace mvdan.cc/sh/v3 => " + strconv.Quote(module) + "\n"
		if err = os.WriteFile(work+"/build/go.mod", []byte(mod), 0600); err != nil {
			return false, err
		}
		artifact := work + "/bin/case"
		if runtime.GOOS == "windows" {
			artifact += ".exe"
		}
		benv := buildEnv(tc, tc.runGocache, tc.gomodcache, work+"/home", work+"/tmp")
		s, e = invoke([]string{tc.goBinary, "build", "-o", artifact, "."}, work+"/build", benv)
		if e != nil {
			return false, e
		}
		if !success(s) {
			return false, fmt.Errorf("build failed (%s): %s", s.Str("state"), fenceExcerpt([]byte(fenceRead(s.Obj("stderr").Str("path")))))
		}
		if row[1] == "build" {
			return true, nil
		}
		argv = []string{artifact}
	}
	s, err := invoke(argv, cwd, env)
	if err != nil {
		return false, err
	}
	want, _ := strconv.ParseInt(row[4], 10, 64)
	got, has := s.Int("exit")
	badStatus := !s.Bool("spawned") || s.Str("state") != "exited" || s.Bool("descendants_survived") || !has || got != want
	expected, err := os.ReadFile(strings.TrimSuffix(source, ".bsh") + ".expected")
	if err != nil {
		return false, err
	}
	_, stdout, _ := bytes.Cut(expected, []byte("\n"))
	actual, err := os.ReadFile(s.Obj("stdout").Str("path"))
	if err != nil {
		return false, err
	}
	actualOut := actual
	stderr := []byte{}
	stderrPath := strings.TrimSuffix(source, ".bsh") + ".stderr"
	if b, e := os.ReadFile(stderrPath); e == nil {
		stderr = b
	} else if !os.IsNotExist(e) {
		return false, e
	}
	actual, err = os.ReadFile(s.Obj("stderr").Str("path"))
	if err != nil {
		return false, err
	}
	name, err := fenceComparator(source)
	if err != nil {
		return false, err
	}
	outOK, outErr := fenceCompare(stdout, actualOut, name, "stdout")
	errOK, errErr := fenceCompare(stderr, actual, name, "stderr")
	if badStatus || !outOK || !errOK {
		if outErr != nil || errErr != nil {
			return false, fmt.Errorf("comparator %s: stdout=%v stderr=%v; rc actual=%v, want %d", name, outErr, errErr, s.Get("exit"), want)
		}
		reasons := []string{}
		if badStatus {
			reasons = append(reasons, fmt.Sprintf("rc differs (state=%s rc actual=%v, want %d)", s.Str("state"), s.Get("exit"), want))
		}
		if !outOK {
			reasons = append(reasons, "stdout differs")
		}
		if !errOK {
			reasons = append(reasons, "stderr differs")
		}
		return false, fmt.Errorf("%s\n%s%s", strings.Join(reasons, "; "),
			fenceDiff("stdout", stdout, actualOut), fenceDiff("stderr", stderr, actual))
	}
	return true, nil
}
func fenceRead(path string) string { b, _ := os.ReadFile(path); return string(b) }

func fenceCheck(args []string) (result error) {
	stop := fenceTrap()
	defer stop()
	dirs := []string{}
	mode, id := "", ""
	update := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--update-counts" {
			update = true
			continue
		}
		if i+1 >= len(args) {
			return fmt.Errorf("missing value for %s", a)
		}
		i++
		switch a {
		case "--corpus":
			dirs = append(dirs, args[i])
		case "--mode":
			mode = args[i]
		case "--case":
			id = args[i]
		default:
			return fmt.Errorf("unknown option %s", a)
		}
	}
	if mode != "" && mode != "run" && mode != "compiled" && mode != "build" {
		return fmt.Errorf("invalid mode")
	}
	if len(dirs) == 0 {
		return fmt.Errorf("usage: check.sh --corpus DIR [--corpus DIR...] [--mode run|compiled|build] [--case ID] [--update-counts]")
	}
	if update && id != "" {
		return fmt.Errorf("cannot update counts for a filtered case")
	}
	limit := 20.0
	if s := os.Getenv("FENCE_TIMEOUT"); s != "" {
		v, e := strconv.ParseFloat(s, 64)
		if e != nil || !(v > 0) || v > 20 {
			return fmt.Errorf("FENCE_TIMEOUT must be >0 and <=20 seconds")
		}
		limit = v
	}
	bashy := os.Getenv("BASHY_BIN")
	if bashy == "" {
		return fmt.Errorf("missing runner: set BASHY_BIN")
	}
	bashy, err := filepath.Abs(bashy)
	if err != nil {
		return err
	}
	st, err := os.Stat(bashy)
	if err != nil || st.IsDir() || (runtime.GOOS != "windows" && st.Mode()&0111 == 0) {
		return fmt.Errorf("missing executable runner: %s", bashy)
	}
	if os.Getenv("GBE_BOOTSTRAP_GO") == "" {
		sdk, e := os.ReadFile(ROOT + "/tools/fence-corpus/.cache/adapter/sdk-path")
		if e != nil {
			return e
		}
		os.Setenv("GBE_BOOTSTRAP_GO", strings.TrimSpace(string(sdk)))
	}
	pin, err := toolchainPin(DOCS + "/toolchain.tsv")
	if err != nil {
		return err
	}
	tc := resolveToolchain(pin)
	// Resolve the union before executing ANY case, including across corpora.
	needs := map[string]bool{}
	for _, dir := range dirs {
		rows, e := fenceCases(dir)
		if e != nil {
			return e
		}
		for _, row := range rows {
			if (mode != "" && row[1] != mode) || (id != "" && row[0] != id) {
				continue
			}
			for _, need := range strings.FieldsFunc(row[3], func(r rune) bool { return r == ',' || r == ' ' }) {
				if need != "-" {
					needs[need] = true
				}
			}
		}
	}
	toolroot, release, err := fenceProvision(tc, bashy, needs)
	if err != nil {
		return err
	}
	defer func() {
		if e := release(); e != nil {
			result = fmt.Errorf("%v; cleanup failed: %w", result, e)
		}
		if e := fenceInterrupted(); e != nil {
			result = e
		}
	}()
	platform := runtime.GOOS + "/" + runtime.GOARCH
	failures, total, passes := 0, 0, 0
	type pending struct {
		path string
		data []byte
	}
	writes := []pending{}
	for _, dir := range dirs {
		dir, err = filepath.Abs(dir)
		if err != nil {
			return err
		}
		rows, e := fenceCases(dir)
		if e != nil {
			return e
		}
		counts, e := fenceTable(dir+"/counts.tsv", "mode\tplatform\tcases\tpass")
		missingCounts := os.IsNotExist(e)
		if e != nil && !missingCounts {
			return e
		}
		floors := map[string][2]int{}
		for _, r := range counts {
			a, e1 := strconv.Atoi(r[2])
			b, e2 := strconv.Atoi(r[3])
			key := r[0] + "\t" + r[1]
			if _, ok := floors[key]; ok || e1 != nil || e2 != nil || a < 0 || b < 0 || b > a || (r[0] != "run" && r[0] != "compiled" && r[0] != "build") {
				return fmt.Errorf("invalid counts row: %v", r)
			}
			floors[key] = [2]int{a, b}
		}
		measured := map[string][2]int{}
		selected := 0
		for _, r := range rows {
			if err := fenceInterrupted(); err != nil {
				return err
			}
			if (mode != "" && r[1] != mode) || (id != "" && r[0] != id) {
				continue
			}
			selected++
			key := r[1] + "\t" + platform
			n := measured[key]
			n[0]++
			total++
			ok, e := fenceRun(tc, bashy, toolroot, dir, r, limit)
			if ok {
				n[1]++
				passes++
				fmt.Printf("PASS %s %s %s\n", filepath.Base(dir), r[0], r[1])
			} else {
				failures++
				fmt.Printf("FAIL %s %s %s: %v\n", filepath.Base(dir), r[0], r[1], e)
			}
			measured[key] = n
		}
		if selected == 0 {
			failures++
			fmt.Printf("FAIL %s: no selected cases\n", dir)
		}
		// Full runs also check modes removed entirely from the manifest.
		for key := range floors {
			parts := strings.Split(key, "\t")
			if parts[1] == platform && (mode == "" || parts[0] == mode) {
				if _, ok := measured[key]; !ok {
					measured[key] = [2]int{}
				}
			}
		}
		if missingCounts && !update {
			failures++
			fmt.Printf("FAIL %s: missing counts file counts.tsv\n", dir)
		}
		for key, n := range measured {
			floor, exists := floors[key]
			if !exists && !update && !missingCounts {
				failures++
				fmt.Printf("FAIL %s: missing counts for %s\n", dir, key)
			}
			if n[0] < floor[0] || n[1] < floor[1] {
				failures++
				fmt.Printf("FAIL %s: ratchet %s executed/pass=%d/%d floor=%d/%d\n", dir, key, n[0], n[1], floor[0], floor[1])
			}
			if update {
				floors[key] = [2]int{max(n[0], floor[0]), max(n[1], floor[1])}
			}
		}
		if update {
			keys := []string{}
			for k := range floors {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var b strings.Builder
			b.WriteString("mode\tplatform\tcases\tpass\n")
			for _, k := range keys {
				n := floors[k]
				fmt.Fprintf(&b, "%s\t%d\t%d\n", k, n[0], n[1])
			}
			writes = append(writes, pending{dir + "/counts.tsv", []byte(b.String())})
		}
	}
	fmt.Printf("Summary: %d cases, %d PASS, %d failures\n", total, passes, failures)
	if failures > 0 {
		return fmt.Errorf("fence corpus gate failed")
	}
	for _, w := range writes {
		if err = os.WriteFile(w.path+".tmp", w.data, 0644); err != nil {
			return err
		}
		if err = os.Rename(w.path+".tmp", w.path); err != nil {
			return err
		}
	}
	return nil
}

// Named GBE licences apply in declaration order with GBE's stream rules.
func fenceComparator(source string) (string, error) {
	b, err := os.ReadFile(strings.TrimSuffix(source, ".bsh") + ".compare")
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(b))
	// "none" is the gate's no-normalization sentinel, not a comparator.
	for _, part := range strings.Split(name, ",") {
		if part == "none" || !contains(NormalizerNames, part) {
			return "", fmt.Errorf("unknown comparator %q", part)
		}
	}
	return name, nil
}

func fenceCompare(want, got []byte, name, stream string) (bool, error) {
	if name == "" {
		return bytes.Equal(want, got), nil
	}
	a, err := Normalize(want, strings.Split(name, ","), stream)
	if err != nil {
		return false, err
	}
	b, err := Normalize(got, strings.Split(name, ","), stream)
	if err != nil {
		return false, err
	}
	return a == b, nil
}
