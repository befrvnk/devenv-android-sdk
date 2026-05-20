# devenv-android-sdk

Reusable [devenv](https://devenv.sh/) module for Android SDKs that can use project-controlled Android SDK metadata.

This is useful when Google has released new Android SDK packages but nixpkgs' vendored `androidenv/repo.json` has not caught up yet. The module composes the SDK with `pkgs.androidenv.composeAndroidPackages`, but lets your project provide its own `repo.json` generated directly from Google's SDK repository XML.

## Features

- Installs Android SDK packages through Nix/androidenv
- Supports project-local SDK metadata via `repoJson` + `repoJsonWritablePath`
- Adds:
  - `check-sdk-versions`
  - `update-android-sdk-repo`
- Exposes:
  - `ANDROID_HOME`
  - `ANDROID_SDK_ROOT`
  - `ANDROID_NDK_ROOT`
- Optionally sets Gradle's `aapt2FromMavenOverride` to the Nix-provided `aapt2`

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

Then configure it in `devenv.nix`:

```nix
{ ... }:

{
  androidSdk = {
    enable = true;

    # Option B: keep writable metadata in your project.
    # Commit this file so every machine gets the same SDK metadata.
    repoJson = ./nix/android-sdk/repo.json;
    repoJsonWritablePath = "nix/android-sdk/repo.json";

    platforms = [ "36" "34" ];
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

## Updating SDK metadata

When Google releases new SDK packages, run:

```bash
update-android-sdk-repo
```

This rewrites your configured `repoJsonWritablePath` from Google's SDK repository XML. Commit the changed `repo.json` so other machines can reproduce the same SDK.

After changing metadata or SDK version options, reload the shell:

```bash
direnv reload
```

## Checking available versions

```bash
check-sdk-versions
```

Example output:

```text
                       Google      project      nixpkgs
─────────────────────────────────────────────────────
platforms                37.0         37.0         37.0
build-tools            37.0.0       37.0.0       36.1.0
platform-tools         37.0.0       37.0.0       36.0.2
emulator               36.6.6       36.6.6       36.4.2
```

The `project` column is what this module can use from your configured `repo.json`.

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

  repoJson = ./nix/android-sdk/repo.json;
  repoJsonWritablePath = "nix/android-sdk/repo.json";

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
  includeSystemImages = true;
  systemImageTypes = [ "google_apis_playstore" ];
  abis = [ "x86_64" ];

  setGradleAapt2Override = true;
  addSdkPaths = true;
};
```

## Repository automation

This repository includes a scheduled GitHub Actions workflow that runs `nix run .#update-repo-json`, validates that only the bundled `repo.json` changed, runs `nix flake check`, and opens or updates an automated PR.

The workflow also attempts to enable auto-merge for that PR. For protected branches, enable repository auto-merge and consider setting an `UPDATE_REPO_JSON_TOKEN` secret backed by a fine-grained PAT or GitHub App token that is allowed to open PRs and merge them after checks pass. Pull requests created with the default `GITHUB_TOKEN` do not trigger follow-up `push` or `pull_request` workflows, so use `UPDATE_REPO_JSON_TOKEN` if branch protection requires PR status checks before auto-merge.

## Notes

- If you use the bundled `repo.json` only, `update-android-sdk-repo` cannot mutate it when the module is imported from GitHub because Nix inputs are read-only store paths.
- For fast per-project updates, set `repoJson` to a project-local committed file and `repoJsonWritablePath` to the same path as a string.
- This module accepts Android SDK licenses by default via `androidSdk.licenseAccepted = true`.
