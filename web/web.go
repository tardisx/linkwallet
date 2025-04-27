package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tardisx/linkwallet/db"
	"github.com/tardisx/linkwallet/entity"
	"github.com/tardisx/linkwallet/meta"
	"github.com/tardisx/linkwallet/version"

	"github.com/gomarkdown/markdown"
	"github.com/hako/durafmt"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

// Server represents a SCUD web server.
// The SCUD web service can serve 2 different kinda of responses. The first is basic static
// vendor-provided files (called "assetFiles" here). An arbitrary number of them can be placed
// in assets/ and served up via a path prefix of /assets. They do not need individual routes
// to be specified.
// The second is htmx responses fragments. We never automatically serve templates (ie no mapping
// from template name to a URL route), there will always be a specific route or routes which
// use one or more templates to return a response.
type Server struct {
	engine *gin.Engine
	templ  *template.Template
	bmm    *db.BookmarkManager
}

type ColumnInfo struct {
	Name   string
	Param  string
	Sorted string
	Class  string
}

func (c ColumnInfo) URLString() string {
	if c.Sorted == "asc" {
		return "-" + c.Param
	}
	return c.Param
}

func (c ColumnInfo) TitleArrow() string {
	if c.Sorted == "asc" {
		return "↑"
	} else if c.Sorted == "desc" {
		return "↓"
	}
	return ""
}

