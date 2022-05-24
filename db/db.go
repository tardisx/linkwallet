package db

import (
	"log"

	"github.com/tardisx/linkwallet/entity"

	badgerhold "github.com/timshannon/badgerhold/v4"
)

type DB struct {
	store *badgerhold.Store
}

func (db *DB) Open(dir string) {
	options := badgerhold.DefaultOptions
	options.Dir = dir
	options.ValueDir = dir
	store, err := badgerhold.Open(options)
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
	db.store.Find(&res, &badgerhold.Query{})
	log.Printf("%v", res)
}
