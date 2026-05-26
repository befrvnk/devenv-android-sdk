package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultGoogleRepositoryURL = "https://dl.google.com/android/repository/repository2-3.xml"

type PackageKind int

const (
	KindPlatform PackageKind = iota
	KindVersionedPath
	KindRevision
)

type GoogleRepository struct {
	RemotePackages []RemotePackage `xml:"remotePackage"`
}

type RemotePackage struct {
	Path     string   `xml:"path,attr"`
	Revision Revision `xml:"revision"`
}

type Revision struct {
	Major   *int `xml:"major"`
	Minor   *int `xml:"minor"`
	Micro   *int `xml:"micro"`
	Preview *int `xml:"preview"`
}

func (r Revision) Version() string {
	if r.Major == nil {
		return ""
	}

	parts := []string{fmt.Sprintf("%d", *r.Major)}
	if r.Minor != nil {
		parts = append(parts, fmt.Sprintf("%d", *r.Minor))
	}
	if r.Micro != nil {
		parts = append(parts, fmt.Sprintf("%d", *r.Micro))
	}
	return strings.Join(parts, ".")
}

func (r Revision) IsPreview() bool {
	return r.Preview != nil
}

func ParseGoogleRepository(data []byte) (GoogleRepository, error) {
	var repo GoogleRepository
	if err := xml.Unmarshal(data, &repo); err != nil {
		return GoogleRepository{}, err
	}
	return repo, nil
}

func FetchGoogleRepository(url string) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (g GoogleRepository) Latest(packageName string, kind PackageKind) string {
	versions := make([]string, 0)
	for _, remotePackage := range g.RemotePackages {
		if remotePackage.Revision.IsPreview() {
			continue
		}

		switch kind {
		case KindPlatform:
			const prefix = "platforms;android-"
			if strings.HasPrefix(remotePackage.Path, prefix) {
				versions = append(versions, strings.TrimPrefix(remotePackage.Path, prefix))
			}
		case KindVersionedPath:
			prefix := packageName + ";"
			if strings.HasPrefix(remotePackage.Path, prefix) {
				version := strings.TrimPrefix(remotePackage.Path, prefix)
				if !strings.EqualFold(version, "latest") {
					versions = append(versions, version)
				}
			}
		case KindRevision:
			if remotePackage.Path == packageName {
				versions = append(versions, remotePackage.Revision.Version())
			}
		}
	}
	return LatestVersion(versions, true)
}
