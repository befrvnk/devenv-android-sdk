{
  pkgs,
  config,
  lib,
  ...
}:

let
  cfg = config.androidSdk;

  androidRepoRuby = pkgs.ruby.withPackages (
    rubyPackages: with rubyPackages; [
      curb
      nokogiri
      slop
    ]
  );

  androidEnv = pkgs.androidenv.override {
    licenseAccepted = cfg.licenseAccepted;
  };

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

  updateRepoScript =
    if cfg.repoJsonWritablePath == null then
      ''
        echo "No writable Android SDK repo.json path is configured."
        echo ""
        echo "Set androidSdk.repoJsonWritablePath in devenv.nix, for example:"
        echo ""
        echo "  androidSdk.repoJson = ./nix/android-sdk/repo.json;"
        echo "  androidSdk.repoJsonWritablePath = \"nix/android-sdk/repo.json\";"
        echo ""
        echo "Then run update-android-sdk-repo again."
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
        echo "Reload your devenv shell so Nix composes the SDK from the updated metadata."
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
      default = [ "36" ];
      description = "Android platform versions to install.";
    };

    buildTools = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ "37.0.0" ];
      description = "Android build-tools versions to install.";
    };

    platformTools = lib.mkOption {
      type = lib.types.str;
      default = "37.0.0";
      description = "Android platform-tools version to install.";
    };

    emulator = lib.mkOption {
      type = lib.types.str;
      default = "36.6.6";
      description = "Android emulator version to install.";
    };

    ndk = lib.mkOption {
      type = lib.types.str;
      default = "28.2.13676358";
      description = "Android NDK version to install.";
    };

    cmdLineTools = lib.mkOption {
      type = lib.types.str;
      default = "11.0";
      description = "Android command line tools version to install.";
    };

    tools = lib.mkOption {
      type = lib.types.str;
      default = "26.1.1";
      description = "Legacy Android SDK tools version to install.";
    };

    cmake = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ "3.22.1" ];
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
      echo "Checking for Android SDK updates..."
      echo ""

      CURRENT_PLATFORM="${builtins.head cfg.platforms}"
      CURRENT_BUILD_TOOLS="${builtins.head cfg.buildTools}"
      CURRENT_PLATFORM_TOOLS="${cfg.platformTools}"
      CURRENT_EMULATOR="${cfg.emulator}"

      echo "Current versions (devenv.nix):"
      echo "  platforms:      ${builtins.concatStringsSep ", " cfg.platforms}"
      echo "  build-tools:    ${builtins.concatStringsSep ", " cfg.buildTools}"
      echo "  platform-tools: ${cfg.platformTools}"
      echo "  emulator:       ${cfg.emulator}"
      echo ""

      latest_platform_from_json() {
        ${pkgs.jq}/bin/jq -r '.packages.platforms | keys[]' "$1" | grep -oE '^[0-9]+(\.[0-9]+)?$' | sort -V | tail -1
      }

      latest_package_from_json() {
        local repo_json="$1"
        local package="$2"
        ${pkgs.jq}/bin/jq -r ".packages.\"$package\" | keys[]" "$repo_json" | grep -v "rc\|alpha\|beta" | sort -V | tail -1
      }

      extract_version_from_xml() {
        local xml="$1"
        local pkg="$2"
        echo "$xml" | grep -A10 "path=\"$pkg\"" | head -10 | \
          grep -oE "<(major|minor|micro)>[0-9]+" | \
          sed 's/<[^>]*>//g' | \
          tr '\n' '.' | sed 's/\.$//'
      }

      echo "Fetching versions..."
      GOOGLE_REPO_XML=$(${pkgs.curl}/bin/curl -s "https://dl.google.com/android/repository/repository2-3.xml" 2>/dev/null)

      if [ -z "$GOOGLE_REPO_XML" ]; then
        echo "  ✗ Failed to fetch repository data"
        echo ""
        exit 1
      fi

      GOOGLE_PLATFORM=$(echo "$GOOGLE_REPO_XML" | grep -oE 'path="platforms;android-[0-9]+(\.[0-9]+)?"' | sed 's/path="platforms;android-//;s/"$//' | sort -V | tail -1)
      GOOGLE_BUILD_TOOLS=$(echo "$GOOGLE_REPO_XML" | grep -oE 'path="build-tools;[0-9]+\.[0-9]+\.[0-9]+"' | sed 's/path="build-tools;//;s/"$//' | sort -V | tail -1)
      GOOGLE_PLATFORM_TOOLS=$(extract_version_from_xml "$GOOGLE_REPO_XML" "platform-tools")
      GOOGLE_EMULATOR=$(extract_version_from_xml "$GOOGLE_REPO_XML" "emulator")

      PROJECT_REPO_JSON="${repoJsonForChecks}"
      NIXPKGS_REPO_JSON="${pkgs.path}/pkgs/development/mobile/androidenv/repo.json"

      PROJECT_PLATFORMS=$(latest_platform_from_json "$PROJECT_REPO_JSON")
      PROJECT_BUILD_TOOLS=$(latest_package_from_json "$PROJECT_REPO_JSON" "build-tools")
      PROJECT_PLATFORM_TOOLS=$(latest_package_from_json "$PROJECT_REPO_JSON" "platform-tools")
      PROJECT_EMULATOR=$(latest_package_from_json "$PROJECT_REPO_JSON" "emulator")

      NIXPKGS_PLATFORMS=$(latest_platform_from_json "$NIXPKGS_REPO_JSON")
      NIXPKGS_BUILD_TOOLS=$(latest_package_from_json "$NIXPKGS_REPO_JSON" "build-tools")
      NIXPKGS_PLATFORM_TOOLS=$(latest_package_from_json "$NIXPKGS_REPO_JSON" "platform-tools")
      NIXPKGS_EMULATOR=$(latest_package_from_json "$NIXPKGS_REPO_JSON" "emulator")

      echo ""
      printf "%-16s %12s %12s %12s\n" "" "Google" "project" "nixpkgs"
      echo "─────────────────────────────────────────────────────"
      printf "%-16s %12s %12s %12s\n" "platforms" "$GOOGLE_PLATFORM" "$PROJECT_PLATFORMS" "$NIXPKGS_PLATFORMS"
      printf "%-16s %12s %12s %12s\n" "build-tools" "$GOOGLE_BUILD_TOOLS" "$PROJECT_BUILD_TOOLS" "$NIXPKGS_BUILD_TOOLS"
      printf "%-16s %12s %12s %12s\n" "platform-tools" "$GOOGLE_PLATFORM_TOOLS" "$PROJECT_PLATFORM_TOOLS" "$NIXPKGS_PLATFORM_TOOLS"
      printf "%-16s %12s %12s %12s\n" "emulator" "$GOOGLE_EMULATOR" "$PROJECT_EMULATOR" "$NIXPKGS_EMULATOR"

      LATEST_PLATFORM="$PROJECT_PLATFORMS"
      LATEST_BUILD_TOOLS="$PROJECT_BUILD_TOOLS"
      LATEST_PLATFORM_TOOLS="$PROJECT_PLATFORM_TOOLS"
      LATEST_EMULATOR="$PROJECT_EMULATOR"

      echo ""
      echo "════════════════════════════════"
      UPDATES_AVAILABLE=0

      if [ "$CURRENT_PLATFORM" != "$LATEST_PLATFORM" ]; then
        echo "⬆ platforms:      $CURRENT_PLATFORM → $LATEST_PLATFORM"
        UPDATES_AVAILABLE=1
      fi

      if [ "$CURRENT_BUILD_TOOLS" != "$LATEST_BUILD_TOOLS" ]; then
        echo "⬆ build-tools:    $CURRENT_BUILD_TOOLS → $LATEST_BUILD_TOOLS"
        UPDATES_AVAILABLE=1
      fi

      if [ "$CURRENT_PLATFORM_TOOLS" != "$LATEST_PLATFORM_TOOLS" ]; then
        echo "⬆ platform-tools: $CURRENT_PLATFORM_TOOLS → $LATEST_PLATFORM_TOOLS"
        UPDATES_AVAILABLE=1
      fi

      if [ "$CURRENT_EMULATOR" != "$LATEST_EMULATOR" ]; then
        echo "⬆ emulator:       $CURRENT_EMULATOR → $LATEST_EMULATOR"
        UPDATES_AVAILABLE=1
      fi

      GOOGLE_NEWER=0
      if [ "$GOOGLE_PLATFORM" != "$PROJECT_PLATFORMS" ] || \
         [ "$GOOGLE_BUILD_TOOLS" != "$PROJECT_BUILD_TOOLS" ] || \
         [ "$GOOGLE_PLATFORM_TOOLS" != "$PROJECT_PLATFORM_TOOLS" ] || \
         [ "$GOOGLE_EMULATOR" != "$PROJECT_EMULATOR" ]; then
        GOOGLE_NEWER=1
      fi

      if [ "$UPDATES_AVAILABLE" -eq 0 ]; then
        echo "✓ All packages are up to date (using project Android SDK metadata)"
      else
        echo ""
        echo "To update: edit androidSdk versions in devenv.nix and reload your shell."
      fi

      if [ "$GOOGLE_NEWER" -eq 1 ]; then
        echo ""
        echo "Google has versions newer than $PROJECT_REPO_JSON."
        echo "Run 'update-android-sdk-repo', then edit androidSdk versions in devenv.nix if needed."
      fi
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
