{
  description = "Reusable devenv module for project-controlled Android SDK metadata";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devenvModules.default = import ./module.nix;

      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          androidSdkDefaults = import ./android-sdk-defaults.nix;
          sdkVersionReportConfiguredJson = pkgs.writeText "sdk-version-report-configured.json" (
            builtins.toJSON {
              platforms = androidSdkDefaults.platforms;
              buildTools = androidSdkDefaults.buildTools;
              platformTools = androidSdkDefaults.platformTools;
              emulator = androidSdkDefaults.emulator;
              ndk = androidSdkDefaults.ndk;
              cmdlineTools = androidSdkDefaults.cmdLineTools;
              cmake = androidSdkDefaults.cmake;
            }
          );

          androidRepoRuby = pkgs.ruby.withPackages (
            rubyPackages: with rubyPackages; [
              curb
              nokogiri
              slop
            ]
          );

          gradleIntegration = pkgs.callPackage ./tools/gradle-integration { };

          generateSdkVersionReport = pkgs.writeShellApplication {
            name = "generate-sdk-version-report";
            text = ''
              exec ${self.packages.${system}.check-sdk-versions}/bin/generate-sdk-version-report \
                --configured-json ${sdkVersionReportConfiguredJson} \
                "$@"
            '';
          };

          updateRepoJson = pkgs.writeShellApplication {
            name = "update-repo-json";
            runtimeInputs = [
              pkgs.coreutils
              androidRepoRuby
            ];
            text = ''
              repo_json="''${ANDROID_SDK_REPO_JSON:-repo.json}"

              if [ ! -f "$repo_json" ]; then
                echo "Android SDK repo metadata does not exist: $repo_json" >&2
                echo "Set ANDROID_SDK_REPO_JSON to update a different path." >&2
                exit 1
              fi

              mkdir -p "$(dirname "$repo_json")"
              chmod u+w "$repo_json"

              tmp="$(mktemp)"
              trap 'rm -f "$tmp"' EXIT

              echo "Updating Android SDK metadata from Google into $repo_json..."
              ruby -e 'load "${pkgs.path}/pkgs/development/mobile/androidenv/update.rb"' -- \
                --packages repo://repository#2-3 \
                --images image://google_apis_playstore#2-3 \
                --addons repo://addon#2-3 \
                --input "$repo_json" \
                --output "$tmp"

              mv "$tmp" "$repo_json"
              trap - EXIT

              echo "✓ Updated $repo_json"
            '';
          };
        in
        {
          repo-json = pkgs.runCommand "android-sdk-repo-json" { } ''
            cp ${./repo.json} $out
          '';

          check-sdk-versions = pkgs.callPackage ./tools/check-sdk-versions { };

          gradle-integration = gradleIntegration;

          generate-sdk-version-report = generateSdkVersionReport;

          update-repo-json = updateRepoJson;

          default = self.packages.${system}.repo-json;
        }
      );

      apps = forAllSystems (system: {
        update-repo-json = {
          type = "app";
          program = "${self.packages.${system}.update-repo-json}/bin/update-repo-json";
          meta.description = "Update repo.json from Google's Android SDK repository metadata";
        };

        generate-sdk-version-report = {
          type = "app";
          program = "${self.packages.${system}.generate-sdk-version-report}/bin/generate-sdk-version-report";
          meta.description = "Generate an Android SDK version report for metadata update PRs";
        };
      });

      checks = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          lib = nixpkgs.lib;

          evalGradleIntegrationModule =
            enabled:
            (lib.evalModules {
              specialArgs = { inherit pkgs; };
              modules = [
                (
                  { lib, ... }:
                  {
                    options = {
                      packages = lib.mkOption {
                        type = lib.types.anything;
                        default = [ ];
                      };
                      env = lib.mkOption {
                        type = lib.types.anything;
                        default = { };
                      };
                      scripts = lib.mkOption {
                        type = lib.types.anything;
                        default = { };
                      };
                      enterShell = lib.mkOption {
                        type = lib.types.lines;
                        default = "";
                      };
                    };
                  }
                )
                self.devenvModules.default
                {
                  androidSdk = {
                    enable = true;
                    addSdkPaths = false;
                    gradleIntegration.enable = enabled;
                  };
                }
              ];
            }).config.enterShell;

          disabledGradleIntegrationEnterShell = evalGradleIntegrationModule false;
          enabledGradleIntegrationEnterShell = evalGradleIntegrationModule true;
          disabledGradleIntegrationScript = pkgs.writeShellScript "disabled-gradle-integration-enter-shell" disabledGradleIntegrationEnterShell;
          enabledGradleIntegrationScript = pkgs.writeShellScript "enabled-gradle-integration-enter-shell" enabledGradleIntegrationEnterShell;
        in
        {
          repo-json-valid =
            pkgs.runCommand "android-sdk-repo-json-valid" { nativeBuildInputs = [ pkgs.jq ]; }
              ''
                jq -e '
                  type == "object"
                  and has("addons")
                  and has("extras")
                  and has("images")
                  and has("latest")
                  and has("licenses")
                  and has("packages")
                ' ${./repo.json} > /dev/null

                cp ${./repo.json} $out
              '';

          check-sdk-versions = self.packages.${system}.check-sdk-versions;

          gradle-integration =
            assert disabledGradleIntegrationEnterShell == "";
            assert lib.hasInfix "sync-android-gradle-integration" enabledGradleIntegrationEnterShell;
            pkgs.runCommand "android-sdk-gradle-integration-tests"
              {
                nativeBuildInputs = [
                  pkgs.coreutils
                  pkgs.gnugrep
                  self.packages.${system}.gradle-integration
                ];
              }
              ''
                project_root="$TMPDIR/project"
                android_home="$TMPDIR/fake-android-sdk"
                mkdir -p "$project_root" "$android_home"

                DEVENV_ROOT="$project_root" ANDROID_HOME="$android_home" \
                  ${disabledGradleIntegrationScript}
                test ! -e "$project_root/.devenv/android-sdk"
                test ! -e "$project_root/local.properties"

                DEVENV_ROOT="$project_root" ANDROID_HOME="$android_home" \
                  ${enabledGradleIntegrationScript}
                test -L "$project_root/.devenv/android-sdk"
                test "$(readlink "$project_root/.devenv/android-sdk")" = "$android_home"
                grep -Fx "sdk.dir=$project_root/.devenv/android-sdk" \
                  "$project_root/local.properties" > /dev/null

                touch $out
              '';

          workflows =
            pkgs.runCommand "github-workflows-valid"
              {
                nativeBuildInputs = [
                  pkgs.actionlint
                  pkgs.findutils
                  pkgs.shellcheck
                ];
              }
              ''
                mapfile -d "" workflow_files < <(
                  find ${./.github/workflows} -type f \( -name '*.yml' -o -name '*.yaml' \) -print0 | sort -z
                )

                if [ "''${#workflow_files[@]}" -eq 0 ]; then
                  echo "No GitHub Actions workflow files found." >&2
                  exit 1
                fi

                actionlint "''${workflow_files[@]}"
                touch $out
              '';
        }
      );
    };
}
