package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

const acceptedPlatformsPath = "docs/tour/accepted-observations.tsv"

type AcceptedPlatform struct {
	GOOS, GOARCH           string
	Results, ResultsSHA256 string
	Pin, PinSHA256         string
	ToolchainIdentity      string
	ToolchainSHA256        string
}

// acceptedPlatform selects the independently recorded native observation for
// one host platform and authenticates both files before any row is consumed.
// There is deliberately no fallback: an unrecorded or ambiguous platform is
// unsupported, never silently compared with another platform's observation.
func acceptedPlatform(root, goos, goarch string) (*AcceptedPlatform, error) {
	var found *AcceptedPlatform
	for _, row := range tsvRowsLoose(filepath.Join(root, acceptedPlatformsPath)) {
		if len(row) != 8 {
			return nil, fmt.Errorf("accepted observations: row must have 8 fields")
		}
		if row[0] != goos || row[1] != goarch {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("accepted observations: duplicate platform %s/%s", goos, goarch)
		}
		found = &AcceptedPlatform{GOOS: row[0], GOARCH: row[1], Results: row[2], ResultsSHA256: row[3], Pin: row[4], PinSHA256: row[5], ToolchainIdentity: row[6], ToolchainSHA256: row[7]}
	}
	if found == nil {
		return nil, fmt.Errorf("accepted observations: unsupported platform %s/%s", goos, goarch)
	}
	for _, rel := range []string{found.Results, found.Pin} {
		if rel == "" || path.IsAbs(rel) || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || path.Clean(rel) != rel || rel == "." || rel[:1] == "." || strings.ContainsAny(rel, "\\:") {
			return nil, fmt.Errorf("accepted observations: unsafe path %q", rel)
		}
	}
	if shaFile(filepath.Join(root, filepath.FromSlash(found.Results))) != found.ResultsSHA256 {
		return nil, fmt.Errorf("accepted observations: results digest mismatch for %s/%s", goos, goarch)
	}
	if shaFile(filepath.Join(root, filepath.FromSlash(found.Pin))) != found.PinSHA256 {
		return nil, fmt.Errorf("accepted observations: pin digest mismatch for %s/%s", goos, goarch)
	}
	return found, nil
}
