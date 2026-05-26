package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoMetadataLatestAndHasVersion(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "repo.json")
	data := []byte(`{
  "packages": {
    "platforms": {
      "36": {},
      "37.0": {},
      "Baklava": {}
    },
    "build-tools": {
      "37.0.0": {},
      "37.0.0-rc2": {}
    },
    "ndk": {
      "29.0.14206865": {},
      "30.0.14904198-rc1": {}
    },
    "cmdline-tools": {
      "latest": {},
      "20.0": {}
    },
    "cmake": {
      "3.22.1": {},
      "4.1.2": {}
    }
  }
}`)
	if err := os.WriteFile(repoPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	repo, err := LoadRepoMetadata(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		pkg  string
		want string
	}{
		{pkg: "platforms", want: "37.0"},
		{pkg: "build-tools", want: "37.0.0"},
		{pkg: "ndk", want: "29.0.14206865"},
		{pkg: "cmdline-tools", want: "20.0"},
		{pkg: "cmake", want: "4.1.2"},
	}

	for _, tt := range tests {
		if got := repo.Latest(tt.pkg); got != tt.want {
			t.Fatalf("repo.Latest(%q) = %q, want %q", tt.pkg, got, tt.want)
		}
	}

	if !repo.HasVersion("ndk", "30.0.14904198-rc1") {
		t.Fatal("HasVersion should check exact configured versions, including previews")
	}
	if repo.HasVersion("ndk", "30.0.14904198") {
		t.Fatal("HasVersion returned true for a missing version")
	}
}
