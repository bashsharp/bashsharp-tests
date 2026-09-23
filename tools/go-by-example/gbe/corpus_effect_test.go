package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnapshotEffectPathsAreRelativeAcrossRoots(t *testing.T) {
	roots := []string{filepath.Join(t.TempDir(), "oracle"), filepath.Join(t.TempDir(), "interpreted")}
	var digests []string
	for _, root := range roots {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		before := snapshotListing(root)
		for name, content := range map[string]string{
			"tmp/defer.txt": "hello\n",
			"tmp/other.txt": "different\n",
		} {
			path := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		snap, err := snapshot(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"tmp", "tmp/defer.txt", "tmp/other.txt"} {
			if snap[name] == nil {
				t.Fatalf("missing relative key %q in %v", name, snap)
			}
		}
		for name := range snap {
			if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, "\\") {
				t.Fatalf("nonportable or escaped effect key %q", name)
			}
		}
		effect, err := effectDigest(before, snapshotListing(root), nil)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(effect.Str("delta"), root) {
			t.Fatalf("effect leaked absolute root: %q", effect.Str("delta"))
		}
		digests = append(digests, effect.Str("sha256"))
	}
	if digests[0] != digests[1] {
		t.Fatalf("identical effects differed across roots: %q", digests)
	}
}
