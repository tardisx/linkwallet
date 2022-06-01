package content

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tardisx/linkwallet/entity"
)

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
</head>
<body>
<h1>Hello World</h1>
<p class="description">This is a test page</p>
<p class="description">This is a test paragraph</p>
</body>
</html>
		`))
	})
	return httptest.NewServer(mux)
}

func TestSimpleScrape(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	bm := entity.Bookmark{URL: ts.URL}
	info := FetchPageInfo(bm)

	if info.Title != "Test Page" {
		t.Errorf("'%s' is wrong title (expected Test Page)", info.Title)
	}
	if info.Size != 208 {
		t.Errorf("expected 208 bytes, got %d", info.Size)
	}
	if info.StatusCode != 200 {
		t.Errorf("got status code %d not 200", info.StatusCode)
	}
}

func TestWords(t *testing.T) {

	bm := entity.Bookmark{
		//		ID:                   0,
		//		URL:                  "",
		Info: entity.PageInfo{RawText: "the quick brown fox jumped over the lazy dog"},
		//		Tags:                 []string{},
	}
	words := Words(&bm)
	if len(words) != 7 {
		t.Errorf("got %d words not 7", len(words))
	} else {
		if words[0] != "quick" ||
			words[1] != "brown" ||
			words[2] != "fox" ||
			words[3] != "jump" ||
			words[4] != "over" ||
			words[5] != "lazi" ||
			words[6] != "dog" {
			t.Error("incorrect words returned")
		}

	}

}
