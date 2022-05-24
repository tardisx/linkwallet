package db

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tardisx/linkwallet/content"
	"github.com/tardisx/linkwallet/entity"

	"github.com/timshannon/badgerhold/v4"
)

type BookmarkManager struct {
	db          *DB
	scrapeQueue chan *entity.Bookmark
}

func NewBookmarkManager(db *DB) *BookmarkManager {
	return &BookmarkManager{db: db, scrapeQueue: make(chan *entity.Bookmark)}
}

// AddBookmark adds a bookmark to the database. It returns an error
// if this bookmark already exists (based on URL match).
// The entity.Bookmark ID field will be updated.
func (m *BookmarkManager) AddBookmark(bm *entity.Bookmark) error {
	existing := entity.Bookmark{}
	err := m.db.store.FindOne(&existing, badgerhold.Where("URL").Eq(bm.URL))
	if err != badgerhold.ErrNotFound {
		return fmt.Errorf("bookmark already exists")
	}
	bm.TimestampCreated = time.Now()
	err = m.db.store.Insert(badgerhold.NextSequence(), bm)
	if err != nil {
		return fmt.Errorf("addBookmark returned: %w", err)
	}
	return nil
}

// ListBookmarks returns all bookmarks.
func (m *BookmarkManager) ListBookmarks() ([]entity.Bookmark, error) {
	bookmarks := make([]entity.Bookmark, 0, 0)
	err := m.db.store.Find(&bookmarks, &badgerhold.Query{})
	if err != nil {
		panic(err)
	}
	return bookmarks, nil
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

func (m *BookmarkManager) LoadBookmarksByIDs(ids []uint64) []entity.Bookmark {
	// log.Printf("loading %v", ids)
	ret := make([]entity.Bookmark, 0, 0)

	s := make([]interface{}, len(ids))
	for i, v := range ids {
		s[i] = v
	}

	err := m.db.store.Find(&ret, badgerhold.Where("ID").In(s...))
	if err != nil {
		panic(err)
	}
	return ret
}

func (m *BookmarkManager) Search(query string) ([]entity.Bookmark, error) {
	rets := make([]uint64, 0, 0)

	counts := make(map[uint64]uint8)

	words := content.StringToSearchWords(query)

	for _, word := range words {
		var wi *entity.WordIndex
		err := m.db.store.Get("word_index_"+word, &wi)
		if err == badgerhold.ErrNotFound {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error retrieving index: %w", err)
		}
		for k := range wi.Bitmap {
			counts[k]++
		}
	}

	// log.Printf("counts: %#v", counts)

	for k, v := range counts {
		if v == uint8(len(words)) {
			rets = append(rets, k)
			if len(rets) > 10 {
				break
			}
		}
	}

	return m.LoadBookmarksByIDs(rets), nil
}

func (m *BookmarkManager) ScrapeAndIndex(bm *entity.Bookmark) error {

	log.Printf("Start scrape for %s", bm.URL)
	info := content.FetchPageInfo(*bm)
	bm.Info = info
	bm.TimestampLastScraped = time.Now()
	err := m.SaveBookmark(bm)
	if err != nil {
		panic(err)
	}

	words := content.Words(bm)
	log.Printf("index for %d %s (%d words)", bm.ID, bm.URL, len(words))
	m.db.UpdateIndexForWordsByID(words, bm.ID)

	return nil
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
		err := m.db.store.Find(&ret, badgerhold.Where("TimestampLastScraped").Lt(deadline))
		if err == badgerhold.ErrNotFound {
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
