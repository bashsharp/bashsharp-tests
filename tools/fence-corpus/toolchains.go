package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// These are actual fence calls, so provisioning follows the candidate's own
// defaults and resolvers (including transitive tools such as Rust's linker).
var fenceProbes = map[string]string{
	"python":     "~~~python\ndef ready() -> int: return 1\n~~~\nx := ready()\n",
	"typescript": "~~~typescript\nexport function ready(): number { return 1; }\n~~~\nx := ready()\n",
	"javascript": "~~~javascript\nexport function ready() { return 1; }\n~~~\nx := ready()\n",
	"rust":       "~~~rust\npub fn ready() -> i64 { 1 }\n~~~\nx := ready()\n",
	"c":          "~~~c\nint ready(void) { return 1; }\n~~~\nx := ready()\n",
	"cpp":        "~~~cpp\nint ready() { return 1; }\n~~~\nx := ready()\n",
	"go":         "~~~go\nfunc Ready() int { return 1 }\n~~~\nx := Ready()\n",
	"powershell": "~~~powershell\nfunction Ready { return 1 }\n~~~\nx := Ready()\n",
	"csharp":     "~~~csharp\npublic static int Ready() { return 1; }\n~~~\nx := Ready()\n",
	"bash":       "~~~bash\nready() { echo 1; }\n~~~\nx := ready()\n",
	"sh":         "~~~sh\nready() { echo 1; }\n~~~\nx := ready()\n",
}

func fenceEnv(tc *toolchainContext, work, root string, offline bool) map[string]string {
	// runEnv constructs an allowlist; capture replaces, never merges, os.Environ.
	env := runEnv(tc, work, nil)
	env["BASHY_BIN_CACHE"] = root + "/bin"
	// yoke external/python.EnsureInterpreter passes these through to uv's
	// managed-only `python find/install`; BASHY_BIN_CACHE alone is insufficient.
	env["UV_PYTHON_INSTALL_DIR"] = root + "/python"
	env["UV_CACHE_DIR"] = root + "/uv"
	env["UV_NO_CONFIG"] = "1"
	env["GOWORK"] = "off"
	env["GOTELEMETRY"] = "off"
	env["CARGO_NET_OFFLINE"] = "true"
	env["PYTHONDONTWRITEBYTECODE"] = "1"
	if offline {
		env["BASHY_OFFLINE"] = "1"
		env["UV_OFFLINE"] = "1"
		// uv initializes writable cache metadata even for `python find`.
		// Only scratch metadata is per-case; managed interpreters stay read-only.
		env["UV_CACHE_DIR"] = work + "/cache/uv"
		env["UV_PYTHON_DOWNLOADS"] = "never"
	}
	return env
}

// Do not follow symlinks: the authenticated Go SDK is linked, not owned here.
func fenceToolMode(root string, writable bool) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		mode := info.Mode().Perm() &^ 0222
		if writable {
			mode |= 0200
			if info.IsDir() {
				mode |= 0700
			}
		}
		return os.Chmod(path, mode)
	})
}

