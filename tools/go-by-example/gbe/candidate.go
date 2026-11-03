// Sprint: #155; Story: S155.10; Story-ID: 67bdd9fae2b3
//
// The reviewed-candidate binding shared by the gate and its independent
// validators, ported from tools/go-by-example/candidate.rb.
//
// Sprint 98 bound one reproducible executor digest from docs/go-by-example/
// executor.tsv. The Go-source front end is not that shape: it is a Makefile
// tag-enabled build (`make build BASHY_GOSOURCE=1`) that installs a small
// launcher beside a large `.real` payload, and whose lowering runtime is a set
// of replaced sibling modules. So the authority moved to a manifest, and the
// manifest is authenticated by the absorbed corpus primitives rather than by a
// second implementation of them here.
//
// The CLI arg contract is `--candidate MANIFEST` (`GBE_CANDIDATE` for the
// environment spelling). A caller chooses WHICH reviewed candidate to run; it
// cannot introduce one, because every field of the manifest must equal a row of
// docs/go-by-example/candidates.tsv, starting with the manifest's own digest.
package main

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// CandidateError is GoByExampleCandidate::Error.
type CandidateError struct{ msg string }

func (e *CandidateError) Error() string { return e.msg }
func candidateErr(format string, args ...any) error {
	return &CandidateError{msg: fmt.Sprintf(format, args...)}
}

// A shipped launcher is a small stub beside its payload. A candidate whose
// launcher IS the payload is a different artifact shape than the one reviewed,
// so the two digests may never coincide.
var candidateFields = []string{"manifest_sha256", "launcher_sha256", "payload_sha256", "frontend_version", "go_identity", "build_recipe", "repositories"}

// Reviewed is one authenticated row of candidates.tsv as a field map.
type Reviewed struct {
	Fields       map[string]string
	Repositories []string          // sorted basenames
	Commits      map[string]string // basename -> commit
}

func (r *Reviewed) RepositoryRecords() []any {
	out := make([]any, 0, len(r.Repositories))
	for _, name := range r.Repositories {
		out = append(out, Obj("name", name, "commit", r.Commits[name]))
	}
	return out
}

// hostIdentity is RbConfig host_os/host_cpu normalized the way candidate.rb did.
func hostIdentity() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

var (
	reHex64      = regexp.MustCompile(`\A[0-9a-f]{64}\z`)
	reAllZero    = regexp.MustCompile(`\A0+\z`)
	reRepoName   = regexp.MustCompile(`\A[a-z0-9._-]+\z`)
	reCommitHash = regexp.MustCompile(`\A[0-9a-f]{40}\z`)
)

