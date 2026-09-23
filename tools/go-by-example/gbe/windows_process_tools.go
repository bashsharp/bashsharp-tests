package main

// Windows process_exec rows use a small, fixed set of commands from the same
// checksum-verified MinGit release that yoke provisions. The archive is staged
// before the gate starts; no network or ambient host PATH participates.

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const windowsMinGitSHA256 = "e3ea2944cea4b3fabcd69c7c1669ef69b1b66c05ac7806d81224d0abad2dec31"

var windowsProcessFiles = map[string]string{
	"date.exe":         "743228ae082e1128a2aff02414e114dc45a51f7461775bc5750d89307f3ccdac",
	"grep.exe":         "497202418334f8a48d9f34cd2e1b2c03c4f957e02304b613e10931c0457e8d4a",
	"ls.exe":           "b45981580702029cfeeffc8ca69e2a4d1e9c68914009e94c31ffec9cbc544aa5",
	"sh.exe":           "acf4ecb52e601f7b4a37db51b07650b5d0315eafd010590e98079fa026da4b7b",
	"msys-2.0.dll":     "2ea49553e4c03055dcf1c4a2bef54668081a07663fba283f4b34cf70f2157191",
	"msys-intl-8.dll":  "50f7de0c712d867ddca2c9eb8562ca6ee2d69e120e798ec6126ad29dd1ed5402",
	"msys-iconv-2.dll": "50012969ed96a5232220185d2fc1ac1e3abe5790f6b3027f4fc3c4cad3fd38f0",
	"msys-pcre-1.dll":  "a94b1a5ba095ac7ff81e138a1231e42e3a07391e85eeb86c3bb5c9245244597a",
}

func provisionWindowsProcessTools(cache string) (string, error) {
	archive := os.Getenv("GBE_WINDOWS_MINGIT_ZIP")
	if archive == "" {
		return "", fmt.Errorf("GBE_WINDOWS_MINGIT_ZIP must name the pinned MinGit-2.55.0.2-64-bit.zip")
	}
	got, err := sha256File(archive)
	if err != nil {
		return "", err
	}
	if got != windowsMinGitSHA256 {
		return "", fmt.Errorf("pinned MinGit archive checksum mismatch: got %s", got)
	}
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	dir := filepath.Join(cache, "mingit", "usr", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	entries := make(map[string]*zip.File, len(windowsProcessFiles))
	for _, entry := range reader.File {
		if _, ok := windowsProcessFiles[filepath.Base(entry.Name)]; ok && entry.Name == "usr/bin/"+filepath.Base(entry.Name) {
			if entries[filepath.Base(entry.Name)] != nil {
				return "", fmt.Errorf("duplicate pinned MinGit entry %s", entry.Name)
			}
			entries[filepath.Base(entry.Name)] = entry
		}
	}
	for name, want := range windowsProcessFiles {
		entry := entries[name]
		if entry == nil || !entry.FileInfo().Mode().IsRegular() {
			return "", fmt.Errorf("missing pinned MinGit file %s", name)
		}
		file, err := entry.Open()
		if err != nil {
			return "", err
		}
		data, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return "", readErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		if got := sha256Hex(data); got != want {
			return "", fmt.Errorf("pinned MinGit file %s checksum mismatch: got %s", name, got)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o755); err != nil {
			return "", err
		}
		if name == "sh.exe" {
			// MinGit distributes GNU Bash as sh.exe. Go's example asks for
			// bash, so give identical authenticated bytes that command name.
			if err := os.WriteFile(filepath.Join(dir, "bash.exe"), data, 0o755); err != nil {
				return "", err
			}
		}
	}
	return dir, nil
}
