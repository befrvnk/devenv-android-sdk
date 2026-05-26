package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGoogleNewerGuidanceBundledMetadata(t *testing.T) {
	var out bytes.Buffer
	printGoogleNewerGuidance(&out, Config{MetadataWritable: false})
	text := out.String()

	if strings.Contains(text, "Run 'update-android-sdk-repo'") {
		t.Fatalf("bundled metadata guidance must not tell users to run update-android-sdk-repo:\n%s", text)
	}
	for _, want := range []string{
		"repoJsonWritablePath = null",
		"pinned android-sdk input",
		"Update this module's bundled repo.json upstream",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("bundled metadata guidance missing %q:\n%s", want, text)
		}
	}
}

func TestGoogleNewerGuidanceWritableMetadata(t *testing.T) {
	var out bytes.Buffer
	printGoogleNewerGuidance(&out, Config{MetadataWritable: true})
	text := out.String()

	for _, want := range []string{
		"project-local and writable",
		"Run 'update-android-sdk-repo'",
		"commit the changed repo.json",
		"reload your shell",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("writable metadata guidance missing %q:\n%s", want, text)
		}
	}
}
