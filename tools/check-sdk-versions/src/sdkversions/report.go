package sdkversions

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type PackageSpec struct {
	Key        string
	Label      string
	Kind       PackageKind
	Configured func(ConfiguredVersions) []string
}

var packageSpecs = []PackageSpec{
	{
		Key:   "platforms",
		Label: "platforms",
		Kind:  KindPlatform,
		Configured: func(c ConfiguredVersions) []string {
			return c.Platforms
		},
	},
	{
		Key:   "build-tools",
		Label: "build-tools",
		Kind:  KindVersionedPath,
		Configured: func(c ConfiguredVersions) []string {
			return c.BuildTools
		},
	},
	{
		Key:   "platform-tools",
		Label: "platform-tools",
		Kind:  KindRevision,
		Configured: func(c ConfiguredVersions) []string {
			return singleVersion(c.PlatformTools)
		},
	},
	{
		Key:   "emulator",
		Label: "emulator",
		Kind:  KindRevision,
		Configured: func(c ConfiguredVersions) []string {
			return singleVersion(c.Emulator)
		},
	},
	{
		Key:   "ndk",
		Label: "ndk",
		Kind:  KindVersionedPath,
		Configured: func(c ConfiguredVersions) []string {
			return singleVersion(c.NDK)
		},
	},
	{
		Key:   "cmdline-tools",
		Label: "cmdline-tools",
		Kind:  KindVersionedPath,
		Configured: func(c ConfiguredVersions) []string {
			return singleVersion(c.CmdlineTools)
		},
	},
	{
		Key:   "cmake",
		Label: "cmake",
		Kind:  KindVersionedPath,
		Configured: func(c ConfiguredVersions) []string {
			return c.CMake
		},
	},
}

func CheckSDKVersions(cfg Config, googleURL string, out io.Writer) error {
	fmt.Fprintln(out, "Checking for Android SDK updates...")
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Current versions (devenv.nix):")
	fmt.Fprintf(out, "  platforms:      %s\n", strings.Join(cfg.Configured.Platforms, ", "))
	fmt.Fprintf(out, "  build-tools:    %s\n", strings.Join(cfg.Configured.BuildTools, ", "))
	fmt.Fprintf(out, "  platform-tools: %s\n", cfg.Configured.PlatformTools)
	fmt.Fprintf(out, "  emulator:       %s\n", cfg.Configured.Emulator)
	fmt.Fprintf(out, "  ndk:            %s\n", cfg.Configured.NDK)
	fmt.Fprintf(out, "  cmdline-tools:  %s\n", cfg.Configured.CmdlineTools)
	fmt.Fprintf(out, "  cmake:          %s\n", strings.Join(cfg.Configured.CMake, ", "))
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Metadata used by this shell (%s):\n", cfg.MetadataMode)
	fmt.Fprintf(out, "  %s\n", cfg.UsedRepoJSON)
	fmt.Fprintln(out)

	if _, err := os.Stat(cfg.UsedRepoJSON); err != nil {
		fmt.Fprintf(out, "✗ Android SDK metadata does not exist: %s\n", cfg.UsedRepoJSON)
		if cfg.MetadataWritable {
			fmt.Fprintln(out, "Run 'update-android-sdk-repo' to create/update it, commit the file, then reload your shell.")
		}
		return err
	}

	usedRepo, err := LoadRepoMetadata(cfg.UsedRepoJSON)
	if err != nil {
		return fmt.Errorf("failed to read used Android SDK metadata %s: %w", cfg.UsedRepoJSON, err)
	}
	nixpkgsRepo, err := LoadRepoMetadata(cfg.NixpkgsRepoJSON)
	if err != nil {
		return fmt.Errorf("failed to read nixpkgs Android SDK metadata %s: %w", cfg.NixpkgsRepoJSON, err)
	}

	fmt.Fprintln(out, "Fetching Google repository versions...")
	googleRepo := GoogleRepository{}
	googleAvailable := false
	if googleURL != "" {
		if googleXML, err := FetchGoogleRepository(googleURL); err != nil {
			fmt.Fprintf(out, "  ⚠ Could not fetch Google's repository XML; continuing with local metadata only: %v\n", err)
		} else if parsed, err := ParseGoogleRepository(googleXML); err != nil {
			fmt.Fprintf(out, "  ⚠ Could not parse Google's repository XML; continuing with local metadata only: %v\n", err)
		} else {
			googleRepo = parsed
			googleAvailable = true
		}
	}

	latest := collectLatestVersions(usedRepo, nixpkgsRepo, googleRepo, googleAvailable)
	printComparisonTable(out, latest)

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Configured-version availability in used metadata:")
	metadataMissing := printConfiguredAvailability(out, cfg, usedRepo)
	if !metadataMissing {
		fmt.Fprintln(out, "✓ All configured versions are present in the metadata this shell uses.")
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "════════════════════════════════")
	updatesAvailable := printUpdateSuggestions(out, cfg, latest)

	googleNewer := googleAvailable && hasGoogleNewerVersions(latest)

	if metadataMissing {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Some configured versions are not present in the metadata this shell uses.")
		fmt.Fprintf(out, "Nix can only compose SDK packages that exist in: %s\n", cfg.UsedRepoJSON)
	} else if !updatesAvailable {
		fmt.Fprintln(out, "✓ Configured packages are up to date with the metadata this shell uses.")
	} else {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "To update installed SDK packages: edit androidSdk versions in devenv.nix to")
		fmt.Fprintln(out, "versions shown in the 'used' column, then reload your shell.")
	}

	if googleNewer {
		printGoogleNewerGuidance(out, cfg)
	}

	return nil
}

