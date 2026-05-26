package sdkversions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultRawGitHubBaseURL = "https://raw.githubusercontent.com"

func DefaultConfiguredVersions() ConfiguredVersions {
	return ConfiguredVersions{
		Platforms:     []string{"36"},
		BuildTools:    []string{"37.0.0"},
		PlatformTools: "37.0.0",
		Emulator:      "36.6.6",
		NDK:           "28.2.13676358",
		CmdlineTools:  "11.0",
		CMake:         []string{"3.22.1"},
	}
}

func GenerateBundledMetadataReport(repoJSON, nixpkgsRepoJSON, googleURL string) (string, error) {
	cfg := Config{
		UsedRepoJSON:     repoJSON,
		NixpkgsRepoJSON:  nixpkgsRepoJSON,
		MetadataMode:     "bundled/pinned read-only",
		MetadataWritable: false,
		Configured:       DefaultConfiguredVersions(),
	}

	var out bytes.Buffer
	if err := CheckSDKVersions(cfg, googleURL, &out); err != nil {
		return "", err
	}
	return out.String(), nil
}

type FlakeLock struct {
	Nodes map[string]FlakeLockNode `json:"nodes"`
}

type FlakeLockNode struct {
	Locked FlakeLockedInput `json:"locked"`
}

type FlakeLockedInput struct {
	Type    string `json:"type"`
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Rev     string `json:"rev"`
	NarHash string `json:"narHash"`
}

func FetchNixpkgsRepoJSONFromFlakeLock(flakeLockPath, rawGitHubBaseURL string) (string, func(), error) {
	locked, err := ReadLockedNixpkgs(flakeLockPath)
	if err != nil {
		return "", nil, err
	}

	if locked.Type != "github" || locked.Owner == "" || locked.Repo == "" || locked.Rev == "" {
		return "", nil, fmt.Errorf("unsupported nixpkgs flake.lock input: type=%q owner=%q repo=%q rev=%q", locked.Type, locked.Owner, locked.Repo, locked.Rev)
	}

	if rawGitHubBaseURL == "" {
		rawGitHubBaseURL = DefaultRawGitHubBaseURL
	}
	rawGitHubBaseURL = strings.TrimRight(rawGitHubBaseURL, "/")
	url := fmt.Sprintf("%s/%s/%s/%s/pkgs/development/mobile/androidenv/repo.json", rawGitHubBaseURL, locked.Owner, locked.Repo, locked.Rev)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	tmpDir, err := os.MkdirTemp("", "check-sdk-versions-nixpkgs-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	path := filepath.Join(tmpDir, "repo.json")
	file, err := os.Create(path)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, err
	}

	return path, cleanup, nil
}

func ReadLockedNixpkgs(flakeLockPath string) (FlakeLockedInput, error) {
	data, err := os.ReadFile(flakeLockPath)
	if err != nil {
		return FlakeLockedInput{}, err
	}

	var lock FlakeLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return FlakeLockedInput{}, err
	}

	nixpkgs, ok := lock.Nodes["nixpkgs"]
	if !ok {
		return FlakeLockedInput{}, fmt.Errorf("flake.lock does not contain nodes.nixpkgs")
	}
	return nixpkgs.Locked, nil
}

func WriteGitHubMultilineOutput(path, name, value string) error {
	if path == "" {
		return nil
	}

	delimiter := "SDK_REPORT_EOF"
	for i := 0; strings.Contains(value, delimiter); i++ {
		delimiter = fmt.Sprintf("SDK_REPORT_EOF_%d", i)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s<<%s\n%s", name, delimiter, value)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(value, "\n") {
		if _, err := fmt.Fprintln(file); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(file, "%s\n", delimiter)
	return err
}
