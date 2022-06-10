package db

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tardisx/linkwallet/content"
	"github.com/tardisx/linkwallet/entity"

	bolthold "github.com/timshannon/bolthold"
)

type BookmarkManager struct {
	db          *DB
	scrapeQueue chan *entity.Bookmark
}

type SearchOptions struct {
	Query string
	Tags  []string
	Sort  string
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
	m.db.UpdateIndexForWordsByID([]string{}, bm.ID)
	return nil
}

// ListBookmarks returns all bookmarks.
func (m *BookmarkManager) ListBookmarks() ([]entity.Bookmark, error) {
	bookmarks := make([]entity.Bookmark, 0, 0)
	err := m.db.store.Find(&bookmarks, &bolthold.Query{})
	if err != nil {
		panic(err)
	}
	return bookmarks, nil
}

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
	log.Printf("loading id %d", id)
	err := m.db.store.Get(id, &ret)
	if err != nil {
		panic(err)
	}
	return ret
}

func (m *BookmarkManager) Search(opts SearchOptions) ([]entity.Bookmark, error) {

	// first get a list of all the ids that match our query
	idsMatchingQuery := make([]uint64, 0, 0)
	counts := make(map[uint64]uint8)
	words := content.StringToSearchWords(opts.Query)

	for _, word := range words {
		var wi *entity.WordIndex
		err := m.db.store.Get("word_index_"+word, &wi)
		if err == bolthold.ErrNotFound {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error retrieving index: %w", err)
		}
		for k := range wi.Bitmap {
			counts[k]++
		}
	}

	for k, v := range counts {
		if v == uint8(len(words)) {
			idsMatchingQuery = append(idsMatchingQuery, k)
			if len(idsMatchingQuery) > 10 {
				break
			}
		}
	}

	// now we can do our search
	bhQuery := bolthold.Query{}
	if opts.Query != "" {
		bhQuery = bolthold.Query(*bhQuery.And("ID").In(bolthold.Slice(idsMatchingQuery)...))
	}
	if opts.Tags != nil && len(opts.Tags) > 0 {
		bhQuery = bolthold.Query(*bhQuery.And("Tags").ContainsAll(bolthold.Slice(opts.Tags)...))
	}

	reverse := false
	sortOrder := opts.Sort
	if sortOrder != "" && sortOrder[0] == '-' {
		reverse = true
		sortOrder = sortOrder[1:]
	}

	if sortOrder == "title" {
		bhQuery.SortBy("Info.Title")
	} else if sortOrder == "created" {
		bhQuery.SortBy("TimestampCreated")
	} else if sortOrder == "scraped" {
		bhQuery.SortBy("TimestampLastScraped")
	} else {
		bhQuery.SortBy("ID")
	}

	if reverse {
		bhQuery = *bhQuery.Reverse()
	}

	out := []entity.Bookmark{}
	err := m.db.store.ForEach(&bhQuery,
		func(bm *entity.Bookmark) error {
			out = append(out, *bm)

			return nil
		})
	if err != nil {
		panic(err)
	}

	return out, nil
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
	words := content.Words(bm)
	words = append(words, bm.Tags...)
	log.Printf("index for %d %s (%d words)", bm.ID, bm.URL, len(words))
	m.db.UpdateIndexForWordsByID(words, bm.ID)
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
