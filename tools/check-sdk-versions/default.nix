{ buildGoModule }:

buildGoModule {
  pname = "check-sdk-versions";
  version = "0.1.0";

  src = ./src;
  vendorHash = null;

  doCheck = true;
}
