package sdkversions

import (
	"encoding/json"
	"os"
	"sort"
)

type RepoMetadata struct {
	Packages map[string]map[string]json.RawMessage `json:"packages"`
}

func LoadRepoMetadata(path string) (RepoMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RepoMetadata{}, err
	}

	var repo RepoMetadata
	if err := json.Unmarshal(data, &repo); err != nil {
		return RepoMetadata{}, err
	}
	if repo.Packages == nil {
		repo.Packages = map[string]map[string]json.RawMessage{}
	}
	return repo, nil
}

func (r RepoMetadata) Versions(packageName string) []string {
	pkg := r.Packages[packageName]
	if pkg == nil {
		return nil
	}

	versions := make([]string, 0, len(pkg))
	for version := range pkg {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool {
		return CompareVersions(versions[i], versions[j]) < 0
	})
	return versions
}

func (r RepoMetadata) Latest(packageName string) string {
	return LatestVersion(r.Versions(packageName), true)
}

func (r RepoMetadata) HasVersion(packageName, version string) bool {
	pkg := r.Packages[packageName]
	if pkg == nil {
		return false
	}
	_, ok := pkg[version]
	return ok
}
