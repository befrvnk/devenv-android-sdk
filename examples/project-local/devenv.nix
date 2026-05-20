{ ... }:

{
  androidSdk = {
    enable = true;

    repoJson = ./nix/android-sdk/repo.json;
    repoJsonWritablePath = "nix/android-sdk/repo.json";

    platforms = [
      "37.0"
      "36"
      "34"
    ];
    buildTools = [
      "37.0.0"
      "36.1.0"
      "36.0.0"
      "35.0.0"
    ];
    platformTools = "37.0.0";
    emulator = "36.6.6";
    ndk = "28.2.13676358";

    # API 37.0 source ZIPs contain duplicate entries; the module's default
    # fixDuplicateZipEntries option keeps includeSources builds non-interactive.
    includeSources = true;

    systemImageTypes = [ "google_apis_playstore" ];
    abis = [ "x86_64" ];
  };
}
