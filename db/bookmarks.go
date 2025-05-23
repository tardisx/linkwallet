package db

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/tardisx/linkwallet/content"
	"github.com/tardisx/linkwallet/entity"

	bolthold "github.com/timshannon/bolthold"
)

type BookmarkManager struct {
	db          *DB
	scrapeQueue chan *entity.Bookmark
}

type SearchOptions struct {
	All     bool
	Query   string
	Results int
}

func NewBookmarkManager(db *DB) *BookmarkManager {
	return &BookmarkManager{db: db, scrapeQueue: make(chan *entity.Bookmark)}
}

// AddBookmark adds a bookmark to the database. It returns an error
// if this bookmark already exists (based on URL match).
// The entity.Bookmark ID field will be updated.
func (m *BookmarkManager) AddBookmark(bm *entity.Bookmark) error {

	if strings.Index(bm.URL, "https://") != 0 &&
		strings.Index(bm.URL, "http://") != 0 {
		return errors.New("URL must begin with http:// or https://")
	}

	existing := entity.Bookmark{}
	err := m.db.store.FindOne(&existing, bolthold.Where("URL").Eq(bm.URL))
	if err != bolthold.ErrNotFound {
		return fmt.Errorf("bookmark already exists")
	}
	bm.TimestampCreated = time.Now()
	err = m.db.store.Insert(bolthold.NextSequence(), bm)
	if err != nil {
		return fmt.Errorf("addBookmark returned: %w", err)
	}
	return nil
}

func (m *BookmarkManager) DeleteBookmark(bm *entity.Bookmark) error {
	err := m.db.store.FindOne(bm, bolthold.Where("URL").Eq(bm.URL))
	if err == bolthold.ErrNotFound {
		return fmt.Errorf("bookmark does not exist")
	}

	// delete it
	m.db.store.DeleteMatching(bm, bolthold.Where("ID").Eq(bm.ID))
	// delete all the index entries
	return m.db.bleve.Delete(fmt.Sprint(bm.ID))
}

// ListBookmarks returns all bookmarks.
// func (m *BookmarkManager) ListBookmarks() ([]entity.Bookmark, error) {
// 	bookmarks := make([]entity.Bookmark, 0)
// 	err := m.db.store.Find(&bookmarks, &bolthold.Query{})
// 	if err != nil {
// 		panic(err)
// 	}
// 	log.Printf("found %d bookmarks", len(bookmarks))
// 	return bookmarks, nil
// }

// ExportBookmarks exports all bookmarks to an io.Writer
func (m *BookmarkManager) ExportBookmarks(w io.Writer) error {
	bms := []entity.Bookmark{}
	err := m.db.store.Find(&bms, &bolthold.Query{})
	if err != nil {
		return fmt.Errorf("could not export bookmarks: %w", err)
	}
	for _, bm := range bms {
		w.Write([]byte(bm.URL + "\n"))
	}
	return nil
}

func (m *BookmarkManager) SaveBookmark(bm *entity.Bookmark) error {
	err := m.db.store.Update(bm.ID, &bm)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}
	return nil
}

func (m *BookmarkManager) LoadBookmarkByID(id uint64) entity.Bookmark {
	// log.Printf("loading %v", ids)
	ret := entity.Bookmark{}
	err := m.db.store.Get(id, &ret)
	if err != nil {
		panic(err)
	}
	return ret
}

func (m *BookmarkManager) Search(opts SearchOptions) ([]entity.BookmarkSearchResult, error) {
	found := []entity.BookmarkSearchResult{}
	if opts.All && opts.Query != "" {
		panic("can't fetch all with query")
	}

	var q query.Query

	if opts.All {
		q = bleve.NewMatchAllQuery()
	} else {
		mq := bleve.NewMatchQuery(opts.Query)
		mq.Analyzer = en.AnalyzerName
		tq := bleve.NewTermQuery(opts.Query)

		q = bleve.NewDisjunctionQuery(mq, tq)
	}

	req := bleve.NewSearchRequest(q)
	if opts.Results > 0 {
		req.Size = opts.Results
	}
	req.Highlight = bleve.NewHighlightWithStyle("html")

	sr, err := m.db.bleve.Search(req)
	if err != nil {
		panic(err)
	}
	// log.Printf("%#v", m.db.bleve.StatsMap())

	if sr.Total > 0 {
		for _, dm := range sr.Hits {

			id, _ := strconv.ParseUint(dm.ID, 10, 64)
			bm := m.LoadBookmarkByID(id)
			bsr := entity.BookmarkSearchResult{
				Bookmark:  bm,
				Score:     dm.Score,
				Highlight: template.HTML(strings.Join(dm.Fragments["Info.RawText"], "\n")),
			}
			found = append(found, bsr)
		}
	}

	m.db.IncrementSearches()

	return found, nil
}