// reviewedCandidate is GoByExampleCandidate.reviewed: the reviewed row for this
// host (the one whose manifest digest matches, or the last one). Malformed
// rows are a repository defect, not something to fall back from.
func reviewedCandidate(path string, manifestSHA string) (*Reviewed, error) {
	goos, goarch := hostIdentity()
	table, err := readTSV(path)
	if err != nil {
		return nil, candidateErr("cannot read candidates table: %s", path)
	}
	var rows [][]string
	seen := map[string]bool{}
	for _, r := range table {
		if len(r) > 1 && r[0] == goos && r[1] == goarch {
			rows = append(rows, r)
		}
	}
	for _, r := range rows {
		key := ""
		if len(r) > 2 {
			key = r[2]
		}
		if seen[key] {
			return nil, candidateErr("duplicate reviewed candidate identity")
		}
		seen[key] = true
	}
	var row []string
	if manifestSHA != "" {
		for _, r := range rows {
			if len(r) > 2 && r[2] == manifestSHA {
				row = r
			}
		}
		if row == nil {
			return nil, candidateErr("candidate manifest is not the repository-reviewed manifest (%s)", manifestSHA)
		}
	} else if len(rows) > 0 {
		row = rows[len(rows)-1]
	}
	if row == nil {
		return nil, candidateErr("no repository-reviewed Bash++ candidate for %s/%s", goos, goarch)
	}
	if len(row) != 9 {
		return nil, candidateErr("reviewed candidate row is malformed")
	}
	fields := map[string]string{}
	for i, key := range candidateFields {
		fields[key] = row[2+i]
	}
	for _, key := range []string{"manifest_sha256", "launcher_sha256", "payload_sha256"} {
		if !reHex64.MatchString(fields[key]) || reAllZero.MatchString(fields[key]) {
			return nil, candidateErr("invalid reviewed candidate %s", key)
		}
	}
	if runtime.GOOS != "windows" && fields["launcher_sha256"] == fields["payload_sha256"] {
		return nil, candidateErr("reviewed candidate launcher and payload digests are identical")
	}
	if fields["frontend_version"] == "" {
		return nil, candidateErr("reviewed candidate declares no frontend version")
	}
	// Historical diagnostic builds used an optional tag. Final default builds
	// are equally admissible only when their exact recipe and all bytes match
	// the separately reviewed candidate row and manifest.
	identityWords := strings.Fields(fields["go_identity"])
	version := ""
	if len(identityWords) > 2 {
		version = identityWords[2]
	}
	recipe := fields["build_recipe"]
	explicitRelease := version != "" && strings.Contains(recipe, "GOTOOLCHAIN="+version)
	pinnedSDKPath := false
	if version != "" && strings.Contains(recipe, "GOTOOLCHAIN=local") {
		pattern := `(?:\A|\s)PATH=(?:/|<home>/|\$HOME/)[^\s:]*golang\.org/toolchain@v0\.0\.1-` + regexp.QuoteMeta(version) + `\.` + regexp.QuoteMeta(goos) + `-` + regexp.QuoteMeta(goarch) + `/bin(?::|\s|\z)`
		pinnedSDKPath = regexp.MustCompile(pattern).MatchString(recipe)
		if runtime.GOOS == "windows" {
			pinnedSDKPath = strings.Contains(strings.ToLower(recipe), "/go/"+strings.TrimPrefix(version, "go")+"/go/bin/go.exe build")
		}
	}
	if !explicitRelease && !pinnedSDKPath {
		return nil, candidateErr("reviewed build recipe does not pin the Go toolchain: %s", rubyInspect(recipe))
	}
	names, commits, err := parseRepositories(fields["repositories"])
	if err != nil {
		return nil, err
	}
	return &Reviewed{Fields: fields, Repositories: names, Commits: commits}, nil
}

