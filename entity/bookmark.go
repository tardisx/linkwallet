package entity

import (
	"html/template"
	"strings"
	"time"
)

type Bookmark struct {
	ID                   uint64 `boltholdKey:"ID"`
	URL                  string
	Info                 PageInfo
	Tags                 []string
	PreserveTitle        bool
	TimestampCreated     time.Time
	TimestampLastScraped time.Time
}

func (bm Bookmark) Type() string {
	return "bookmark"
}

func (bm Bookmark) DisplayTitle() string {
	if strings.TrimSpace(bm.Info.Title) == "" {
		return bm.URL
	}
	return bm.Info.Title
}

type PageInfo struct {
	Fetched    time.Time
	Title      string
	Size       int
	StatusCode int
	RawText    string
}

func (pi PageInfo) Type() string {
	return "info"
}

type BookmarkSearchResult struct {
	Bookmark  Bookmark
	Score     float64
	Highlight template.HTML
}