func (m *BookmarkManager) ScrapeAndIndex(bm *entity.Bookmark) error {

	log.Printf("Start scrape for %s", bm.URL)
	info := content.FetchPageInfo(*bm)
	// keep the existing title if necessary
	if bm.PreserveTitle {
		info.Title = bm.Info.Title
	}
	bm.Info = info
	bm.TimestampLastScraped = time.Now()
	err := m.SaveBookmark(bm)
	if err != nil {
		panic(err)
	}

	m.UpdateIndexForBookmark(bm)
	return nil

}

func (m *BookmarkManager) UpdateIndexForBookmark(bm *entity.Bookmark) {
	log.Printf("inserting into bleve data for %s", bm.URL)
	err := m.db.bleve.Index(fmt.Sprint(bm.ID), bm)
	if err != nil {
		panic(err)
	}
}

func (m *BookmarkManager) QueueScrape(bm *entity.Bookmark) {
	m.scrapeQueue <- bm
}

func (m *BookmarkManager) RunQueue() {
	type localScrapeQueue struct {
		queue []*entity.Bookmark
		mutex sync.Mutex
	}

	localQueue := localScrapeQueue{queue: make([]*entity.Bookmark, 0)}
	// accept things off the queue immediately
	go func() {
		for {
			newItem := <-m.scrapeQueue

			newItem.TimestampLastScraped = time.Now()
			err := m.SaveBookmark(newItem)
			if err != nil {
				panic(err)
			}

			localQueue.mutex.Lock()
			localQueue.queue = append(localQueue.queue, newItem)
			localQueue.mutex.Unlock()
			log.Printf("queue now has %d entries", len(localQueue.queue))
		}
	}()

	for {
		localQueue.mutex.Lock()
		if len(localQueue.queue) > 0 {
			processBM := localQueue.queue[0]
			localQueue.queue = localQueue.queue[1:]
			localQueue.mutex.Unlock()

			m.ScrapeAndIndex(processBM)

		} else {
			localQueue.mutex.Unlock()
		}
		time.Sleep(time.Second)

	}

}

func (m *BookmarkManager) UpdateContent() {
	ret := make([]entity.Bookmark, 0)
	for {
		ret = []entity.Bookmark{}
		deadline := time.Now().Add(time.Hour * -24 * 7)
		err := m.db.store.Find(&ret, bolthold.Where("TimestampLastScraped").Lt(deadline))
		if err == bolthold.ErrNotFound {
			log.Printf("none qualify")
			time.Sleep(time.Second)
			continue
		}
		if err != nil {
			panic(err)
		}

		for _, bm := range ret {
			thisBM := bm
			log.Printf("queueing %d because %s", thisBM.ID, thisBM.TimestampLastScraped)
			m.QueueScrape(&thisBM)
		}
		time.Sleep(time.Second * 5)
	}
}

// AllBookmarks returns all bookmarks. It does not use the index for this
// operation.
func (m *BookmarkManager) AllBookmarks() ([]entity.Bookmark, error) {
	bookmarks := make([]entity.Bookmark, 0)
	err := m.db.store.Find(&bookmarks, &bolthold.Query{})
	if err != nil {
		panic(err)
	}
	return bookmarks, nil
}

func (m *BookmarkManager) Stats() (entity.DBStats, error) {
	stats := entity.DBStats{}
	err := m.db.store.Get("stats", &stats)
	if err != nil && err != bolthold.ErrNotFound {
		return stats, fmt.Errorf("could not load stats: %s", err)
	}
	// get the DB size
	fi, err := os.Stat(m.db.file)
	if err != nil {
		return stats, fmt.Errorf("could not load db file size: %s", err)
	}
	stats.FileSize = int(fi.Size())
	indexSize, err := getBleveIndexSize(m.db.file + ".bleve")
	if err != nil {
		return entity.DBStats{}, err
	}
	stats.IndexSize = int(indexSize)

	return stats, nil
}

func getBleveIndexSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
