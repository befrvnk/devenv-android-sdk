# check-sdk-versions

Typed implementation of the `check-sdk-versions` command exposed by `module.nix`.

The tool is intentionally dependency-light: it uses only the Go standard library for JSON, XML, HTTP, and version handling. `module.nix` generates a small JSON config from the current `androidSdk` options and runs the `check-sdk-versions` binary.

## Layout

```text
tools/check-sdk-versions/
  default.nix  # Nix package for the Go commands
  README.md    # this file
  src/
    cmd/
      check-sdk-versions/          # user-facing checker command
      generate-sdk-version-report/ # GitHub Actions report helper
    sdkversions/                   # shared implementation and tests
    go.mod
```

Keeping Go code under `src/` leaves the tool root for Nix packaging and documentation files. The Nix package builds both commands.

## Responsibilities

- Read the Android SDK metadata this shell actually uses.
- Compare it with Google's current repository XML and nixpkgs' vendored `androidenv/repo.json`.
- Support decimal platform versions such as `37.0`.
- Filter preview/RC/alpha/beta versions from latest-version suggestions.
- Include platforms, build-tools, platform-tools, emulator, NDK, cmdline-tools, and CMake.
- Print workflow-specific guidance for bundled/pinned metadata vs project-local writable metadata.

## Development environment

From the repository root, enter the devenv shell:

```bash
devenv shell
```

The root `devenv.nix` provides Go, `gopls`, and `nixfmt`, plus helper scripts:

```bash
fmt-check-sdk-versions
# formats Go sources under tools/check-sdk-versions/src

test-check-sdk-versions
# runs: cd tools/check-sdk-versions/src && go test ./...

check-repo
# runs: nix flake check
```

If you do not use devenv, you can run the same commands directly with Nix:

```bash
nix shell nixpkgs#go --command sh -c 'cd tools/check-sdk-versions/src && go test ./...'
nix shell nixpkgs#nixfmt --command nixfmt module.nix flake.nix devenv.nix
```

## Running locally

Build the binaries through the repository flake:

```bash
nix build .#check-sdk-versions
```

Run the checker with a config file:

```bash
./result/bin/check-sdk-versions --config /path/to/check-sdk-versions-config.json
```

For tests that should not hit the network, pass an empty Google URL:

```bash
./result/bin/check-sdk-versions --config /path/to/config.json --google-url ''
```

Generate the bundled-metadata report used by the repository automation:

```bash
nix run .#generate-sdk-version-report -- \
  --repo-json "$PWD/repo.json" \
  --flake-lock "$PWD/flake.lock"
```

The flake app injects a Nix-generated `--configured-json` file built from `android-sdk-defaults.nix`, which is also imported by `module.nix` for the module option defaults. This keeps the automation report in sync with the Nix defaults without hardcoding versions in Go.

In GitHub Actions, pass `--github-output "$GITHUB_OUTPUT"` to also write the report to the `report` step output.

## Config shape

`module.nix` writes this structure with `pkgs.writeText`:

```json
{
  "usedRepoJson": "repo.json",
  "nixpkgsRepoJson": "/nix/store/.../pkgs/development/mobile/androidenv/repo.json",
  "metadataMode": "bundled/pinned read-only",
  "metadataWritable": false,
  "configured": {
    "platforms": ["37.0", "36"],
    "buildTools": ["37.0.0"],
    "platformTools": "37.0.0",
    "emulator": "36.6.6",
    "ndk": "28.2.13676358",
    "cmdlineTools": "11.0",
    "cmake": ["3.22.1"]
  }
}
```

## Tests

Unit tests cover:

- version parsing and comparison
- preview/RC/alpha/beta filtering
- repo.json parsing and exact configured-version checks
- Google repository XML parsing
- guidance for bundled vs writable metadata workflows
- end-to-end checker output with local fixture metadata and an in-process Google XML test server
- missing configured-version reporting
- flake.lock nixpkgs resolution for the automation helper
- loading Nix-generated configured-version JSON for the automation helper
- GitHub Actions multiline output formatting

Run:

```bash
cd tools/check-sdk-versions/src
go test ./...
```