func fenceProvision(tc *toolchainContext, bashy string, needs map[string]bool) (string, func(), error) {
	base := ROOT + "/tools/fence-corpus/.cache"
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", nil, err
	}
	run, err := os.MkdirTemp(base, "run-")
	if err != nil {
		return "", nil, err
	}
	root := os.Getenv("FENCE_TOOLCHAIN_DIR")
	if root == "" {
		root = run + "/toolchains"
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return "", nil, err
	}
	// Hold the lease through all cases, so reuse cannot thaw another run's tools.
	lock := root + "/.fence-lock"
	if err = os.Mkdir(lock, 0700); err != nil {
		return "", nil, fmt.Errorf("toolchain directory busy: %s: %w", root, err)
	}
	release := func() { os.Remove(lock) }
	good := false
	defer func() {
		if !good {
			release()
		}
	}()
	logPath := run + "/provision.log"
	log, err := os.Create(logPath)
	if err != nil {
		return "", nil, err
	}
	defer log.Close()
	fmt.Println("Provision log:", logPath)
	fail := func(e error) (string, func(), error) {
		fmt.Fprintln(log, e)
		return "", nil, fmt.Errorf("toolchain provisioning failed; log: %s\n%s", logPath, fenceExcerpt([]byte(fenceRead(logPath))))
	}
	for _, d := range []string{"home", "tmp", "cache"} {
		if err = os.MkdirAll(run+"/"+d, 0700); err != nil {
			return fail(err)
		}
	}
	for _, d := range []string{"bin", "python", "uv"} {
		if err = os.MkdirAll(root+"/"+d, 0700); err != nil {
			return fail(err)
		}
		if err = fenceToolMode(root+"/"+d, true); err != nil {
			return fail(err)
		}
	}
	defer func() {
		for _, d := range []string{"bin", "python", "uv"} {
			fenceToolMode(root+"/"+d, false)
		}
	}()
	tc.binCache = root + "/bin"
	tc.runGocache = run + "/cache"
	// The shared helper creates a new link; reuse must verify the existing SDK.
	link := filepath.Join(tc.binCache, "go", strings.TrimPrefix(tc.goVersion, "go"), "go")
	if st, e := os.Stat(link); e == nil {
		target, e := os.Stat(tc.goroot)
		if e != nil || !os.SameFile(st, target) || sha(goExecutable(link)) != tc.goSHA256 {
			return fail(fmt.Errorf("reused Go SDK does not match authenticated toolchain: %s", link))
		}
	} else if err = provisionManagedToolCaches(tc.binCache, tc); err != nil {
		return fail(err)
	}
	aliases := map[string]string{"py": "python", "ts": "typescript", "node": "javascript", "js": "javascript", "rs": "rust", "cc": "c", "zig": "c", "c++": "cpp", "pwsh": "powershell", "ps1": "powershell", "cs": "csharp"}
	canonical := map[string]bool{}
	for need := range needs {
		if v := aliases[need]; v != "" {
			need = v
		}
		canonical[need] = true
	}
	names := []string{}
	for need := range canonical {
		names = append(names, need)
	}
	sort.Strings(names)
	for _, need := range names {
		probe, ok := fenceProbes[need]
		if !ok {
			return fail(fmt.Errorf("unsupported fence need %q", need))
		}
		file := "provision-" + need + ".bsh"
		if err = os.WriteFile(run+"/"+file, []byte(probe), 0600); err != nil {
			return fail(err)
		}
		fmt.Fprintln(log, "Provision", need)
		s, e := capture([]string{bashy, "--bashsharp", file}, run, run+"/logs/"+need, fenceEnv(tc, run, root, false), Int(120), os.DevNull)
		if e != nil {
			return fail(e)
		}
		fmt.Fprint(log, fenceRead(s.Obj("stdout").Str("path")), fenceRead(s.Obj("stderr").Str("path")))
		if !success(s) {
			return fail(fmt.Errorf("%s: state=%s rc=%v", need, s.Str("state"), s.Get("exit")))
		}
	}
	for _, d := range []string{"bin", "python", "uv"} {
		if err = fenceToolMode(root+"/"+d, false); err != nil {
			return fail(err)
		}
	}
	good = true
	return root, release, nil
}

func fenceExcerpt(b []byte) string {
	lines := bytes.SplitAfter(b, []byte("\n"))
	if len(lines) > 20 {
		lines = lines[:20]
	}
	out := bytes.Join(lines, nil)
	if len(out) > 4096 {
		out = out[:4096]
	}
	return string(out)
}

func fenceDiff(stream string, want, got []byte) string {
	if bytes.Equal(want, got) {
		return stream + ": bytes match\n"
	}
	a, b := strings.SplitAfter(string(want), "\n"), strings.SplitAfter(string(got), "\n")
	var out strings.Builder
	fmt.Fprintf(&out, "%s bytes differ (expected %d bytes, actual %d bytes)\n", stream, len(want), len(got))
	shown := 0
	for i := 0; i < len(a) || i < len(b); i++ {
		x, y := "<EOF>", "<EOF>"
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x == y {
			continue
		}
		// Quoting makes CR, missing final newline, and control bytes visible.
		if len(x) > 200 {
			x = x[:200] + "…"
		}
		if len(y) > 200 {
			y = y[:200] + "…"
		}
		fmt.Fprintf(&out, "line %d - %q\n         + %q\n", i+1, x, y)
		shown++
		if shown == 20 {
			out.WriteString("... excerpt limited to 20 differing lines\n")
			break
		}
	}
	return out.String()
}
