package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/befrvnk/devenv-android-sdk/tools/check-sdk-versions/sdkversions"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("generate-sdk-version-report", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoJSON := flags.String("repo-json", "repo.json", "bundled Android SDK repo.json to report on")
	flakeLock := flags.String("flake-lock", "flake.lock", "flake.lock used to resolve pinned nixpkgs metadata")
	nixpkgsRepoJSON := flags.String("nixpkgs-repo-json", "", "explicit nixpkgs androidenv repo.json path; skips flake.lock resolution when set")
	githubOutput := flags.String("github-output", "", "optional GitHub Actions output file path")
	googleURL := flags.String("google-url", sdkversions.DefaultGoogleRepositoryURL, "Google Android SDK repository XML URL")
	rawGitHubBaseURL := flags.String("raw-github-base-url", sdkversions.DefaultRawGitHubBaseURL, "base URL for raw GitHub content when resolving nixpkgs from flake.lock")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	cleanup := func() {}
	resolvedNixpkgsRepoJSON := *nixpkgsRepoJSON
	if resolvedNixpkgsRepoJSON == "" {
		path, cleanupFn, err := sdkversions.FetchNixpkgsRepoJSONFromFlakeLock(*flakeLock, *rawGitHubBaseURL)
		if err != nil {
			fmt.Fprintf(stderr, "error: failed to resolve nixpkgs repo.json from flake.lock: %v\n", err)
			return 1
		}
		resolvedNixpkgsRepoJSON = path
		cleanup = cleanupFn
	}
	defer cleanup()

	report, err := sdkversions.GenerateBundledMetadataReport(*repoJSON, resolvedNixpkgsRepoJSON, *googleURL)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to generate SDK version report: %v\n", err)
		return 1
	}

	fmt.Fprint(stdout, report)
	if err := sdkversions.WriteGitHubMultilineOutput(*githubOutput, "report", report); err != nil {
		fmt.Fprintf(stderr, "error: failed to write GitHub output: %v\n", err)
		return 1
	}

	return 0
}
