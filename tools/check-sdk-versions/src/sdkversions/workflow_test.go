package sdkversions

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLockedNixpkgs(t *testing.T) {
	flakeLock := writeTestFile(t, "flake.lock", `{
  "nodes": {
    "nixpkgs": {
      "locked": {
        "type": "github",
        "owner": "NixOS",
        "repo": "nixpkgs",
        "rev": "abc123",
        "narHash": "sha256-example"
      }
    }
  }
}`)

	locked, err := ReadLockedNixpkgs(flakeLock)
	if err != nil {
		t.Fatal(err)
	}
	if locked.Type != "github" || locked.Owner != "NixOS" || locked.Repo != "nixpkgs" || locked.Rev != "abc123" {
		t.Fatalf("unexpected locked nixpkgs: %#v", locked)
	}
}

func TestFetchNixpkgsRepoJSONFromFlakeLock(t *testing.T) {
	const repoJSON = `{"packages":{"platforms":{"37.0":{}}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/NixOS/nixpkgs/abc123/pkgs/development/mobile/androidenv/repo.json"
		if r.URL.Path != wantPath {
			t.Fatalf("request path = %q, want %q", r.URL.Path, wantPath)
		}
		_, _ = w.Write([]byte(repoJSON))
	}))
	defer server.Close()

	flakeLock := writeTestFile(t, "flake.lock", `{
  "nodes": {
    "nixpkgs": {
      "locked": {
        "type": "github",
        "owner": "NixOS",
        "repo": "nixpkgs",
        "rev": "abc123",
        "narHash": "sha256-example"
      }
    }
  }
}`)

	path, cleanup, err := FetchNixpkgsRepoJSONFromFlakeLock(flakeLock, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != repoJSON {
		t.Fatalf("repo.json = %q, want %q", string(data), repoJSON)
	}
}

func TestGenerateBundledMetadataReport(t *testing.T) {
	repoJSON := writeTestRepoJSON(t, `{
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

	configured := ConfiguredVersions{
		Platforms:     []string{"36"},
		BuildTools:    []string{"37.0.0"},
		PlatformTools: "37.0.0",
		Emulator:      "36.6.6",
		NDK:           "28.2.13676358",
		CmdlineTools:  "11.0",
		CMake:         []string{"3.22.1"},
	}
	report, err := GenerateBundledMetadataReport(repoJSON, repoJSON, "", configured)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Metadata used by this shell (bundled/pinned read-only):",
		"platforms",
		"⬆ platforms:     36 → 37.0",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestLoadConfiguredVersions(t *testing.T) {
	path := writeTestFile(t, "configured.json", `{
  "platforms": ["36"],
  "buildTools": ["37.0.0"],
  "platformTools": "37.0.0",
  "emulator": "36.6.6",
  "ndk": "28.2.13676358",
  "cmdlineTools": "11.0",
  "cmake": ["3.22.1"]
}`)

	configured, err := LoadConfiguredVersions(path)
	if err != nil {
		t.Fatal(err)
	}
	if configured.Platforms[0] != "36" || configured.CmdlineTools != "11.0" || configured.CMake[0] != "3.22.1" {
		t.Fatalf("unexpected configured versions: %#v", configured)
	}
}

func TestWriteGitHubMultilineOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "github-output")
	value := "line 1\nline 2\n"
	if err := WriteGitHubMultilineOutput(path, "report", value); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	want := "report<<SDK_REPORT_EOF\nline 1\nline 2\nSDK_REPORT_EOF\n"
	if got != want {
		t.Fatalf("GitHub output = %q, want %q", got, want)
	}
}

func writeTestFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
