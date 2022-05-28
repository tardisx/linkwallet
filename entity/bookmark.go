package entity

import "time"

type Bookmark struct {
	ID                   uint64 `boltholdKey:"ID"`
	URL                  string
	Info                 PageInfo
	Tags                 []string
	TimestampCreated     time.Time
	TimestampLastScraped time.Time
}

type PageInfo struct {
	Fetched    time.Time
	Title      string
	Size       int
	StatusCode int
	RawText    string
}
