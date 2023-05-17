package content

import (
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/tardisx/linkwallet/entity"

	"github.com/gocolly/colly"
	snowballeng "github.com/kljensen/snowball/english"
)

func FetchPageInfo(bm entity.Bookmark) entity.PageInfo {
	info := entity.PageInfo{
		Fetched: time.Now(),
	}

	url := bm.URL

	c := colly.NewCollector()
	c.SetRequestTimeout(5 * time.Second)

	c.OnHTML("p,h1,h2,h3,h4,h5,h6,li", func(e *colly.HTMLElement) {
		info.RawText = info.RawText + e.Text + "\n"
	})

	c.OnHTML("head>title", func(h *colly.HTMLElement) {
		info.Title = h.Text
	})

	c.OnResponse(func(r *colly.Response) {
		info.StatusCode = r.StatusCode
		info.Size = len(r.Body)
		//	log.Printf("content type for %s: %s (%d)", r.Request.URL.String(), r.Headers.Get("Content-Type"), info.Size)

	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		// log.Println("Visiting", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("error for %s: %s", r.Request.URL.String(), err)
	})

	c.Visit(url)
	return info
}

func Words(bm *entity.Bookmark) []string {
	words := []string{}

	words = append(words, StringToStemmedSearchWords(bm.Info.RawText)...)
	words = append(words, StringToStemmedSearchWords(bm.Info.Title)...)
	words = append(words, StringToStemmedSearchWords(bm.URL)...)
	return words
}

// StringToStemmedSearchWords returns a list of stemmed words with stop words
// removed.
func StringToStemmedSearchWords(s string) []string {
	words := []string{}

	words = append(words, stemmerFilter(stopwordFilter(tokenize(s)))...)
	return words
}

func tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		// Split on any character that is not a letter or a number.
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func stemmerFilter(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = snowballeng.Stem(token, false)
	}
	return r
}

var stopwords = map[string]struct{}{ // I wish Go had built-in sets.
	"a": {}, "and": {}, "be": {}, "have": {}, "i": {},
	"in": {}, "of": {}, "that": {}, "the": {}, "to": {},
}

func stopwordFilter(tokens []string) []string {
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := stopwords[token]; !ok {
			r = append(r, token)
		}
	}
	return r
}
