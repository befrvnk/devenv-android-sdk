{ buildGoModule }:

buildGoModule {
  pname = "check-sdk-versions";
  version = "0.1.0";

  src = ./src;
  vendorHash = null;

  subPackages = [
    "cmd/check-sdk-versions"
    "cmd/generate-sdk-version-report"
  ];

  doCheck = true;
}
