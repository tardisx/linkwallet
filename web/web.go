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
	"github.com/tardisx/linkwallet/version"

	"github.com/hako/durafmt"

	"github.com/gin-gonic/gin"
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

// Create creates a new web server instance and sets up routing.
func Create(bmm *db.BookmarkManager, cmm *db.ConfigManager) *Server {

	// setup routes for the static assets (vendor includes)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("problem with assetFS: %s", err)
	}

	// templ := template.Must(template.New("").Funcs(template.FuncMap{"dict": dictHelper}).ParseFS(templateFiles, "templates/*.html"))
	templ := template.Must(template.New("").Funcs(template.FuncMap{"nicetime": niceTime, "niceURL": niceURL, "join": strings.Join, "version": version.Is}).ParseFS(templateFiles, "templates/*.html"))

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

		sr, err := bmm.Search(query)
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

		meta := gin.H{"page": "edit", "bookmark": bookmark, "tw": gin.H{"tags": bookmark.Tags, "tags_hidden": strings.Join(bookmark.Tags, "|")}}

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

	return server
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
			keys[strings.ToLower(k)] = struct{}{}
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

	return timeVariations{HumanDuration: ago}
}

func niceURL(url string) string {
	if len(url) > 50 {
		return url[0:50] + " ..."
	}
	return url
}
