package main

import (
	"strconv"
	"strings"
	"unicode"
)

func IsPreviewVersion(version string) bool {
	lower := strings.ToLower(version)
	return strings.Contains(lower, "preview") ||
		strings.Contains(lower, "rc") ||
		strings.Contains(lower, "alpha") ||
		strings.Contains(lower, "beta")
}

func IsNumericVersion(version string) bool {
	parsed := ParseVersion(version)
	return parsed.Valid
}

func IsStableNumericVersion(version string) bool {
	return IsNumericVersion(version) && !IsPreviewVersion(version)
}

type ParsedVersion struct {
	Numbers []int
	Suffix  string
	Valid   bool
}

func ParseVersion(version string) ParsedVersion {
	version = strings.TrimSpace(version)
	if version == "" {
		return ParsedVersion{}
	}

	base := version
	suffix := ""
	if before, after, found := strings.Cut(version, "-"); found {
		base = before
		suffix = after
		if suffix == "" {
			return ParsedVersion{}
		}
		for _, r := range suffix {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_') {
				return ParsedVersion{}
			}
		}
	}

	parts := strings.Split(base, ".")
	if len(parts) == 0 {
		return ParsedVersion{}
	}

	numbers := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return ParsedVersion{}
		}
		for _, r := range part {
			if !unicode.IsDigit(r) {
				return ParsedVersion{}
			}
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return ParsedVersion{}
		}
		numbers = append(numbers, n)
	}

	return ParsedVersion{Numbers: numbers, Suffix: suffix, Valid: true}
}

func CompareVersions(a, b string) int {
	pa := ParseVersion(a)
	pb := ParseVersion(b)
	if !pa.Valid && !pb.Valid {
		return strings.Compare(a, b)
	}
	if !pa.Valid {
		return -1
	}
	if !pb.Valid {
		return 1
	}

	maxLen := len(pa.Numbers)
	if len(pb.Numbers) > maxLen {
		maxLen = len(pb.Numbers)
	}

	for i := 0; i < maxLen; i++ {
		av := 0
		if i < len(pa.Numbers) {
			av = pa.Numbers[i]
		}
		bv := 0
		if i < len(pb.Numbers) {
			bv = pb.Numbers[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}

	// For the same numeric version, prefer stable releases over preview/RC
	// suffixes. This makes 37.0.0 newer than 37.0.0-rc2.
	if pa.Suffix == "" && pb.Suffix != "" {
		return 1
	}
	if pa.Suffix != "" && pb.Suffix == "" {
		return -1
	}
	return strings.Compare(strings.ToLower(pa.Suffix), strings.ToLower(pb.Suffix))
}

func VersionGreater(a, b string) bool {
	if a == "" || b == "" || a == b {
		return false
	}
	return CompareVersions(a, b) > 0
}

func LatestVersion(versions []string, stableOnly bool) string {
	latest := ""
	for _, version := range versions {
		version = strings.TrimSpace(version)
		if version == "" || !IsNumericVersion(version) {
			continue
		}
		if stableOnly && IsPreviewVersion(version) {
			continue
		}
		if latest == "" || CompareVersions(version, latest) > 0 {
			latest = version
		}
	}
	return latest
}

func ShouldSuggestUpdate(current, latest string) bool {
	if current == "" || latest == "" || current == latest {
		return false
	}
	if IsPreviewVersion(current) {
		return true
	}
	cmp := CompareVersions(latest, current)
	return cmp > 0 || cmp == 0
}
