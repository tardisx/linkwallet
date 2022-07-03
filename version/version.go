package version

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/go-github/v44/github"
	"golang.org/x/mod/semver"
)

const Tag = "v0.0.28"

type Info struct {
	Local struct {
		Tag string
	}
	Remote struct {
		Valid bool
		Tag   string
	}
	UpgradeReleaseNotes string
	m                   sync.Mutex
}

var VersionInfo Info

func init() {
	VersionInfo.Remote.Valid = false
	VersionInfo.Local.Tag = Tag
}

func (vi *Info) UpgradeAvailable() bool {
	vi.m.Lock()
	defer vi.m.Unlock()
	if !vi.Remote.Valid {
		return false
	}
	if semver.Compare(vi.Local.Tag, vi.Remote.Tag) < 0 {
		return true
	}
	return false
}

func (vi *Info) UpdateVersionInfo() {
	client := github.NewClient(nil)

	rels, _, err := client.Repositories.ListReleases(context.Background(), "tardisx", "linkwallet", nil)
	if err != nil {
		return
	}
	if len(rels) == 0 {
		return
	}

	vi.m.Lock()
	vi.Remote.Tag = *rels[0].TagName
	vi.Remote.Valid = true
	vi.UpgradeReleaseNotes = ""
	for _, r := range rels {
		if semver.Compare(VersionInfo.Local.Tag, *r.TagName) < 0 {
			vi.UpgradeReleaseNotes += fmt.Sprintf("*Version %s*\n\n", *r.TagName)
			bodyLines := strings.Split(*r.Body, "\n")
			for _, l := range bodyLines {
				if strings.Index(l, "#") == 0 && strings.Contains(l, "Changelog") {
					// do nothing, ignore the changelog heading
				} else {
					vi.UpgradeReleaseNotes += l + "\n"
				}
			}
		}
	}

	vi.m.Unlock()

}
