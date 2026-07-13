package gradleintegration

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSyncCreatesMissingFiles(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, ".devenv", "android-sdk")
	assertLinkTarget(t, link, sdk)
	assertFileContent(t, filepath.Join(root, "local.properties"), "sdk.dir="+link+"\n")
}

func TestSyncPreservesUnrelatedProperties(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	properties := filepath.Join(root, "local.properties")
	initial := "# local settings\napi.token=secret-value\n\nplugin.metadata=keep-me\n"
	writeFile(t, properties, initial)

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, ".devenv", "android-sdk")
	assertFileContent(t, properties, initial+"sdk.dir="+link+"\n")
}

func TestSyncUpdatesOldSDKDirInPlace(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	properties := filepath.Join(root, "local.properties")
	writeFile(t, properties, "# before\nsdk.dir=/nix/store/old-sdk/libexec/android-sdk\nother=value\n")

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, ".devenv", "android-sdk")
	assertFileContent(t, properties, "# before\nsdk.dir="+link+"\nother=value\n")
}

func TestSyncAtomicallyUpdatesRegularFileAndPreservesPermissions(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	properties := filepath.Join(root, "local.properties")
	writeFile(t, properties, "sdk.dir=/old/sdk\n")
	if err := os.Chmod(properties, 0o640); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(properties)
	if err != nil {
		t.Fatal(err)
	}

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	after, err := os.Stat(properties)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("regular local.properties was updated in place instead of atomically replaced")
	}
	if after.Mode().Perm() != 0o640 {
		t.Fatalf("local.properties permissions changed: want 0640, got %04o", after.Mode().Perm())
	}
}

func TestSyncNormalizesDuplicateSDKDirEntries(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	properties := filepath.Join(root, "local.properties")
	writeFile(t, properties, "first=keep\nsdk.dir=/old/one\n# sdk.dir=/commented\n  sdk.dir = /old/two\nsdk.dir: /old/three\nlast=keep\n")

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, ".devenv", "android-sdk")
	assertFileContent(t, properties, "first=keep\nsdk.dir="+link+"\n# sdk.dir=/commented\nlast=keep\n")
}

func TestSyncPreservesLocalPropertiesSymlink(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	target := filepath.Join(root, "config", "local.properties")
	writeFile(t, target, "token=preserved\nsdk.dir=/old/sdk\n")
	properties := filepath.Join(root, "local.properties")
	if err := os.Symlink(target, properties); err != nil {
		t.Fatal(err)
	}
	targetBefore, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(properties)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("local.properties symlink was replaced")
	}
	targetAfter, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(targetBefore, targetAfter) {
		t.Fatal("local.properties symlink target was replaced instead of updated in place")
	}
	link := filepath.Join(root, ".devenv", "android-sdk")
	assertFileContent(t, target, "token=preserved\nsdk.dir="+link+"\n")
}

func TestSyncRejectsBrokenLocalPropertiesSymlink(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	properties := filepath.Join(root, "local.properties")
	missingTarget := filepath.Join(root, "missing-target")
	if err := os.Symlink(missingTarget, properties); err != nil {
		t.Fatal(err)
	}

	err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk")
	assertErrorContains(t, err, "broken symlink")
	assertLinkTarget(t, properties, missingTarget)
}

func TestSyncRejectsUnwritableLocalPropertiesTarget(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	target := filepath.Join(root, "config", "local.properties")
	writeFile(t, target, "token=still-preserved\n")
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, 0o644) })
	properties := filepath.Join(root, "local.properties")
	if err := os.Symlink(target, properties); err != nil {
		t.Fatal(err)
	}

	err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk")
	assertErrorContains(t, err, "not writable")
	assertLinkTarget(t, properties, target)
	assertFileContent(t, target, "token=still-preserved\n")
}

func TestSyncAllowsUnchangedReadOnlyLocalProperties(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}
	properties := filepath.Join(root, "local.properties")
	if err := os.Chmod(properties, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(properties, 0o644) })
	oldTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(properties, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatalf("unchanged read-only local.properties should not require a write: %v", err)
	}
	info, err := os.Stat(properties)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(oldTime) {
		t.Fatalf("unchanged read-only local.properties was rewritten: mtime = %s", info.ModTime())
	}
}

func TestSyncRejectsNonSymlinkAtSDKLinkPath(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	collision := filepath.Join(root, ".devenv", "android-sdk")
	writeFile(t, filepath.Join(collision, "keep"), "sentinel\n")

	err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk")
	assertErrorContains(t, err, "exists and is not a symlink")
	assertFileContent(t, filepath.Join(collision, "keep"), "sentinel\n")
	if _, statErr := os.Stat(filepath.Join(root, "local.properties")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("local.properties unexpectedly exists: %v", statErr)
	}
}

