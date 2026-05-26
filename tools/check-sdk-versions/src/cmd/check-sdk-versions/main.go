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
	flags := flag.NewFlagSet("check-sdk-versions", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "path to check-sdk-versions JSON config")
	googleURL := flags.String("google-url", sdkversions.DefaultGoogleRepositoryURL, "Google Android SDK repository XML URL")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *configPath == "" {
		fmt.Fprintln(stderr, "error: --config is required")
		return 2
	}

	cfg, err := sdkversions.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to load config: %v\n", err)
		return 1
	}

	if err := sdkversions.CheckSDKVersions(cfg, *googleURL, stdout); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}
