# devenv-android-sdk

Reusable [devenv](https://devenv.sh/) module for Android SDKs that can use either this module's bundled Android SDK metadata or project-local writable metadata.

This is useful when Google has released new Android SDK packages but nixpkgs' vendored `androidenv/repo.json` has not caught up yet. The module composes the SDK with `pkgs.androidenv.composeAndroidPackages` and makes `check-sdk-versions` the canonical way to see what your configured shell can install, what Google currently publishes, and what nixpkgs contains.

## Features

- Installs Android SDK packages through Nix/androidenv
- Uses bundled/pinned SDK metadata by default (`repoJsonWritablePath = null`)
- Supports project-local writable SDK metadata via `repoJson` + `repoJsonWritablePath`
- Adds:
  - `check-sdk-versions` — canonical Android SDK update checker for this module
  - `update-android-sdk-repo` — project-local metadata updater when `repoJsonWritablePath` is set
- Exposes:
  - `ANDROID_HOME`
  - `ANDROID_SDK_ROOT`
  - `ANDROID_NDK_ROOT`
- Optionally sets Gradle's `aapt2FromMavenOverride` to the Nix-provided `aapt2`
- Provides an opt-in Gradle/Android Studio integration that keeps `local.properties` on a stable project-local SDK path
- Fixes Android SDK source ZIPs with duplicate entries (for example API 37.0 sources) by unpacking them non-interactively

## Quick start

Add this module as a devenv input.

```yaml
# devenv.yaml
inputs:
  android-sdk:
    url: github:befrvnk/devenv-android-sdk
    flake: false

imports:
  - android-sdk/module.nix
```

Then configure it in `devenv.nix`.

### Option A: bundled/pinned metadata (default)

Use this when you want SDK metadata to come from the pinned `android-sdk` input. Leave `repoJsonWritablePath = null` (the default).

```nix
{ ... }:

{
  androidSdk = {
    enable = true;

    # Uses this module's bundled repo.json from the pinned android-sdk input.
    # repoJsonWritablePath = null; # default

    platforms = [ "37.0" "36" "34" ];
    buildTools = [ "37.0.0" "36.1.0" "36.0.0" "35.0.0" ];
    platformTools = "37.0.0";
    emulator = "36.6.6";
    ndk = "28.2.13676358";

    systemImageTypes = [ "google_apis_playstore" ];
    abis = [ "x86_64" ];
  };
}
```

### Option B: project-local writable metadata

Use this when you want your project to update and commit its own `repo.json` without waiting for this module's bundled metadata to change.

```nix
{ ... }:

{
  androidSdk = {
    enable = true;

    # Commit this file so every machine gets the same SDK metadata.
    repoJson = ./nix/android-sdk/repo.json;
    repoJsonWritablePath = "nix/android-sdk/repo.json";

    platforms = [ "37.0" "36" "34" ];
    buildTools = [ "37.0.0" "36.1.0" "36.0.0" "35.0.0" ];
    platformTools = "37.0.0";
    emulator = "36.6.6";
    ndk = "28.2.13676358";

    systemImageTypes = [ "google_apis_playstore" ];
    abis = [ "x86_64" ];
  };
}
```

Bootstrap the project-local metadata file once:

```bash
mkdir -p nix/android-sdk
cp "$(nix build --no-link --print-out-paths github:befrvnk/devenv-android-sdk#repo-json)" \
  nix/android-sdk/repo.json
```

Alternatively, copy `repo.json` from this repository into `nix/android-sdk/repo.json`.

Then reload your shell:

```bash
direnv reload
# or: devenv shell
```

## Gradle and Android Studio integration

Nix store paths are immutable and include a hash derived from their contents. Changing the Android SDK composition—for example, adding a Build Tools version—normally changes `$ANDROID_HOME` from one `/nix/store/<hash>-androidsdk/...` path to another. Gradle and Android Studio often retain the old absolute path in the ignored, machine-local `local.properties` file, which can make an updated SDK appear incomplete or unwritable.

The optional Gradle integration avoids stale store paths by maintaining this indirection on every shell entry:

```text
<project>/.devenv/android-sdk -> $ANDROID_HOME

# local.properties
sdk.dir=<absolute-project-path>/.devenv/android-sdk
```

Enable it explicitly in `devenv.nix`:

```nix
androidSdk = {
  enable = true;

  platforms = [ "35" ];
  buildTools = [ "36.0.0" ];

  gradleIntegration = {
    enable = true;
    localPropertiesPath = "local.properties";
    sdkLinkPath = ".devenv/android-sdk";
  };
};
```

The integration is **disabled by default** because not every SDK consumer is a Gradle project and monorepos may use different layouts. Both paths default to the values shown above and are resolved relative to the devenv project root (`DEVENV_ROOT`), not the directory from which the shell was entered. Relative alternatives and absolute paths are supported, for example:

```nix
androidSdk.gradleIntegration = {
  enable = true;
  localPropertiesPath = "android/local.properties";
  sdkLinkPath = ".devenv/sdks/android";
};
```

On shell entry, the module creates or refreshes the SDK symlink without copying or making the Nix SDK writable. It creates `local.properties` when needed, updates or adds exactly one active `sdk.dir` entry, and preserves comments, blank lines, ordering, and unrelated properties such as tokens or plugin metadata. `local.properties` remains a mutable machine-local file. Existing regular files are updated atomically while preserving their permission bits. If `local.properties` is itself a valid symlink, the integration writes through it and preserves both the symlink and its target file. Conflicting non-symlink objects at `sdkLinkPath`, broken `local.properties` symlinks, and property files that require an update but are unwritable fail with an actionable error instead of being replaced.

After changing SDK versions or composition, reload the environment so the stable link points to the new SDK:

```bash
direnv reload
# or: devenv shell
```

You can then verify the paths without displaying unrelated `local.properties` content:

```bash
readlink .devenv/android-sdk
grep '^sdk.dir=' local.properties
```

Typical output is:

```text
/nix/store/<current-hash>-androidsdk/libexec/android-sdk
sdk.dir=/absolute/path/to/project/.devenv/android-sdk
```

Android Studio can continue using `local.properties` after it launches: `sdk.dir` stays constant while future environment reloads refresh the project-local symlink to the current `$ANDROID_HOME`.

## Checking available versions

Run this first when deciding whether to update Android SDK versions:

```bash
check-sdk-versions
```

`check-sdk-versions` compares:

- `Google`: latest stable versions currently published by Google
- `used`: latest stable versions in the metadata this shell actually uses (`androidSdk.repoJson`, or `repoJsonWritablePath` when set)
- `nixpkgs`: latest stable versions in nixpkgs' vendored `androidenv/repo.json`

It also verifies that your configured versions are present in the metadata this shell uses and suggests version updates from that same `used` metadata. Preview/RC/alpha/beta build-tools and NDK versions are filtered out of latest-version suggestions.

Example output:

```text
                           Google             used          nixpkgs
──────────────────────────────────────────────────────────────────────
platforms                    37.0             37.0             37.0
build-tools                37.0.0           37.0.0           36.1.0
platform-tools             37.0.0           37.0.0           36.0.2
emulator                   36.6.6           36.6.6           36.4.2
ndk                 29.0.14206865    29.0.14206865    28.2.13676358
cmdline-tools                20.0             20.0             19.0
cmake                       4.1.2            4.1.2           3.31.6
```

## Updating SDK metadata

There are two workflows. `check-sdk-versions` prints guidance for the mode your shell is using.

### Workflow A: bundled/pinned metadata (`repoJsonWritablePath = null`)

In this mode, the `used` metadata comes from this module's bundled `repo.json` through your pinned `android-sdk` input. It is read-only when imported from the Nix store.

If `check-sdk-versions` reports that Google has newer versions than `used` metadata:

1. Update this module's bundled `repo.json` upstream (or wait for this repository's automation to do so).
2. Update the consuming project's pinned `android-sdk` input so it points at a module revision containing the new bundled metadata.
3. Reload the shell.
4. Edit `androidSdk` version options in `devenv.nix` if needed, using versions shown in the `used` column.

