package db

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/tardisx/linkwallet/entity"
)

var bmm *BookmarkManager
var corporaDB string = "/tmp/corpora.db"

func createCorporaIfNecessary() {
	_, err := os.Stat(corporaDB)
	if err != nil {
		log.Printf("creating corpora")
		dbh := DB{}
		dbh.Open(corporaDB)
		bmm := NewBookmarkManager(&dbh)
		importCorpora(*bmm)
		dbh.Close()
		log.Printf("finished creating corpora")

	}
}

func newCorpusTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id := 0
		n, _ := fmt.Sscanf(r.URL.Path, "/%d", &id)
		if n != 1 {
			panic("bad req")
		}
		w.Header().Set("Content-Type", "text/html")
		f, err := os.Open(fmt.Sprintf("../content/corpora/%d.html", id))
		if err != nil {
			panic(err)
		}
		io.Copy(w, f)
	})
	return httptest.NewServer(mux)
}

func importCorpora(bmm BookmarkManager) {
	ts := newCorpusTestServer()
	defer ts.Close()

	for i := 1; i <= 100; i++ {
		url := fmt.Sprintf("%s/%d", ts.URL, i)
		bm := entity.Bookmark{URL: url}
		bmm.AddBookmark(&bm)
		bmm.ScrapeAndIndex(&bm)
	}

}

func createDBAndImportCorpora() *BookmarkManager {

	return bmm
}

func BenchmarkOneWordSearch(b *testing.B) {
	createCorporaIfNecessary()
	dbh := DB{}
	dbh.Open(corporaDB)
	bmm := NewBookmarkManager(&dbh)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bmm.Search(SearchOptions{Query: "hello"})
	}
}

func BenchmarkTwoWordSearch(b *testing.B) {
	createCorporaIfNecessary()
	dbh := DB{}
	dbh.Open(corporaDB)
	bmm := NewBookmarkManager(&dbh)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bmm.Search(SearchOptions{Query: "human relate"})
	}
}

func BenchmarkThreeWordSearch(b *testing.B) {
	createCorporaIfNecessary()
	dbh := DB{}
	dbh.Open(corporaDB)
	bmm := NewBookmarkManager(&dbh)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bmm.Search(SearchOptions{Query: "human wiki editor"})
	}
}
