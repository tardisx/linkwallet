package version

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/google/go-github/v44/github"
	"golang.org/x/mod/semver"
)

var version string // populated by goreleaser, without leading 'v'
var commit string
var date string

var VersionInfo Info

func init() {
	VersionInfo.Remote.Valid = false
	VersionInfo.Local.Version = "v" + version
}

type Info struct {
	Local struct {
		Version string
	}
	Remote struct {
		Valid bool
		Tag   string
	}
	UpgradeReleaseNotes string
	m                   sync.Mutex
}

func (vi *Info) UpgradeAvailable() bool {
	vi.m.Lock()
	defer vi.m.Unlock()
	if !vi.Remote.Valid {
		return false
	}

	log.Printf("checking if upgrade available - local %s remote %s", vi.Local.Version, vi.Remote.Tag)
	localValid := semver.IsValid(vi.Local.Version)
	remoteValid := semver.IsValid(vi.Remote.Tag)

	if !localValid {
		log.Printf("version %s invalid", vi.Local.Version)
	}
	if !remoteValid {
		log.Printf("version %s invalid", vi.Remote.Tag)
	}

	if !localValid || !remoteValid {
		return false
	}

	return semver.Compare(vi.Local.Version, vi.Remote.Tag) < 0
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
		if semver.Compare(VersionInfo.Local.Version, *r.TagName) < 0 {
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