Do **not** run `update-android-sdk-repo` for this workflow; it cannot mutate a bundled `repo.json` from a pinned Nix input.

### Workflow B: project-local writable metadata (`repoJsonWritablePath != null`)

In this mode, the `used` metadata is the project-local `repo.json` configured by `repoJsonWritablePath`.

When Google has newer versions, run:

```bash
update-android-sdk-repo
```

This rewrites your configured `repoJsonWritablePath` from Google's SDK repository XML. Commit the changed `repo.json` so other machines can reproduce the same SDK, then reload the shell:

```bash
git add nix/android-sdk/repo.json
git commit -m "Update Android SDK metadata"
direnv reload
```

After reloading, run `check-sdk-versions` again and edit `androidSdk` version options in `devenv.nix` if needed, using versions shown in the `used` column.

## Terminal version checks

Inside the devenv shell:

```bash
echo "$ANDROID_HOME"
ls -1 "$ANDROID_HOME/build-tools"
adb version
emulator -version | head -1
```

## Options

Common options:

```nix
androidSdk = {
  enable = true;

  # Default bundled/pinned metadata workflow:
  # repoJson = <this module's bundled repo.json>;
  # repoJsonWritablePath = null;

  # Project-local writable metadata workflow:
  # repoJson = ./nix/android-sdk/repo.json;
  # repoJsonWritablePath = "nix/android-sdk/repo.json";

  platforms = [ "36" ];
  buildTools = [ "37.0.0" ];
  platformTools = "37.0.0";
  emulator = "36.6.6";
  ndk = "28.2.13676358";

  cmdLineTools = "11.0";
  tools = "26.1.1";
  cmake = [ "3.22.1" ];

  includeEmulator = true;
  includeNDK = true;
  includeSources = false;
  fixDuplicateZipEntries = true;
  includeSystemImages = true;
  systemImageTypes = [ "google_apis_playstore" ];
  abis = [ "x86_64" ];

  setGradleAapt2Override = true;
  addSdkPaths = true;

  # Disabled by default.
  gradleIntegration = {
    enable = false;
    localPropertiesPath = "local.properties";
    sdkLinkPath = ".devenv/android-sdk";
  };
};
```

