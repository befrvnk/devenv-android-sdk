package main

import "testing"

func TestLatestVersionSupportsDecimalPlatforms(t *testing.T) {
	versions := []string{"36", "36x", "Baklava", "37.0", "35"}
	got := LatestVersion(versions, true)
	if got != "37.0" {
		t.Fatalf("LatestVersion() = %q, want %q", got, "37.0")
	}
}

func TestLatestVersionFiltersPreviews(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		want     string
	}{
		{
			name:     "build tools rc",
			versions: []string{"36.1.0", "37.0.0-rc1", "37.0.0"},
			want:     "37.0.0",
		},
		{
			name:     "ndk rc higher than stable",
			versions: []string{"29.0.14206865", "30.0.14904198-rc1"},
			want:     "29.0.14206865",
		},
		{
			name:     "alpha beta preview",
			versions: []string{"20.0", "21.0-alpha01", "21.0-beta01", "21.0-preview1"},
			want:     "20.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LatestVersion(tt.versions, true)
			if got != tt.want {
				t.Fatalf("LatestVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{a: "37.0", b: "37", want: 0},
		{a: "37.0.0", b: "37.0.0-rc2", want: 1},
		{a: "29.0.14206865", b: "28.2.13676358", want: 1},
		{a: "3.31.6", b: "4.1.2", want: -1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if tt.want < 0 && got >= 0 || tt.want == 0 && got != 0 || tt.want > 0 && got <= 0 {
			t.Fatalf("CompareVersions(%q, %q) = %d, want sign %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestShouldSuggestUpdate(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "newer stable", current: "36", latest: "37.0", want: true},
		{name: "same numeric but metadata key differs", current: "37", latest: "37.0", want: true},
		{name: "same", current: "37.0", latest: "37.0", want: false},
		{name: "preview to stable", current: "37.0.0-rc2", latest: "37.0.0", want: true},
		{name: "ndk preview to latest stable", current: "30.0.14904198-rc1", latest: "29.0.14206865", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSuggestUpdate(tt.current, tt.latest)
			if got != tt.want {
				t.Fatalf("ShouldSuggestUpdate(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
