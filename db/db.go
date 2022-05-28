package db

import (
	"log"

	"github.com/tardisx/linkwallet/entity"
	bolthold "github.com/timshannon/bolthold"
)

type DB struct {
	store *bolthold.Store
}

func (db *DB) Open(dir string) {
	// options := bolthold.DefaultOptions
	// options.Dir = dir
	// options.ValueDir = dir
	store, err := bolthold.Open("bolt.db", 0666, nil)
	if err != nil {
		panic(err)

	}
	db.store = store
}

func (db *DB) Close() {
	db.store.Close()
}

func (db *DB) Dumpy() {
	res := make([]entity.Bookmark, 0, 0)
	db.store.Find(&res, &bolthold.Query{})
	log.Printf("%v", res)
}
