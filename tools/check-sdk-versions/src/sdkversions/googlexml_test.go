package sdkversions

import "testing"

func TestGoogleRepositoryLatest(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8"?>
<sdk-repository>
  <remotePackage path="platforms;android-36">
    <revision><major>1</major></revision>
  </remotePackage>
  <remotePackage path="platforms;android-37.0">
    <revision><major>1</major></revision>
  </remotePackage>
  <remotePackage path="build-tools;37.0.0-rc2">
    <revision><major>37</major><minor>0</minor><micro>0</micro><preview>2</preview></revision>
  </remotePackage>
  <remotePackage path="build-tools;37.0.0">
    <revision><major>37</major><minor>0</minor><micro>0</micro></revision>
  </remotePackage>
  <remotePackage path="platform-tools">
    <revision><major>37</major><minor>0</minor><micro>0</micro></revision>
  </remotePackage>
  <remotePackage path="emulator">
    <revision><major>36</major><minor>6</minor><micro>9</micro></revision>
  </remotePackage>
  <remotePackage path="ndk;29.0.14206865">
    <revision><major>29</major><minor>0</minor><micro>14206865</micro></revision>
  </remotePackage>
  <remotePackage path="ndk;30.0.14904198">
    <revision><major>30</major><minor>0</minor><micro>14904198</micro><preview>1</preview></revision>
  </remotePackage>
  <remotePackage path="cmdline-tools;latest">
    <revision><major>20</major><minor>0</minor></revision>
  </remotePackage>
  <remotePackage path="cmdline-tools;20.0">
    <revision><major>20</major><minor>0</minor></revision>
  </remotePackage>
  <remotePackage path="cmake;4.1.2">
    <revision><major>4</major><minor>1</minor><micro>2</micro></revision>
  </remotePackage>
</sdk-repository>`

	repo, err := ParseGoogleRepository([]byte(xmlData))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		pkg  string
		kind PackageKind
		want string
	}{
		{pkg: "platforms", kind: KindPlatform, want: "37.0"},
		{pkg: "build-tools", kind: KindVersionedPath, want: "37.0.0"},
		{pkg: "platform-tools", kind: KindRevision, want: "37.0.0"},
		{pkg: "emulator", kind: KindRevision, want: "36.6.9"},
		{pkg: "ndk", kind: KindVersionedPath, want: "29.0.14206865"},
		{pkg: "cmdline-tools", kind: KindVersionedPath, want: "20.0"},
		{pkg: "cmake", kind: KindVersionedPath, want: "4.1.2"},
	}

	for _, tt := range tests {
		if got := repo.Latest(tt.pkg, tt.kind); got != tt.want {
			t.Fatalf("repo.Latest(%q) = %q, want %q", tt.pkg, got, tt.want)
		}
	}
}