## Development

This repository includes a root `devenv.nix` for contributors. Enter it with:

```bash
devenv shell
```

Useful scripts:

```bash
fmt-check-sdk-versions   # gofmt for tools/check-sdk-versions/src
test-check-sdk-versions  # Go unit tests for the checker
fmt-gradle-integration   # gofmt for tools/gradle-integration/src
test-gradle-integration  # Go unit tests for Gradle synchronization
lint-workflows           # actionlint + shellcheck for GitHub Actions workflows
check-repo               # nix flake check
```

The `check-sdk-versions` implementation lives in `tools/check-sdk-versions`. See [`tools/check-sdk-versions/README.md`](tools/check-sdk-versions/README.md) for its design, config format, and test workflow.

## Repository automation

This repository includes a scheduled GitHub Actions workflow that runs `nix run .#update-repo-json`, validates that only the bundled `repo.json` changed, runs `nix flake check`, and opens or updates an automated PR.

The workflow also attempts to enable auto-merge for that PR. For protected branches, enable repository auto-merge and consider setting an `UPDATE_REPO_JSON_TOKEN` secret backed by a fine-grained PAT or GitHub App token that is allowed to open PRs and merge them after checks pass. Pull requests created with the default `GITHUB_TOKEN` do not trigger follow-up `push` or `pull_request` workflows, so use `UPDATE_REPO_JSON_TOKEN` if branch protection requires PR status checks before auto-merge.

## Notes

- `check-sdk-versions` is the canonical update checker for this module. Use the `used` column for version choices because that is the metadata this shell actually consumes.
- If you use the bundled `repo.json` only, `update-android-sdk-repo` cannot mutate it when the module is imported from GitHub because Nix inputs are read-only store paths.
- For fast per-project updates, set `repoJson` to a project-local committed file and `repoJsonWritablePath` to the same path as a string.
- This module accepts Android SDK licenses by default via `androidSdk.licenseAccepted = true`.
- When `includeSources = true`, `fixDuplicateZipEntries = true` scopes a patched unzip setup hook to androidenv. This works around Google source ZIPs such as `source-37.0_r01.zip` that contain duplicate entries and would otherwise make nixpkgs' default `unzip -qq` prompt during non-interactive Nix builds.
