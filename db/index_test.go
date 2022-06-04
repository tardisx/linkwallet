package db

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/tardisx/linkwallet/entity"
)

var serverResponse string

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(serverResponse))
	})
	return httptest.NewServer(mux)
}

func TestAddRemove(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()
	serverResponse = "<p>the quick brown fox</p>"

	db := DB{}
	f, _ := os.CreateTemp("", "test_boltdb_*")
	f.Close()
	defer os.Remove(f.Name())
	db.Open(f.Name())

	bmm := NewBookmarkManager(&db)
	bm := entity.Bookmark{URL: ts.URL}

	err := bmm.AddBookmark(&bm)
	if err != nil {
		t.Fatalf("error adding: %s", err)
	}
	if bm.ID == 0 {
		t.Error("bookmark did not get an id")
	}
	err = bmm.ScrapeAndIndex(&bm)
	if err != nil {
		t.Errorf("scrape index returned %s", err)
	}

	searchRes, err := bmm.Search("fox")
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 1 {
		t.Error("did not get one id")
	}

	// change content, rescrape
	serverResponse = "<p>the quick brown rabbit</p>"
	err = bmm.ScrapeAndIndex(&bm)
	if err != nil {
		t.Errorf("scrape index returned %s", err)
	}

	searchRes, err = bmm.Search("fox")
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 0 {
		t.Error("got result when should not")
	}

	searchRes, err = bmm.Search("rabbit")
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 1 {
		t.Error("did not get result when should")
	}

	err = bmm.DeleteBookmark(&bm)
	if err != nil {
		t.Errorf("got error when deleting: %s", err)
	}

	searchRes, err = bmm.Search("rabbit")
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 0 {
		t.Error("rabbit should be gone from index")
	}

}

func TestTagIndexing(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()
	serverResponse = "<p>the quick brown fox</p>"

	db := DB{}
	f, _ := os.CreateTemp("", "test_boltdb_*")
	f.Close()
	defer os.Remove(f.Name())
	db.Open(f.Name())

	bmm := NewBookmarkManager(&db)
	bm := entity.Bookmark{URL: ts.URL}

	err := bmm.AddBookmark(&bm)
	if err != nil {
		t.Fatalf("error adding: %s", err)
	}
	if bm.ID == 0 {
		t.Error("bookmark did not get an id")
	}
	err = bmm.ScrapeAndIndex(&bm)
	if err != nil {
		t.Errorf("scrape index returned %s", err)
	}

	searchRes, err := bmm.Search("fox")
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 1 {
		t.Error("did not get one id")
	}

	// add a tag
	bm.Tags = []string{"sloth"}
	err = bmm.ScrapeAndIndex(&bm)
	if err != nil {
		t.Errorf("scrape index returned %s", err)
	}
	searchRes, err = bmm.Search("sloth")
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 1 {
		t.Error("did not get one id for sloth")
	}
}
