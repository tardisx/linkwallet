package entity

import (
	"fmt"
	"sort"
	"time"
)

type DBStats struct {
	History  map[time.Time]BookmarkInfo
	FileSize int
	Searches int
}

type BookmarkInfo struct {
	Bookmarks    int
	IndexedWords int
}

func (stats DBStats) String() string {
	out := fmt.Sprintf("searches: %d\n", stats.Searches)

	dates := []time.Time{}

	for k := range stats.History {
		dates = append(dates, k)
	}

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	for _, k := range dates {
		out += fmt.Sprintf("%s - %d bookmarks, %d words indexed\n", k, stats.History[k].Bookmarks, stats.History[k].IndexedWords)
	}
	return out
}

func (stats DBStats) MostRecentBookmarkInfo() BookmarkInfo {
	mostRecent := time.Time{}
	for k := range stats.History {
		if k.After(mostRecent) {
			mostRecent = k
		}
	}
	if !mostRecent.IsZero() {
		return stats.History[mostRecent]
	}
	return BookmarkInfo{}
}