type VersionColumns struct {
	Google  string
	Used    string
	Nixpkgs string
}

func collectLatestVersions(usedRepo, nixpkgsRepo RepoMetadata, googleRepo GoogleRepository, googleAvailable bool) map[string]VersionColumns {
	latest := make(map[string]VersionColumns, len(packageSpecs))
	for _, spec := range packageSpecs {
		columns := VersionColumns{
			Used:    usedRepo.Latest(spec.Key),
			Nixpkgs: nixpkgsRepo.Latest(spec.Key),
		}
		if googleAvailable {
			columns.Google = googleRepo.Latest(spec.Key, spec.Kind)
		}
		latest[spec.Key] = columns
	}
	return latest
}

func printComparisonTable(out io.Writer, latest map[string]VersionColumns) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%-16s %16s %16s %16s\n", "", "Google", "used", "nixpkgs")
	fmt.Fprintln(out, "──────────────────────────────────────────────────────────────────────")
	for _, spec := range packageSpecs {
		columns := latest[spec.Key]
		fmt.Fprintf(out, "%-16s %16s %16s %16s\n", spec.Label, valueOrNA(columns.Google), valueOrNA(columns.Used), valueOrNA(columns.Nixpkgs))
	}
}

func printConfiguredAvailability(out io.Writer, cfg Config, usedRepo RepoMetadata) bool {
	missing := false
	for _, spec := range packageSpecs {
		for _, version := range spec.Configured(cfg.Configured) {
			if version == "" {
				continue
			}
			if !usedRepo.HasVersion(spec.Key, version) {
				fmt.Fprintf(out, "✗ %s: configured version %s is not in used metadata (%s)\n", spec.Label, version, cfg.UsedRepoJSON)
				missing = true
			}
		}
	}
	return missing
}

func printUpdateSuggestions(out io.Writer, cfg Config, latest map[string]VersionColumns) bool {
	updatesAvailable := false
	for _, spec := range packageSpecs {
		current := LatestVersion(spec.Configured(cfg.Configured), false)
		usedLatest := latest[spec.Key].Used
		if ShouldSuggestUpdate(current, usedLatest) {
			fmt.Fprintf(out, "⬆ %-14s %s → %s\n", spec.Label+":", current, usedLatest)
			updatesAvailable = true
		}
	}
	return updatesAvailable
}

func hasGoogleNewerVersions(latest map[string]VersionColumns) bool {
	for _, spec := range packageSpecs {
		columns := latest[spec.Key]
		if VersionGreater(columns.Google, columns.Used) {
			return true
		}
	}
	return false
}

func printGoogleNewerGuidance(out io.Writer, cfg Config) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Google has versions newer than the metadata this shell uses.")
	if cfg.MetadataWritable {
		fmt.Fprintln(out, "Metadata is project-local and writable because androidSdk.repoJsonWritablePath is set.")
		fmt.Fprintln(out, "Run 'update-android-sdk-repo', commit the changed repo.json, then reload your shell.")
		fmt.Fprintln(out, "After reloading, edit androidSdk versions in devenv.nix if needed.")
		return
	}

	fmt.Fprintln(out, "Metadata is bundled with this module because androidSdk.repoJsonWritablePath = null.")
	fmt.Fprintln(out, "Do not run 'update-android-sdk-repo' for this shell; the bundled repo.json comes")
	fmt.Fprintln(out, "from the pinned android-sdk input and is read-only in the Nix store.")
	fmt.Fprintln(out, "Update this module's bundled repo.json upstream, then update the consuming")
	fmt.Fprintln(out, "project's pinned android-sdk input and reload the shell.")
}

func singleVersion(version string) []string {
	if version == "" {
		return nil
	}
	return []string{version}
}

func valueOrNA(value string) string {
	if value == "" {
		return "n/a"
	}
	return value
}
