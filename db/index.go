package db

import (
	"log"
	"time"

	"github.com/tardisx/linkwallet/entity"

	badgerhold "github.com/timshannon/badgerhold/v4"
)

func (db *DB) InitIndices() {
	wi := entity.WordIndex{}
	db.store.DeleteMatching(wi, &badgerhold.Query{})
}

func (db *DB) UpdateIndexForWordsByID(words []string, id uint64) {
	// delete this id from all indices
	txn := db.store.Badger().NewTransaction(true)

	db.store.TxForEach(txn, &badgerhold.Query{}, func(wi *entity.WordIndex) {
		// log.Printf("considering this one: %s", wi.Word)
		delete(wi.Bitmap, id)
	})

	// addiing
	var find, store time.Duration
	for i, word := range words {
		// log.Printf("indexing %s", word)
		tF := time.Now()
		thisWI := entity.WordIndex{Word: word}
		err := db.store.TxGet(txn, "word_index_"+word, &thisWI)
		// err := db.store.TxFindOne(txn, &thisWI, badgerhold.Where("Word").Eq(word).Index("Word"))
		if err == badgerhold.ErrNotFound {
			// create it
			thisWI.Bitmap = map[uint64]bool{}
		} else if err != nil {
			panic(err)
		}
		findT := time.Since(tF)

		tS := time.Now()
		thisWI.Bitmap[id] = true
		// log.Printf("BM: %v", thisWI.Bitmap)
		err = db.store.TxUpsert(txn, "word_index_"+word, thisWI)
		if err != nil {
			panic(err)
		}
		findS := time.Since(tS)
		find += findT
		store += findS

		if i > 0 && i%100 == 0 {
			txn.Commit()
			txn = db.store.Badger().NewTransaction(true)
		}

	}
	//log.Printf("find %s store %s", find, store)

	txn.Commit()
}

func (db *DB) DumpIndex() {

	// delete this id from all indices
	err := db.store.ForEach(&badgerhold.Query{}, func(wi *entity.WordIndex) error {
		log.Printf("%10s: %v", wi.Word, wi.Bitmap)
		return nil
	})
	if err != nil {
		panic(err)
	}

}