func TestReplaceSymlinkCleansTemporaryFileAfterFailure(t *testing.T) {
	parent := t.TempDir()
	link := filepath.Join(parent, "existing-directory")
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceSymlink(link, "/new/target"); err == nil {
		t.Fatal("expected replacing a directory to fail")
	}
	matches, err := filepath.Glob(filepath.Join(parent, ".devenv-android-sdk-link-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary SDK links were not cleaned up: %v", matches)
	}
}

func TestAtomicLocalPropertiesWriteCleansTemporaryFileAfterFailure(t *testing.T) {
	parent := t.TempDir()
	destination := filepath.Join(parent, "existing-directory")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeRegularFileAtomically(destination, []byte("content"), 0o644); err == nil {
		t.Fatal("expected replacing a directory to fail")
	}
	matches, err := filepath.Glob(filepath.Join(parent, ".local.properties-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary local.properties files were not cleaned up: %v", matches)
	}
}

func TestSyncLeavesCurrentSDKLinkUnchanged(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	link := filepath.Join(root, ".devenv", "android-sdk")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sdk, link); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	after, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("already-correct SDK symlink was replaced")
	}
}

func TestSyncRefreshesSDKLinkWithoutChangingProperty(t *testing.T) {
	root := t.TempDir()
	oldSDK := makeSDK(t, root, "old-sdk")
	newSDK := makeSDK(t, root, "new-sdk")

	if err := syncForTest(root, oldSDK, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}
	properties := filepath.Join(root, "local.properties")
	before := readFile(t, properties)

	if err := syncForTest(root, newSDK, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	assertLinkTarget(t, filepath.Join(root, ".devenv", "android-sdk"), newSDK)
	assertFileContent(t, properties, before)
}

func TestSyncHandlesPathsContainingSpaces(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "project with spaces")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	sdk := makeSDK(t, base, filepath.Join("sdk paths", "current sdk"))

	if err := syncForTest(root, sdk, filepath.Join("gradle config", "local.properties"), filepath.Join("stable sdk", "android sdk")); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "stable sdk", "android sdk")
	assertLinkTarget(t, link, sdk)
	assertFileContent(t, filepath.Join(root, "gradle config", "local.properties"), "sdk.dir="+link+"\n")
}

func TestSyncEscapesBackslashesInSDKDirProperty(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, `project\with-backslash`)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	sdk := makeSDK(t, base, "sdk")

	if err := syncForTest(root, sdk, "local.properties", `.devenv/android\sdk`); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, `.devenv/android\sdk`)
	assertLinkTarget(t, link, sdk)
	escapedLink := strings.ReplaceAll(link, `\`, `\\`)
	assertFileContent(t, filepath.Join(root, "local.properties"), "sdk.dir="+escapedLink+"\n")
}

func TestSyncSupportsAbsoluteConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	properties := filepath.Join(root, "absolute config", "local.properties")
	link := filepath.Join(root, "absolute links", "android-sdk")

	if err := syncForTest(root, sdk, properties, link); err != nil {
		t.Fatal(err)
	}

	assertLinkTarget(t, link, sdk)
	assertFileContent(t, properties, "sdk.dir="+link+"\n")
}

func TestSyncDoesNotRewriteUnchangedLocalProperties(t *testing.T) {
	root := t.TempDir()
	sdk := makeSDK(t, root, "sdk")
	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}
	properties := filepath.Join(root, "local.properties")
	oldTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(properties, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	if err := syncForTest(root, sdk, "local.properties", ".devenv/android-sdk"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(properties)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(oldTime) {
		t.Fatalf("unchanged local.properties was rewritten: mtime = %s", info.ModTime())
	}
}

func TestSynchronizeSDKDirPreservesLineEndingsAndMissingFinalNewline(t *testing.T) {
	current := []byte("first=keep\r\nsdk.dir=/old/sdk\r\nlast=keep")
	got := synchronizeSDKDir(current, "/stable/sdk")
	want := "first=keep\r\nsdk.dir=/stable/sdk\r\nlast=keep"
	if string(got) != want {
		t.Fatalf("unexpected content:\nwant %q\n got %q", want, got)
	}
}

func syncForTest(root, sdk, propertiesPath, linkPath string) error {
	return Sync(Config{
		ProjectRoot:         root,
		AndroidHome:         sdk,
		LocalPropertiesPath: propertiesPath,
		SDKLinkPath:         linkPath,
	})
}

func makeSDK(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	if actual := readFile(t, path); actual != expected {
		t.Fatalf("unexpected content in %s:\nwant %q\n got %q", path, expected, actual)
	}
}

func assertLinkTarget(t *testing.T, path, expected string) {
	t.Helper()
	actual, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("unexpected target for %s: want %q, got %q", path, expected, actual)
	}
}

func assertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", expected)
	}
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %q", expected, err)
	}
}
