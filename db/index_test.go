package db

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"

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

	searchRes, err := bmm.Search(SearchOptions{Query: "fox"})
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

	searchRes, err = bmm.Search(SearchOptions{Query: "fox"})
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 0 {
		t.Error("got result when should not")
	}

	searchRes, err = bmm.Search(SearchOptions{Query: "rabbit"})
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

	searchRes, err = bmm.Search(SearchOptions{Query: "rabbit"})
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

	searchRes, err := bmm.Search(SearchOptions{Query: "fox"})
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
	searchRes, err = bmm.Search(SearchOptions{Query: "sloth"})
	if err != nil {
		t.Errorf("search returned %s", err)
	}
	if len(searchRes) != 1 {
		t.Error("did not get one id for sloth")
	}
}

func testBM() entity.Bookmark {
	return entity.Bookmark{
		ID:  1,
		URL: "https://one.com",
		Info: entity.PageInfo{
			Fetched:    time.Time{},
			Title:      "one web",
			Size:       200,
			StatusCode: 200,
			RawText:    "one web site is great for all humans",
		},
		Tags:                 []string{"hello", "big friends"},
		PreserveTitle:        false,
		TimestampCreated:     time.Time{},
		TimestampLastScraped: time.Time{},
	}
}

func TestMappings(t *testing.T) {
	mapping := createIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	bm := testBM()
	err = idx.Index("1", bm)
	if err != nil {
		panic(err)
	}

	type tc struct {
		query   query.Query
		expHits int
	}
	tcs := []tc{
		{query: bleve.NewMatchQuery("human"), expHits: 1},
		{query: bleve.NewMatchQuery("humanoid"), expHits: 0},
		{query: bleve.NewMatchQuery("hello"), expHits: 1},
		{query: bleve.NewMatchQuery("big"), expHits: 0},
		{query: bleve.NewMatchQuery("friends"), expHits: 0},
		{query: bleve.NewMatchQuery("big friend"), expHits: 0},
		{query: bleve.NewTermQuery("big friends"), expHits: 1},
		{query: bleve.NewMatchQuery("web great"), expHits: 1},
	}

	for i := range tcs {
		q := tcs[i].query

		sr, err := idx.Search(bleve.NewSearchRequest(q))
		if err != nil {
			t.Error(err)
		} else {
			if len(sr.Hits) != tcs[i].expHits {
				t.Errorf("wrong hits - expected %d got %d for %s", tcs[i].expHits, len(sr.Hits), tcs[i].query)
			}
		}
	}

}

func TestMappingsDisjunctionQuery(t *testing.T) {
	mapping := createIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	bm := testBM()
	err = idx.Index("1", bm)
	if err != nil {
		panic(err)
	}

	type tc struct {
		query   string
		expHits int
	}
	tcs := []tc{
		{query: "human", expHits: 1},
		{query: "humanoid", expHits: 0},
		{query: "hello", expHits: 1},
		{query: "big", expHits: 0},
		{query: "friends", expHits: 0},
		{query: "big friend", expHits: 0},
		{query: "big friends", expHits: 1},
		{query: "web great", expHits: 1},
	}

	for i := range tcs {
		q := tcs[i].query
		req := bleve.NewDisjunctionQuery(
			bleve.NewMatchQuery(q),
			bleve.NewTermQuery(q),
		)

		sr, err := idx.Search(bleve.NewSearchRequest(req))
		if err != nil {
			t.Error(err)
		} else {
			if len(sr.Hits) != tcs[i].expHits {
				t.Errorf("wrong hits - expected %d got %d for %s", tcs[i].expHits, len(sr.Hits), tcs[i].query)
			}
		}
	}

}
