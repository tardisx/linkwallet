package db

import (
	"fmt"
	"log"

	"github.com/tardisx/linkwallet/entity"
	bolthold "github.com/timshannon/bolthold"
)

type DB struct {
	store *bolthold.Store
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
	return nil
}

func (db *DB) Close() {
	db.store.Close()
}

func (db *DB) Dumpy() {
	res := make([]entity.Bookmark, 0, 0)
	db.store.Find(&res, &bolthold.Query{})
	log.Printf("%v", res)
}