// rubyInspect renders a string the way String#inspect does for the plain
// ASCII values that reach diagnostics here.
func rubyInspect(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '#':
			if i+1 < len(s) && (s[i+1] == '{' || s[i+1] == '$' || s[i+1] == '@') {
				b.WriteString(`\#`)
			} else {
				b.WriteByte(c)
			}
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, `\x%02X`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func parseRepositories(value string) ([]string, map[string]string, error) {
	pairs := strings.Split(value, ";")
	if value == "" {
		pairs = nil
	}
	names := []string{}
	commits := map[string]string{}
	if len(pairs) == 0 {
		return nil, nil, candidateErr("reviewed candidate repository list is malformed")
	}
	for _, entry := range pairs {
		name, commit, _ := strings.Cut(entry, "=")
		if !reRepoName.MatchString(name) || !reCommitHash.MatchString(commit) {
			return nil, nil, candidateErr("reviewed candidate repository list is malformed")
		}
		names = append(names, name)
		commits[name] = commit
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	if strings.Join(sorted, ";") != strings.Join(names, ";") || len(commits) != len(names) {
		return nil, nil, candidateErr("reviewed candidate repository list is unsorted or duplicated")
	}
	return names, commits, nil
}

// Toolchain is the reviewed Go release for this host from toolchain.tsv.
type Toolchain struct {
	Version            string
	Identity           string
	GoSHA256           string
	Distribution       string
	DistributionSHA256 string
	ChecksumSource     string
}

// toolchainPin is the fixed oracle SDK coordinate, independent of the reviewed
// candidate's build coordinate.
func toolchainPin(path string) (*Toolchain, error) {
	goos, goarch := hostIdentity()
	rows, err := readTSV(path)
	if err != nil {
		return nil, candidateErr("cannot read toolchain table: %s", path)
	}
	for _, r := range rows {
		if len(r) >= 5 && r[0] == goos && r[1] == goarch {
			return &Toolchain{Version: r[2], Identity: r[3], GoSHA256: r[4]}, nil
		}
	}
	return nil, candidateErr("no authenticated Go toolchain pin for %s/%s", goos, goarch)
}

// manifestPath is GoByExampleCandidate.manifest_path: where --candidate came
// from, in precedence order. Absent is a provisioning error: there is no
// default candidate and no implicit fall-back to a binary merely found on PATH.
func manifestPath(argvValue string) (string, error) {
	path := argvValue
	if path == "" {
		path = os.Getenv("GBE_CANDIDATE")
	}
	if path == "" {
		return "", candidateErr("pass --candidate MANIFEST (or set GBE_CANDIDATE) naming the authenticated Bash++ candidate manifest; there is no default candidate")
	}
	if !isRegularFile(path) {
		return "", candidateErr("candidate manifest is not a readable file: %s", path)
	}
	return realPath(path)
}

// moduleName reads the `module` directive of path/go.mod.
func moduleName(path string) string {
	data, err := os.ReadFile(filepath.Join(path, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range rubyLinesChomp(string(data)) {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(line[len("module "):])
		}
	}
	return ""
}

// authenticateManifest is GoByExampleCandidate.authenticate: a supplied
// manifest against the reviewed table and the bytes on disk. Returns the
// provenance the gate records; errors on any divergence.
func authenticateManifest(manifest string, bashy string, row *Reviewed, tool *Toolchain, docs string) (*Object, error) {
	if err := authenticateBuildCoordinate(row, tool); err != nil {
		return nil, err
	}
	sum, err := digest(manifest)
	if err != nil {
		return nil, err
	}
	if sum != row.Fields["manifest_sha256"] {
		return nil, candidateErr("candidate manifest is not the repository-reviewed manifest (%s)", sum)
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		return nil, err
	}
	parsed, err := Parse(data)
	if err != nil {
		return nil, &jsonParseError{err}
	}
	doc, ok := parsed.(*Object)
	if !ok {
		return nil, candidateErr("candidate manifest is not a JSON object")
	}
	for _, key := range []string{"launcher_sha256", "payload_sha256", "frontend_version", "build_recipe"} {
		if v, isStr := doc.Get(key).(string); !isStr || v != row.Fields[key] {
			return nil, candidateErr("candidate manifest %s differs from the reviewed table", key)
		}
	}
	declared, isArr := doc.Get("repositories").([]any)
	if !isArr || len(declared) == 0 {
		return nil, candidateErr("candidate manifest declares no repositories")
	}
	supplied := map[string]string{}
	var suppliedNames []string
	var declaredPaths []string
	for _, entry := range declared {
		repo, _ := entry.(*Object)
		if repo == nil || !repo.Has("path") {
			return nil, candidateErr("candidate manifest declares no repositories")
		}
		path := fmt.Sprint(repo.Get("path"))
		name := filepath.Base(path)
		commit, _ := repo.Get("commit").(string)
		if _, dup := supplied[name]; !dup {
			suppliedNames = append(suppliedNames, name)
		}
		supplied[name] = commit
		declaredPaths = append(declaredPaths, path)
	}
	sort.Strings(suppliedNames)
	same := len(suppliedNames) == len(row.Repositories)
	for i := range suppliedNames {
		if !same || suppliedNames[i] != row.Repositories[i] || supplied[suppliedNames[i]] != row.Commits[suppliedNames[i]] {
			same = false
		}
	}
	if !same {
		return nil, candidateErr("candidate manifest repository set differs from the reviewed runtime dependencies: %s vs %s", rubyStringArray(suppliedNames), rubyStringArray(row.Repositories))
	}
	if !isRegularFile(bashy) || !isExecutable(bashy) {
		return nil, candidateErr("Bash++ candidate launcher missing: %s", bashy)
	}
	// The corpus primitives own launcher/payload digest authentication and the
	// clean-exact-revision check for every declared repository, including
	// untracked files.
	provenance, err := authenticateCandidate(expandPath(bashy), doc)
	if err != nil {
		return nil, err
	}
	if !nativeBinary(candidatePayloadPath(expandPath(bashy))) {
		return nil, candidateErr("candidate payload is not a native binary")
	}
	if err := authenticateBuildPayload(candidatePayloadPath(expandPath(bashy)), row); err != nil {
		return nil, err
	}
	sh := ""
	for _, path := range declaredPaths {
		if moduleName(path) == "mvdan.cc/sh/v3" {
			sh = path
			break
		}
	}
	if sh == "" {
		return nil, candidateErr("the authenticated candidate declares no mvdan.cc/sh/v3 lowering runtime; the compiled mode cannot build generated Go without it")
	}
	shReal, err := realPath(sh)
	if err != nil {
		return nil, err
	}
	candidatesSHA, err := digest(filepath.Join(docs, "candidates.tsv"))
	if err != nil {
		return nil, err
	}
	provenance.Set("manifest", Obj("path", manifest, "sha256", sum))
	provenance.Set("sh_module", Obj("path", shReal, "commit", supplied[filepath.Base(sh)]))
	provenance.Set("go_identity", row.Fields["go_identity"])
	provenance.Set("candidates_sha256", candidatesSHA)
	return provenance, nil
}

// Only the reviewed Go 1.27.2 security build may advance beyond the fixed
// Go 1.27.1 corpus oracle. The row and manifest must agree on the complete
// identity and recipe; this exception cannot select an arbitrary SDK.
func authenticateBuildCoordinate(row *Reviewed, oracle *Toolchain) error {
	goos, goarch := hostIdentity()
	identity := row.Fields["go_identity"]
	words := strings.Fields(identity)
	if len(words) != 4 || words[0] != "go" || words[1] != "version" || words[3] != goos+"/"+goarch {
		return candidateErr("invalid reviewed candidate go_identity %s", rubyInspect(identity))
	}
	version := words[2]
	if identity != oracle.Identity && !(oracle.Version == "go1.27.1" && version == "go1.27.2") {
		return candidateErr("unreviewed candidate build/oracle coordinate: %s against %s", rubyInspect(identity), rubyInspect(oracle.Identity))
	}
	// reviewedCandidate checks the recipe's pin too. Repeat the assertion here
	// so evidence validation and direct authentication use the same contract.
	explicitRelease := regexp.MustCompile(`(?:\A|\s)GOTOOLCHAIN=` + regexp.QuoteMeta(version) + `(?:\s|;|\z)`).MatchString(row.Fields["build_recipe"])
	if !explicitRelease &&
		!strings.Contains(row.Fields["build_recipe"], "/"+strings.TrimPrefix(version, "go")+"/go/bin/go.exe build") &&
		!strings.Contains(row.Fields["build_recipe"], "golang.org/toolchain@v0.0.1-"+version+"."+goos+"-"+goarch+"/bin") {
		return candidateErr("reviewed build recipe does not pin candidate %s", version)
	}
	return nil
}

func authenticateBuildPayload(path string, row *Reviewed) error {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return candidateErr("candidate payload has no readable Go build metadata: %v", err)
	}
	version := strings.Fields(row.Fields["go_identity"])[2]
	if info.GoVersion != version {
		return candidateErr("candidate payload Go build version %s differs from reviewed %s", rubyInspect(info.GoVersion), rubyInspect(version))
	}
	return nil
}

// reviewedBuildToolchain selects the independently reviewed SDK pin named by
// the already-authenticated candidate row. The fixed corpus Toolchain remains
// the oracle; build-toolchains.tsv is a separate review boundary whose Go
// binary digest was derived from the named official distribution only after
// that distribution matched the checksum published by its ChecksumSource.
func reviewedBuildToolchain(row *Reviewed, oracle *Toolchain) (*Toolchain, error) {
	return reviewedBuildToolchainAt(filepath.Join(DOCS, "build-toolchains.tsv"), row, oracle)
}

func reviewedBuildToolchainAt(path string, row *Reviewed, oracle *Toolchain) (*Toolchain, error) {
	if err := authenticateBuildCoordinate(row, oracle); err != nil {
		return nil, err
	}
	identity := row.Fields["go_identity"]
	words := strings.Fields(identity)
	version := words[2]
	if identity == oracle.Identity {
		return oracle, nil
	}
	goos, goarch := hostIdentity()
	rows, err := readTSV(path)
	if err != nil {
		return nil, candidateErr("cannot read reviewed build toolchain table: %s", path)
	}
	for _, r := range rows {
		if len(r) != 8 || r[0] != goos || r[1] != goarch || r[2] != version {
			continue
		}
		if r[3] != identity || !reHex64.MatchString(r[4]) || reAllZero.MatchString(r[4]) ||
			r[5] == "" || !reHex64.MatchString(r[6]) || reAllZero.MatchString(r[6]) ||
			r[7] != "https://go.dev/dl/?mode=json&include=all" {
			return nil, candidateErr("reviewed candidate build SDK pin is malformed")
		}
		return &Toolchain{Version: version, Identity: identity, GoSHA256: r[4],
			Distribution: r[5], DistributionSHA256: r[6], ChecksumSource: r[7]}, nil
	}
	return nil, candidateErr("no independently reviewed candidate build SDK pin for %s/%s %s", goos, goarch, version)
}

type jsonParseError struct{ err error }

func (e *jsonParseError) Error() string { return e.err.Error() }

func rubyStringArray(values []string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = rubyInspect(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// pinnedGoroot locates a reviewed SDK without asking that SDK to execute. The
// local bootstrap may identify its own GOROOT and module cache, but target Go
// bytes are checked against the independent pin before resolveToolchain runs
// `go version` or links the SDK into the managed cache.
func pinnedGoroot(toolpin *Toolchain) (string, error) {
	bootstrap := os.Getenv("GBE_BOOTSTRAP_GO")
	if bootstrap == "" {
		bootstrap = "go"
	}
	gorootCmd := exec.Command(bootstrap, "env", "GOROOT")
	gorootCmd.Env = append(envWithout("GOTOOLCHAIN"), "GOTOOLCHAIN=local")
	gorootOut, err := gorootCmd.Output()
	if err != nil {
		return "", fmt.Errorf("cannot resolve bootstrap Go GOROOT")
	}
	goroot := strings.TrimSpace(string(gorootOut))
	if authenticateToolchainRoot(goroot, toolpin) == nil {
		return goroot, nil
	}
	modcacheCmd := exec.Command(bootstrap, "env", "GOMODCACHE")
	modcacheCmd.Env = append(envWithout("GOTOOLCHAIN"), "GOTOOLCHAIN=local")
	modcacheOut, _ := modcacheCmd.Output()
	goos, goarch := hostIdentity()
	module := strings.TrimSpace(string(modcacheOut)) + "/golang.org/toolchain@v0.0.1-" + toolpin.Version + "." + goos + "-" + goarch
	if authenticateToolchainRoot(module, toolpin) == nil {
		return module, nil
	}
	return "", fmt.Errorf("cannot resolve authenticated Go toolchain %s", toolpin.Version)
}

// authenticateToolchainRoot is deliberately static: it must reject bytes
// before any executable from the candidate SDK is started or linked.
func authenticateToolchainRoot(goroot string, toolpin *Toolchain) error {
	goBinary := goExecutable(goroot)
	sum, err := sha256File(goBinary)
	if err != nil || sum != toolpin.GoSHA256 {
		return candidateErr("Go binary checksum mismatch")
	}
	versionBytes, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	lines := rubyLinesChomp(string(versionBytes))
	if err != nil || len(lines) == 0 || strings.TrimSpace(lines[0]) != toolpin.Version {
		return candidateErr("Go release source mismatch")
	}
	if !nativeBinary(goBinary) {
		return candidateErr("Go tool is not native")
	}
	return nil
}
