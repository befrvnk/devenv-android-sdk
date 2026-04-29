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
        in
        {
          repo-json = pkgs.runCommand "android-sdk-repo-json" { } ''
            cp ${./repo.json} $out
          '';
          default = self.packages.${system}.repo-json;
        }
      );
    };
}
