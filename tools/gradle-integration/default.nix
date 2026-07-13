{ buildGoModule }:

buildGoModule {
  pname = "sync-android-gradle-integration";
  version = "0.1.0";

  src = ./src;
  vendorHash = null;

  subPackages = [ "cmd/sync-android-gradle-integration" ];

  doCheck = true;
  checkPhase = ''
    runHook preCheck
    go test ./...
    runHook postCheck
  '';
}
