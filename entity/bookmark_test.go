package entity

import (
	"testing"
)

func TestTitle(t *testing.T) {
	bm := Bookmark{
		URL: "http://example.org",
		Info: PageInfo{
			Title: "",
		},
	}
	if bm.DisplayTitle() != "http://example.org" {
		t.Errorf("title incorrect - got %s", bm.DisplayTitle())
	}
	bm.Info.Title = "Example Site"
	if bm.DisplayTitle() != "Example Site" {
		t.Errorf("title incorrect - got %s", bm.DisplayTitle())
	}

}
