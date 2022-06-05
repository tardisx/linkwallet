package version

import (
	"context"
	"sync"

	"github.com/google/go-github/v44/github"
	"golang.org/x/mod/semver"
)

const Tag = "v0.0.17"

var versionInfo struct {
	Local struct {
		Tag string
	}
	Remote struct {
		Valid bool
		Tag   string
	}
	m sync.Mutex
}

func init() {
	versionInfo.Remote.Valid = false
	versionInfo.Local.Tag = Tag
}

func Is() string {
	return versionInfo.Local.Tag
}

func UpgradeAvailable() (bool, string) {
	versionInfo.m.Lock()
	defer versionInfo.m.Unlock()
	if !versionInfo.Remote.Valid {
		return false, ""
	}
	if semver.Compare(versionInfo.Local.Tag, versionInfo.Remote.Tag) < 0 {
		return true, versionInfo.Remote.Tag
	}
	return false, ""

}

func UpdateVersionInfo() {
	client := github.NewClient(nil)

	rels, _, err := client.Repositories.ListReleases(context.Background(), "tardisx", "linkwallet", nil)
	if err != nil {
		panic(err)
	}
	if len(rels) == 0 {
		return
	}
	versionInfo.m.Lock()
	versionInfo.Remote.Tag = *rels[0].TagName
	versionInfo.Remote.Valid = true
	versionInfo.m.Unlock()

}
