// Sprint: #155; Story: S155.10; Story-ID: 67bdd9fae2b3
//
// Supported telemetry opt-outs, configured identically in every mode's
// isolated HOME before the effect baseline is taken. Ported from
// tools/go-by-example/runtime-config.rb.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var reTelemetryMode = regexp.MustCompile(`\Aoff(?: \d{4}-\d{2}-\d{2})?\z`)

// Initialize through the authenticated product before measuring any row. A
// linked SDK alone is not a complete binmgr tree: the producer owns its cache
// publication protocol. Never synthesize its completion marker in the harness.
func initializeManagedToolCaches(candidate, cache, logs string, env map[string]string, run func([]string, string, string, map[string]string, Number, string) (*Object, error), toolchains ...*toolchainContext) error {
	seen := map[string]bool{}
	for _, tc := range toolchains {
		if tc == nil || seen[tc.goVersion] {
			continue
		}
		if err := authenticateManagedSDK(tc.goroot, tc); err != nil {
			return err
		}
		version := strings.TrimPrefix(tc.goVersion, "go")
		before := newInputBinding(cache, false, "shared")
		initEnv := map[string]string{}
		for key, value := range env {
			initEnv[key] = value
		}
		initEnv["BASHY_BIN_CACHE"] = cache
		initEnv["BASHY_GO_VERSION"] = version
		prefix := filepath.Join(logs, version)
		stage, runErr := run([]string{candidate, "go", "version"}, filepath.Dir(cache), prefix, initEnv, Int(180), os.DevNull)
		evidence := Obj("version", tc.goVersion, "expected_go_sha256", tc.goSHA256, "stage", nil, "changed_paths", before.Changes())
		if stage != nil {
			evidence.Set("stage", stage)
		}
		if runErr != nil {
			evidence.Set("error", runErr.Error())
		}
		if err := os.MkdirAll(logs, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(prefix+".initialization.json", []byte(Generate(evidence)+"\n"), 0o644); err != nil {
			return err
		}
		if runErr != nil || stage == nil || !success(stage) {
			return fmt.Errorf("managed SDK %s initialization failed; see %s", version, prefix)
		}
		installedRoot := filepath.Join(cache, "go", version, "go")
		authErr := authenticateManagedSDK(installedRoot, tc)
		if authErr == nil {
			authErr = authenticateManagedSDK(tc.goroot, tc)
		}
		evidence.Set("installed_go_sha256", sha(goExecutable(installedRoot)))
		evidence.Set("original_go_sha256", sha(goExecutable(tc.goroot)))
		evidence.Set("authenticated", authErr == nil)
		if authErr != nil {
			evidence.Set("error", authErr.Error())
		}
		if err := os.WriteFile(prefix+".initialization.json", []byte(Generate(evidence)+"\n"), 0o644); err != nil {
			return err
		}
		if authErr != nil {
			return authErr
		}
		seen[tc.goVersion] = true
	}
	return nil
}

func authenticateManagedSDK(root string, tc *toolchainContext) error {
	if got := sha(goExecutable(root)); got != tc.goSHA256 {
		return fmt.Errorf("managed SDK binary checksum mismatch at %s", root)
	}
	data, err := os.ReadFile(filepath.Join(root, "VERSION"))
	lines := rubyLinesChomp(string(data))
	if err != nil || len(lines) == 0 || strings.TrimSpace(lines[0]) != tc.goVersion {
		return fmt.Errorf("managed SDK VERSION mismatch at %s", root)
	}
	return nil
}

// provisionManagedToolCache seeds the gate-owned cache with the SDK
// authenticated by resolveToolchain. initializeManagedToolCaches must finish
// the product-owned installation protocol before any measured row starts.
func provisionManagedToolCache(cache string, tc *toolchainContext) (string, error) {
	version := strings.TrimPrefix(tc.goVersion, "go")
	if version == "" || version == tc.goVersion {
		return "", fmt.Errorf("invalid pinned Go version %q", tc.goVersion)
	}
	if got := sha(tc.goBinary); got != tc.goSHA256 {
		return "", fmt.Errorf("authenticated Go binary checksum changed: got %s", got)
	}
	versionBytes, err := os.ReadFile(filepath.Join(tc.goroot, "VERSION"))
	if err != nil || len(rubyLinesChomp(string(versionBytes))) == 0 || strings.TrimSpace(rubyLinesChomp(string(versionBytes))[0]) != tc.goVersion {
		return "", fmt.Errorf("authenticated GOROOT version changed")
	}
	link := filepath.Join(cache, "go", version, "go")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return "", err
	}
	if err := linkSDK(link, tc.goroot); err != nil {
		return "", err
	}
	// Windows junctions may retain their own path through EvalSymlinks. File
	// identity proves this managed directory names the authenticated SDK.
	linked, err := os.Stat(link)
	target, targetErr := os.Stat(tc.goroot)
	if err != nil || targetErr != nil || !os.SameFile(linked, target) {
		return "", fmt.Errorf("managed Go SDK target mismatch")
	}
	fastGo := goExecutable(link)
	if got := sha(fastGo); got != tc.goSHA256 {
		return "", fmt.Errorf("managed Go binary checksum mismatch: got %s", got)
	}
	return fastGo, nil
}