// Create creates a new web server instance and sets up routing.
func Create(bmm *db.BookmarkManager, cmm *db.ConfigManager) *Server {

	// Set the default font for graphs
	plot.DefaultFont = font.Font{
		Typeface: "Liberation",
		Variant:  "Mono",
	}

	// setup routes for the static assets (vendor includes)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("problem with assetFS: %s", err)
	}

	// templ := template.Must(template.New("").Funcs(template.FuncMap{"dict": dictHelper}).ParseFS(templateFiles, "templates/*.html"))
	templ := template.Must(template.New("").Funcs(
		template.FuncMap{
			"nicetime":   niceTime,
			"niceURL":    niceURL,
			"niceSizeMB": func(s int) string { return fmt.Sprintf("%.1f", float32(s)/1024/1024) },
			"join":       strings.Join,
			"version":    func() *version.Info { return &version.VersionInfo },
			"meminfo":    meta.MemInfo,
			"markdown":   func(s string) template.HTML { return template.HTML(string(markdown.ToHTML([]byte(s), nil, nil))) },
		}).ParseFS(templateFiles, "templates/*.html"))

	config, err := cmm.LoadConfig()
	if err != nil {
		log.Fatalf("could not start server - failed to load config: %s", err)
	}

	r := gin.Default()

	server := &Server{
		engine: r,
		templ:  templ,
		bmm:    bmm,
	}

	r.Use(headersByURI())
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedExtensions([]string{".pdf", ".mp4"})))

	r.SetHTMLTemplate(templ)
	r.StaticFS("/assets", http.FS(staticFS))

	r.GET("/", func(c *gin.Context) {
		meta := gin.H{"page": "root", "config": config}
		c.HTML(http.StatusOK,
			"_layout.html", meta,
		)
	})

	r.GET("/manage", func(c *gin.Context) {

		allBookmarks, _ := bmm.ListBookmarks()
		meta := gin.H{"page": "manage", "config": config, "bookmarks": allBookmarks}
		c.HTML(http.StatusOK,
			"_layout.html", meta,
		)
	})

	r.POST("/manage/results", func(c *gin.Context) {
		query := c.PostForm("query")
		sort := c.Query("sort")

		bookmarks := []entity.Bookmark{}
		if query == "" {
			bookmarks, _ = bmm.ListBookmarks()
		} else {
			bookmarks, _ = bmm.Search(db.SearchOptions{Query: query, Sort: sort})
		}
		meta := gin.H{"config": config, "bookmarks": bookmarks}

		colTitle := &ColumnInfo{Name: "Title/URL", Param: "title"}
		colCreated := &ColumnInfo{Name: "Created", Param: "created", Class: "show-for-large"}
		colScraped := &ColumnInfo{Name: "Scraped", Param: "scraped", Class: "show-for-large"}
		if sort == "title" {
			colTitle.Sorted = "asc"
		}
		if sort == "-title" {
			colTitle.Sorted = "desc"
		}
		if sort == "scraped" {
			colScraped.Sorted = "asc"
		}
		if sort == "-scraped" {
			colScraped.Sorted = "desc"
		}
		if sort == "created" {
			colCreated.Sorted = "asc"
		}
		if sort == "-created" {
			colCreated.Sorted = "desc"
		}

		cols := gin.H{
			"title":   colTitle,
			"created": colCreated,
			"scraped": colScraped,
		}
		meta["column"] = cols

		c.HTML(http.StatusOK,
			"manage_results.html", meta,
		)

	})

	r.GET("/config", func(c *gin.Context) {
		meta := gin.H{"page": "config", "config": config}
		c.HTML(http.StatusOK,
			"_layout.html", meta,
		)
	})

	r.POST("/config", func(c *gin.Context) {
		config.BaseURL = c.PostForm("baseurl")
		config.BaseURL = strings.TrimRight(config.BaseURL, "/")
		cmm.SaveConfig(&config)
		meta := gin.H{"config": config}

		c.HTML(http.StatusOK, "config_form.html", meta)
	})

	r.POST("/search", func(c *gin.Context) {
		query := c.PostForm("query")

		// no query, return an empty response
		if len(query) == 0 {
			c.Status(http.StatusNoContent)
			c.Writer.Write([]byte{})
			return
		}

		sr, err := bmm.Search(db.SearchOptions{Query: query})
		data := gin.H{
			"results": sr,
			"error":   err,
		}

		c.HTML(http.StatusOK,
			"search_results.html", data,
		)
	})

	r.POST("/add", func(c *gin.Context) {
		url := c.PostForm("url")
		tags := []string{}
		if c.PostForm("tags_hidden") != "" {
			tags = strings.Split(c.PostForm("tags_hidden"), "|")
		}
		bm := entity.Bookmark{
			ID:   0,
			URL:  url,
			Tags: tags,
		}
		err := bmm.AddBookmark(&bm)

		data := gin.H{
			"bm":    bm,
			"error": err,
		}

		if err != nil {
			data["url"] = url
			data["tags"] = tags
			data["tags_hidden"] = c.PostForm("tags_hidden")
		}

		c.HTML(http.StatusOK, "add_url_form.html", data)
	})

	r.POST("/add_bulk", func(c *gin.Context) {
		urls := c.PostForm("urls")

		urlsSplit := strings.Split(urls, "\n")
		urlsTrimmed := make([]string, 0, 0)
		for _, url := range urlsSplit {
			urlsTrimmed = append(urlsTrimmed, strings.TrimSpace(url))
		}
		totalErrors := make([]string, 0, 0)
		added := 0
		for _, url := range urlsTrimmed {
			if url != "" {
				bm := entity.Bookmark{
					ID:  0,
					URL: url,
				}

				err := bmm.AddBookmark(&bm)
				if err != nil {
					totalErrors = append(totalErrors, fmt.Sprintf("url: %s (%s)", url, err.Error()))
				} else {
					added++
				}
			}
		}

		data := gin.H{
			"added":  added,
			"errors": totalErrors,
		}
		c.HTML(http.StatusOK, "add_url_form_bulk.html", data)
	})

	r.GET("/bulk_add", func(c *gin.Context) {
		c.HTML(http.StatusOK, "add_url_form_bulk.html", nil)
	})

	r.POST("/tags", func(c *gin.Context) {

		newTag := c.PostForm("tag") // new tag
		oldTags := strings.Split(c.PostForm("tags_hidden"), "|")

		remove := c.Query("remove")
		if remove != "" {
			log.Printf("removing %s", remove)
			trimmedTags := []string{}
			for _, k := range oldTags {
				if k != remove {
					trimmedTags = append(trimmedTags, k)
				}
			}
			oldTags = trimmedTags
		}

		tags := append(oldTags, newTag)
		tags = cleanupTags(tags)
		tagsHidden := strings.Join(tags, "|")

		data := gin.H{"tags": tags, "tags_hidden": tagsHidden}
		c.HTML(http.StatusOK, "tags_widget.html", data)
	})

	r.GET("/single_add", func(c *gin.Context) {
		c.HTML(http.StatusOK, "add_url_form.html", nil)
	})

	// XXX this should properly replace the button
	r.POST("/scrape/:id", func(c *gin.Context) {
		id := c.Params.ByName("id")
		idNum, _ := strconv.ParseInt(id, 10, 32)
		bm := bmm.LoadBookmarkByID(uint64(idNum))
		bmm.QueueScrape(&bm)
		c.String(http.StatusOK, "<p>scrape queued</p>")
	})

	r.GET("/export", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/plain")
		c.Writer.Header().Set("Content-Disposition", "attachment; filename=\"bookmarks.txt\"")
		err := bmm.ExportBookmarks(c.Writer)
		// this is a bit late, but we already added headers, so at least log it.
		if err != nil {
			log.Printf("got error when exporting: %s", err)
		}
	})

	r.GET("/bookmarklet", func(c *gin.Context) {
		url := c.Query("url")

		meta := gin.H{"page": "bookmarklet_click", "config": config, "url": url}

		// check if they just clicked it from the actual app
		if strings.Index(url, config.BaseURL) == 0 {
			meta["clicked"] = true
		}

		c.HTML(http.StatusOK,
			"_layout.html", meta,
		)
	})

	r.GET("/edit/:id", func(c *gin.Context) {
		bookmarkIDstring := c.Param("id")
		bookmarkID, ok := strconv.ParseUint(bookmarkIDstring, 10, 64)
		if ok != nil {
			c.String(http.StatusBadRequest, "bad id")
			return
		}

		bookmark := bmm.LoadBookmarkByID(bookmarkID)
		meta := gin.H{"page": "edit", "bookmark": bookmark, "tw": gin.H{"tags": bookmark.Tags, "tags_hidden": strings.Join(bookmark.Tags, "|")}}

		c.HTML(http.StatusOK,
			"_layout.html", meta,
		)
	})

	r.POST("/edit/:id", func(c *gin.Context) {
		bookmarkIDstring := c.Param("id")
		bookmarkID, ok := strconv.ParseUint(bookmarkIDstring, 10, 64)
		if ok != nil {
			c.String(http.StatusBadRequest, "bad id")
			return
		}

		bookmark := bmm.LoadBookmarkByID(bookmarkID)

		// update title and override title
		overrideTitle := c.PostForm("override_title")
		if overrideTitle != "" {
			title := c.PostForm("title")
			bookmark.Info.Title = title
			bookmark.PreserveTitle = true
		} else {
			bookmark.PreserveTitle = false
		}

		// freshen tags
		if c.PostForm("tags_hidden") == "" {
			// empty
			bookmark.Tags = []string{}
		} else {
			bookmark.Tags = strings.Split(c.PostForm("tags_hidden"), "|")
		}

		bmm.SaveBookmark(&bookmark)
		bmm.UpdateIndexForBookmark(&bookmark) // because title may have changed

		meta := gin.H{"page": "edit", "bookmark": bookmark, "saved": true, "tw": gin.H{"tags": bookmark.Tags, "tags_hidden": strings.Join(bookmark.Tags, "|")}}

		c.HTML(http.StatusOK,
			"edit_form.html", meta,
		)
	})

	r.DELETE("/edit/:id", func(c *gin.Context) {
		bookmarkIDstring := c.Param("id")
		bookmarkID, ok := strconv.ParseUint(bookmarkIDstring, 10, 64)
		if ok != nil {
			c.String(http.StatusBadRequest, "bad id")
			return
		}

		bookmark := bmm.LoadBookmarkByID(bookmarkID)
		err := bmm.DeleteBookmark(&bookmark)
		if err != nil {
			panic(err)
		}
		c.HTML(http.StatusOK,
			"edit_form_deleted.html", nil,
		)
	})

	r.GET("/info", func(c *gin.Context) {
		dbStats, err := bmm.Stats()
		if err != nil {
			panic("could not load stats for info page")
		}
		meta := gin.H{"page": "info", "stats": dbStats, "config": config}
		c.HTML(http.StatusOK,
			"_layout.html", meta,
		)
	})

	r.GET("/graph/:type", func(c *gin.Context) {

		graphType := c.Param("type")
		p := plot.New()

		dbStats, err := bmm.Stats()
		if err != nil {
			panic("could not load stats for graph page")
		}

		sortedKeys := make([]time.Time, 0)
		for k := range dbStats.History {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Slice(sortedKeys, func(i, j int) bool {
			return sortedKeys[i].Before(sortedKeys[j])
		})

		xTicks := plot.TimeTicks{Format: "2006-01-02"}
		p.X.Tick.Marker = xTicks

		plotPoints(sortedKeys, dbStats, p, graphType)

		writerTo, err := p.WriterTo(vg.Points(640), vg.Points(480), "png")
		if err != nil {
			panic("error creating WriterTo: " + err.Error())
		}

		c.Header("Content-Type", "image/png")
		writerTo.WriteTo(c.Writer)

	})

	return server
}

