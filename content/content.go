package content

import (
	"log"
	"time"

	"github.com/tardisx/linkwallet/entity"

	"github.com/gocolly/colly"
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
	})

	c.OnRequest(func(r *colly.Request) {
		// log.Println("Visiting", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("error for %s: %s", r.Request.URL.String(), err)
	})

	c.Visit(url)
	return info
}