// provisionManagedToolCaches prepares every reviewed SDK coordinate before an
// execution root is snapshotted. A security-build candidate may differ from
// the fixed corpus oracle; deduplicate identical versions, but never defer a
// distinct candidate SDK to the measured interpreter.
func provisionManagedToolCaches(cache string, toolchains ...*toolchainContext) error {
	seen := map[string]bool{}
	for _, tc := range toolchains {
		if tc == nil || seen[tc.goVersion] {
			continue
		}
		if _, err := provisionManagedToolCache(cache, tc); err != nil {
			return err
		}
		seen[tc.goVersion] = true
	}
	return nil
}

// configureRuntime is GoByExampleRuntimeConfig.configure: `go telemetry off`
// then `go env -json GOTELEMETRY GOTELEMETRYDIR`, both captured, then the mode
// file proved to live inside the isolated HOME.
func configureRuntime(goBinary string, root string, env map[string]string, deadline float64, logPrefix string) *Object {
	stages := []any{}
	fail := func(message string) *Object {
		return Obj("state", "configuration_failure", "detail", message, "stages", stages)
	}
	commands := [][]string{{goBinary, "telemetry", "off"}, {goBinary, "env", "-json", "GOTELEMETRY", "GOTELEMETRYDIR"}}
	var last *Object
	for index, argv := range commands {
		budget := deadline - monotonicSeconds()
		if !(budget > 0) {
			return fail("runtime configuration deadline expired")
		}
		// `[budget, 20].min`: the Integer 20 unless the Float budget is smaller.
		timeout := Int(20)
		if budget < 20 {
			timeout = Flt(budget)
		}
		stage, err := capture(argv, root, logPrefix+"/"+itoa(index), env, timeout, os.DevNull)
		if err != nil {
			return fail(err.Error())
		}
		stages = append(stages, stage)
		last = stage
		if !success(stage) {
			return fail("SDK telemetry configuration failed")
		}
	}
	data, err := os.ReadFile(last.Obj("stdout").Str("path"))
	if err != nil {
		return fail(err.Error())
	}
	config, err := ParseObject(data)
	if err != nil {
		return fail(err.Error())
	}
	if config.Str("GOTELEMETRY") != "off" {
		return fail("Go telemetry did not report off")
	}
	dir, ok := config.Get("GOTELEMETRYDIR").(string)
	if !ok {
		return fail("key not found: \"GOTELEMETRYDIR\"")
	}
	mode := filepath.Join(dir, "mode")
	modeReal, err := realPath(mode)
	if err != nil {
		return fail(err.Error())
	}
	rootReal, err := realPath(root)
	if err != nil {
		return fail(err.Error())
	}
	rel, err := filepath.Rel(rootReal, modeReal)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fail("telemetry configuration escaped isolated HOME")
	}
	record, err := fileRecord(mode)
	if err != nil {
		return fail(err.Error())
	}
	return Obj("state", "complete", "environment", Obj("OTEL_TRACES_EXPORTER", env["OTEL_TRACES_EXPORTER"]),
		"go_mode", "off", "mode_file", record, "stages", stages)
}
