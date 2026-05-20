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

          androidRepoRuby = pkgs.ruby.withPackages (
            rubyPackages: with rubyPackages; [
              curb
              nokogiri
              slop
            ]
          );

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
      });

      checks = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          repo-json-valid = pkgs.runCommand "android-sdk-repo-json-valid" { nativeBuildInputs = [ pkgs.jq ]; } ''
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

        }
      );
    };
}
