package db

import (
	"fmt"
	"time"

	"github.com/tardisx/linkwallet/entity"
	bolthold "github.com/timshannon/bolthold"
)

type DB struct {
	store *bolthold.Store
	file  string
}

func (db *DB) Open(path string) error {
	// options := bolthold.DefaultOptions
	// options.Dir = dir
	// options.ValueDir = dir
	store, err := bolthold.Open(path, 0666, nil)
	if err != nil {
		return fmt.Errorf("cannot open '%s' - %s", path, err)
	}
	db.store = store
	db.file = path
	return nil
}

func (db *DB) Close() {
	db.store.Close()
}

// func (db *DB) Dumpy() {
// 	res := make([]entity.Bookmark, 0, 0)
// 	db.store.Find(&res, &bolthold.Query{})
// 	log.Printf("%v", res)
// }

// IncrementSearches increments the number of searches we have ever performed by one.
func (db *DB) IncrementSearches() error {
	txn, err := db.store.Bolt().Begin(true)
	if err != nil {
		return fmt.Errorf("could not start transaction for increment searches: %s", err)
	}

	stats := entity.DBStats{}
	err = db.store.TxGet(txn, "stats", &stats)
	if err != nil && err != bolthold.ErrNotFound {
		txn.Rollback()
		return fmt.Errorf("could not get stats for incrementing searches: %s", err)
	}

	stats.Searches += 1
	err = db.store.TxUpsert(txn, "stats", &stats)
	if err != nil {
		txn.Rollback()
		return fmt.Errorf("could not upsert stats for incrementing searches: %s", err)
	}
	err = txn.Commit()
	if err != nil {
		return fmt.Errorf("could not commit increment searches transaction: %s", err)
	}

	return nil
}

// UpdateBookmarkStats updates the history on the number of bookmarks and words indexed.
func (db *DB) UpdateBookmarkStats() error {

	txn, err := db.store.Bolt().Begin(true)
	if err != nil {
		return fmt.Errorf("could not start transaction for update stats: %s", err)
	}
	// count bookmarks and words indexed
	bmI := entity.Bookmark{}
	wiI := entity.WordIndex{}
	bookmarkCount, err := db.store.TxCount(txn, &bmI, &bolthold.Query{})
	if err != nil {
		txn.Rollback()
		return fmt.Errorf("could not get bookmark count: %s", err)
	}
	indexWordCount, err := db.store.TxCount(txn, &wiI, &bolthold.Query{})
	if err != nil {
		txn.Rollback()
		return fmt.Errorf("could not get index word count: %s", err)
	}

	// bucket these stats by day
	now := time.Now().Truncate(time.Hour * 24)

	stats := entity.DBStats{}
	err = db.store.TxGet(txn, "stats", &stats)
	if err != nil && err != bolthold.ErrNotFound {
		txn.Rollback()
		return fmt.Errorf("could not get stats: %s", err)
	}
	if stats.History == nil {
		stats.History = make(map[time.Time]entity.BookmarkInfo)
	}
	stats.History[now] = entity.BookmarkInfo{Bookmarks: bookmarkCount, IndexedWords: indexWordCount}
	err = db.store.TxUpsert(txn, "stats", &stats)
	if err != nil {
		txn.Rollback()
		return fmt.Errorf("could not upsert stats: %s", err)
	}

	err = txn.Commit()
	if err != nil {
		return fmt.Errorf("could not commit stats transaction: %s", err)
	}

	return nil
}
