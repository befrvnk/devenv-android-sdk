{
  pkgs,
  config,
  lib,
  ...
}:

let
  cfg = config.androidSdk;
  androidSdkDefaults = import ./android-sdk-defaults.nix;

  androidRepoRuby = pkgs.ruby.withPackages (
    rubyPackages: with rubyPackages; [
      curb
      nokogiri
      slop
    ]
  );

  # Google occasionally publishes Android SDK source ZIPs with duplicate entries
  # (for example source-37.0_r01.zip). nixpkgs' unzip setup hook uses
  # `unzip -qq`, which prompts before replacing existing files and fails in
  # non-interactive Nix builds. Scope a tiny unzip wrapper to androidenv so
  # deploy-androidpackages.nix still uses nixpkgs unchanged, but its unpackFile
  # calls overwrite duplicate ZIP entries non-interactively with `unzip -oqq`.
  androidEnvUnzip = pkgs.runCommand "${pkgs.unzip.name}-androidenv" { preferLocalBuild = true; } ''
    mkdir -p "$out/bin" "$out/nix-support"
    ln -s ${pkgs.unzip}/bin/* "$out/bin/"

    cat > "$out/nix-support/setup-hook" <<'EOF'
    unpackCmdHooks+=(_tryUnzip)
    _tryUnzip() {
        if ! [[ "$curSrc" =~ \.zip$ ]]; then return 1; fi

        # Keep nixpkgs' UTF-8 handling, but add -o so duplicate entries in
        # Google's Android SDK source ZIPs do not trigger an interactive prompt.
        LANG=en_US.UTF-8 unzip -oqq "$curSrc"
    }
    EOF
  '';

  androidEnvCallPackage = pkgs.newScope {
    unzip = androidEnvUnzip;
    callPackage = androidEnvCallPackage;
  };

  androidEnvPkgs = pkgs // {
    unzip = androidEnvUnzip;
    callPackage = androidEnvCallPackage;
  };

  androidEnv = pkgs.androidenv.override (
    {
      licenseAccepted = cfg.licenseAccepted;
    }
    // lib.optionalAttrs (cfg.fixDuplicateZipEntries && cfg.includeSources) {
      pkgs = androidEnvPkgs;
    }
  );

  androidSdkArgs = {
    repoJson = cfg.repoJson;
    cmdLineToolsVersion = cfg.cmdLineTools;
    toolsVersion = cfg.tools;
    platformToolsVersion = cfg.platformTools;
    buildToolsVersions = cfg.buildTools;
    includeEmulator = cfg.includeEmulator;
    emulatorVersion = cfg.emulator;
    platformVersions = cfg.platforms;
    includeSources = cfg.includeSources;
    includeSystemImages = cfg.includeSystemImages;
    systemImageTypes = cfg.systemImageTypes;
    abiVersions = cfg.abis;
    cmakeVersions = cfg.cmake;
    includeNDK = cfg.includeNDK;
    ndkVersions = [ cfg.ndk ];
    useGoogleAPIs = cfg.useGoogleAPIs;
    useGoogleTVAddOns = cfg.useGoogleTVAddOns;
    includeExtras = cfg.includeExtras;
    extraLicenses = cfg.extraLicenses;
  };

  androidComposition = androidEnv.composeAndroidPackages androidSdkArgs;
  androidSdk = androidComposition.androidsdk;
  platformTools = androidComposition.platform-tools;

  repoJsonForChecks =
    if cfg.repoJsonWritablePath != null then cfg.repoJsonWritablePath else toString cfg.repoJson;

  repoJsonMetadataMode =
    if cfg.repoJsonWritablePath != null then "project-local writable" else "bundled/pinned read-only";

  checkSdkVersions = pkgs.callPackage ./tools/check-sdk-versions { };

  checkSdkVersionsConfig = pkgs.writeText "check-sdk-versions-config.json" (
    builtins.toJSON {
      usedRepoJson = repoJsonForChecks;
      nixpkgsRepoJson = "${pkgs.path}/pkgs/development/mobile/androidenv/repo.json";
      metadataMode = repoJsonMetadataMode;
      metadataWritable = cfg.repoJsonWritablePath != null;
      configured = {
        platforms = cfg.platforms;
        buildTools = cfg.buildTools;
        platformTools = cfg.platformTools;
        emulator = cfg.emulator;
        ndk = cfg.ndk;
        cmdlineTools = cfg.cmdLineTools;
        cmake = cfg.cmake;
      };
    }
  );

  updateRepoScript =
    if cfg.repoJsonWritablePath == null then
      ''
        echo "This shell uses bundled/pinned Android SDK metadata (repoJsonWritablePath = null)."
        echo "The bundled repo.json comes from the pinned android-sdk input and is read-only."
        echo ""
        echo "To update this workflow, update this module's bundled repo.json upstream,"
        echo "then update the consuming project's pinned android-sdk input and reload the shell."
        echo ""
        echo "If you intentionally want project-local writable metadata instead, configure:"
        echo ""
        echo "  androidSdk.repoJson = ./nix/android-sdk/repo.json;"
        echo "  androidSdk.repoJsonWritablePath = \"nix/android-sdk/repo.json\";"
        exit 1
      ''
    else
      ''
        set -euo pipefail

        REPO_JSON="${cfg.repoJsonWritablePath}"
        mkdir -p "$(dirname "$REPO_JSON")"

        if [ ! -f "$REPO_JSON" ]; then
          cp "${cfg.repoJson}" "$REPO_JSON"
        fi
        chmod u+w "$REPO_JSON"

        echo "Updating Android SDK metadata from Google..."
        ${androidRepoRuby}/bin/ruby -e 'load "${pkgs.path}/pkgs/development/mobile/androidenv/update.rb"' -- \
          --packages repo://repository#2-3 \
          --images image://google_apis_playstore#2-3 \
          --addons repo://addon#2-3 \
          --input "$REPO_JSON" \
          --output "$REPO_JSON"

        echo ""
        echo "✓ Updated $REPO_JSON"
        echo ""
        echo "Commit the changed $REPO_JSON, then reload your devenv shell so Nix composes"
        echo "the SDK from the updated metadata."
        echo ""
        check-sdk-versions
      '';
in
{
  options.androidSdk = {
    enable = lib.mkEnableOption "project-controlled Android SDK from androidenv metadata";

    repoJson = lib.mkOption {
      type = lib.types.path;
      default = ./repo.json;
      description = ''
        Android SDK repository metadata used by androidenv.

        For project-local metadata, set this to a committed file such as
        ./nix/android-sdk/repo.json and set repoJsonWritablePath to the same
        path as a string.
      '';
    };

    repoJsonWritablePath = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "nix/android-sdk/repo.json";
      description = ''
        Writable repository metadata path, relative to the project root, used by
        the update-android-sdk-repo script. Leave null to use the module's
        bundled read-only repo.json.
      '';
    };

    platforms = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = androidSdkDefaults.platforms;
      description = "Android platform versions to install.";
    };

    buildTools = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = androidSdkDefaults.buildTools;
      description = "Android build-tools versions to install.";
    };

    platformTools = lib.mkOption {
      type = lib.types.str;
      default = androidSdkDefaults.platformTools;
      description = "Android platform-tools version to install.";
    };

    emulator = lib.mkOption {
      type = lib.types.str;
      default = androidSdkDefaults.emulator;
      description = "Android emulator version to install.";
    };

    ndk = lib.mkOption {
      type = lib.types.str;
      default = androidSdkDefaults.ndk;
      description = "Android NDK version to install.";
    };

    cmdLineTools = lib.mkOption {
      type = lib.types.str;
      default = androidSdkDefaults.cmdLineTools;
      description = "Android command line tools version to install.";
    };

    tools = lib.mkOption {
      type = lib.types.str;
      default = androidSdkDefaults.tools;
      description = "Legacy Android SDK tools version to install.";
    };

    cmake = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = androidSdkDefaults.cmake;
      description = "CMake versions to install.";
    };

    includeEmulator = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to include the Android emulator.";
    };

    includeNDK = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to include the Android NDK.";
    };

    includeSources = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Whether to include Android platform sources.";
    };

    fixDuplicateZipEntries = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = ''
        Whether to patch androidenv's unzip setup hook when sources are enabled
        so Android SDK ZIPs with duplicate entries (such as Google's
        source-37.0_r01.zip) are unpacked non-interactively.
      '';
    };

    includeSystemImages = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to include Android system images.";
    };

    systemImageTypes = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ "google_apis_playstore" ];
      description = "Android system image types to install.";
    };

    abis = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ "x86_64" ];
      description = "Android system image ABIs to install.";
    };

    useGoogleAPIs = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to include Google APIs add-ons.";
    };

    useGoogleTVAddOns = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to include Google TV add-ons.";
    };

    includeExtras = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ "extras;google;gcm" ];
      description = "Android extras to install.";
    };

    extraLicenses = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [
        "android-sdk-preview-license"
        "android-googletv-license"
        "android-sdk-arm-dbt-license"
        "google-gdk-license"
        "intel-android-extra-license"
        "intel-android-sysimage-license"
        "mips-android-sysimage-license"
      ];
      description = "Additional Android SDK licenses to accept.";
    };

    licenseAccepted = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to accept Android SDK licenses for androidenv.";
    };

    setGradleAapt2Override = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to set Gradle's aapt2 override to the Nix-provided aapt2.";
    };

    addSdkPaths = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to add Android SDK tools to PATH in enterShell.";
    };
  };

  config = lib.mkIf cfg.enable {
    packages = [
      androidSdk
      platformTools
    ];

    env = {
      ANDROID_HOME = "${androidSdk}/libexec/android-sdk";
      ANDROID_SDK_ROOT = "${androidSdk}/libexec/android-sdk";
      ANDROID_NDK_ROOT = "${androidSdk}/libexec/android-sdk/ndk-bundle";
    }
    // lib.optionalAttrs cfg.setGradleAapt2Override {
      GRADLE_OPTS = lib.mkDefault "-Dorg.gradle.project.android.aapt2FromMavenOverride=${androidSdk}/libexec/android-sdk/build-tools/${builtins.head cfg.buildTools}/aapt2";
    };

    scripts.check-sdk-versions.exec = ''
      exec ${checkSdkVersions}/bin/check-sdk-versions --config ${checkSdkVersionsConfig}
    '';

    scripts.update-android-sdk-repo.exec = updateRepoScript;

    enterShell = lib.mkIf cfg.addSdkPaths ''
      export PATH="$PATH:$ANDROID_HOME/tools:$ANDROID_HOME/tools/bin:$ANDROID_HOME/platform-tools"
      export LD_LIBRARY_PATH="$LD_LIBRARY_PATH:${
        pkgs.lib.makeLibraryPath [
          pkgs.vulkan-loader
          pkgs.libGL
        ]
      }:$ANDROID_HOME/build-tools/${builtins.head cfg.buildTools}/lib64/:$ANDROID_NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/lib/:$LD_LIBRARY_PATH"
      export ANDROID_USER_HOME="$(pwd)/.android"
      export ANDROID_AVD_HOME="$ANDROID_USER_HOME/avd"
      mkdir -p "$ANDROID_USER_HOME" "$ANDROID_AVD_HOME"
    '';
  };
}
