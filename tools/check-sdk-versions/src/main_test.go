package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSDKVersionsEndToEndBundledMetadata(t *testing.T) {
	usedRepoJSON := writeTestRepoJSON(t, `{
  "packages": {
    "platforms": { "36": {}, "37.0": {} },
    "build-tools": { "37.0.0": {}, "38.0.0-rc1": {} },
    "platform-tools": { "37.0.0": {} },
    "emulator": { "36.6.9": {} },
    "ndk": { "29.0.14206865": {}, "30.0.14904198-rc1": {} },
    "cmdline-tools": { "20.0": {}, "latest": {} },
    "cmake": { "4.1.2": {} }
  }
}`)
	nixpkgsRepoJSON := writeTestRepoJSON(t, `{
  "packages": {
    "platforms": { "36": {} },
    "build-tools": { "36.1.0": {} },
    "platform-tools": { "36.0.2": {} },
    "emulator": { "36.4.2": {} },
    "ndk": { "28.2.13676358": {} },
    "cmdline-tools": { "19.0": {} },
    "cmake": { "3.31.6": {} }
  }
}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sdk-repository>
  <remotePackage path="platforms;android-38.0"><revision><major>1</major></revision></remotePackage>
  <remotePackage path="build-tools;38.0.0"><revision><major>38</major><minor>0</minor><micro>0</micro></revision></remotePackage>
  <remotePackage path="platform-tools"><revision><major>38</major><minor>0</minor><micro>0</micro></revision></remotePackage>
  <remotePackage path="emulator"><revision><major>37</major><minor>0</minor><micro>0</micro></revision></remotePackage>
  <remotePackage path="ndk;30.0.14904198"><revision><major>30</major><minor>0</minor><micro>14904198</micro></revision></remotePackage>
  <remotePackage path="cmdline-tools;21.0"><revision><major>21</major><minor>0</minor></revision></remotePackage>
  <remotePackage path="cmake;4.2.0"><revision><major>4</major><minor>2</minor><micro>0</micro></revision></remotePackage>
</sdk-repository>`))
	}))
	defer server.Close()

	cfg := Config{
		UsedRepoJSON:     usedRepoJSON,
		NixpkgsRepoJSON:  nixpkgsRepoJSON,
		MetadataMode:     "bundled/pinned read-only",
		MetadataWritable: false,
		Configured: ConfiguredVersions{
			Platforms:     []string{"36"},
			BuildTools:    []string{"37.0.0"},
			PlatformTools: "37.0.0",
			Emulator:      "36.6.9",
			NDK:           "29.0.14206865",
			CmdlineTools:  "20.0",
			CMake:         []string{"4.1.2"},
		},
	}

	var out bytes.Buffer
	if err := CheckSDKVersions(cfg, server.URL, &out); err != nil {
		t.Fatal(err)
	}
	text := out.String()

	for _, want := range []string{
		"Metadata used by this shell (bundled/pinned read-only):",
		"platforms                    38.0             37.0               36",
		"ndk                 30.0.14904198    29.0.14206865    28.2.13676358",
		"✓ All configured versions are present in the metadata this shell uses.",
		"⬆ platforms:     36 → 37.0",
		"Google has versions newer than the metadata this shell uses.",
		"Do not run 'update-android-sdk-repo' for this shell",
		"pinned android-sdk input",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestCheckSDKVersionsReportsMissingConfiguredVersions(t *testing.T) {
	usedRepoJSON := writeTestRepoJSON(t, `{
  "packages": {
    "platforms": { "37.0": {} },
    "build-tools": { "37.0.0": {} },
    "platform-tools": { "37.0.0": {} },
    "emulator": { "36.6.9": {} },
    "ndk": { "29.0.14206865": {} },
    "cmdline-tools": { "20.0": {} },
    "cmake": { "4.1.2": {} }
  }
}`)

	cfg := Config{
		UsedRepoJSON:     usedRepoJSON,
		NixpkgsRepoJSON:  usedRepoJSON,
		MetadataMode:     "project-local writable",
		MetadataWritable: true,
		Configured: ConfiguredVersions{
			Platforms:     []string{"99.0"},
			BuildTools:    []string{"37.0.0"},
			PlatformTools: "37.0.0",
			Emulator:      "36.6.9",
			NDK:           "29.0.14206865",
			CmdlineTools:  "20.0",
			CMake:         []string{"4.1.2"},
		},
	}

	var out bytes.Buffer
	if err := CheckSDKVersions(cfg, "", &out); err != nil {
		t.Fatal(err)
	}
	text := out.String()

	for _, want := range []string{
		"✗ platforms: configured version 99.0 is not in used metadata",
		"Some configured versions are not present in the metadata this shell uses.",
		"Nix can only compose SDK packages that exist in:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func writeTestRepoJSON(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repo.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
