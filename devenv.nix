{ pkgs, ... }:

{
  packages = [
    pkgs.actionlint
    pkgs.go
    pkgs.gopls
    pkgs.nixfmt
    pkgs.shellcheck
  ];

  scripts.fmt-check-sdk-versions.exec = ''
    cd tools/check-sdk-versions/src
    gofmt -w .
  '';

  scripts.test-check-sdk-versions.exec = ''
    cd tools/check-sdk-versions/src
    go test ./...
  '';

  scripts.fmt-gradle-integration.exec = ''
    cd tools/gradle-integration/src
    gofmt -w .
  '';

  scripts.test-gradle-integration.exec = ''
    cd tools/gradle-integration/src
    go test ./...
  '';

  scripts.lint-workflows.exec = ''
    actionlint
  '';

  scripts.check-repo.exec = ''
    nix flake check
  '';
}
