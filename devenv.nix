{ pkgs, ... }:

{
  packages = [
    pkgs.go
    pkgs.gopls
    pkgs.nixfmt
  ];

  scripts.fmt-check-sdk-versions.exec = ''
    cd tools/check-sdk-versions/src
    gofmt -w .
  '';

  scripts.test-check-sdk-versions.exec = ''
    cd tools/check-sdk-versions/src
    go test ./...
  '';

  scripts.check-repo.exec = ''
    nix flake check
  '';
}