func plotPoints(sortedKeys []time.Time, dbStats entity.DBStats, p *plot.Plot, k string) {

	if k == "indexed_words" {
		p.Title.Text = "Indexed words over time"
		p.Y.Label.Text = "Words indexed"
	} else if k == "bookmarks" {
		p.Title.Text = "Bookmarks over time"
		p.Y.Label.Text = "Bookmarks"
	} else {
		panic("bad k")
	}
	p.X.Label.Text = "Date"

	pts := make(plotter.XYs, len(sortedKeys))
	for i := range sortedKeys {
		pts[i].X = float64(sortedKeys[i].Unix())
		if k == "indexed_words" {
			pts[i].Y = float64(dbStats.History[sortedKeys[i]].IndexedWords)
		} else if k == "bookmarks" {
			pts[i].Y = float64(dbStats.History[sortedKeys[i]].Bookmarks)
		} else {
			panic("bad key")
		}
	}

	l, err := plotter.NewLine(pts)
	if err != nil {
		panic(err)
	}
	p.Add(l)
}

// headersByURI sets the headers for some special cases, set a custom long cache time for
// static resources.
func headersByURI() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.String(), "/assets/") {
			c.Header("Cache-Control", "max-age=86400")
			c.Header("Expires", time.Now().Add(time.Hour*24*1).Format("Mon 2 Jan 2006 15:04:05 MST"))
		}
	}
}

// Start starts the web server, blocking forever.
func (s *Server) Start() {
	s.engine.Run()
}

func cleanupTags(tags []string) []string {
	keys := make(map[string]struct{})
	for _, k := range tags {
		if k != "" && k != "|" {
			for _, subKey := range strings.Split(k, ",") {
				subKey := strings.Trim(subKey, " ")
				keys[strings.ToLower(subKey)] = struct{}{}
			}
		}
	}
	out := []string{}
	for k := range keys {
		out = append(out, k)
	}

	sort.Strings(out)
	return out
}

type timeVariations struct {
	HumanDuration string
}

func niceTime(t time.Time) timeVariations {

	u := "y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,us:us"
	units, err := durafmt.DefaultUnitsCoder.Decode(u)
	if err != nil {
		panic(err)
	}
	ago := durafmt.Parse(time.Since(t)).LimitFirstN(1).Format(units)
	ago = strings.ReplaceAll(ago, " ", "")

	return timeVariations{HumanDuration: ago}
}

func niceURL(url string) string {
	if len(url) > 50 {
		return url[0:50] + " ..."
	}
	return url
}
