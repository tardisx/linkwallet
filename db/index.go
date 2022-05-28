package db

import (
	"log"
	"time"

	"github.com/tardisx/linkwallet/entity"
	bolthold "github.com/timshannon/bolthold"
)

func (db *DB) InitIndices() {
	wi := entity.WordIndex{}
	db.store.DeleteMatching(wi, &bolthold.Query{})
}

func (db *DB) UpdateIndexForWordsByID(words []string, id uint64) {
	// delete this id from all indices
	txn, err := db.store.Bolt().Begin(true)
	if err != nil {
		panic(err)
	}

	db.store.TxForEach(txn, &bolthold.Query{}, func(wi *entity.WordIndex) {
		// log.Printf("considering this one: %s", wi.Word)
		delete(wi.Bitmap, id)
	})

	// adding
	var find, store time.Duration
	for i, word := range words {
		// log.Printf("indexing %s", word)
		tF := time.Now()
		thisWI := entity.WordIndex{Word: word}
		err := db.store.TxGet(txn, "word_index_"+word, &thisWI)
		if err == bolthold.ErrNotFound {
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
			txn, err = db.store.Bolt().Begin(true)
			if err != nil {
				panic(err)
			}
		}

	}
	//log.Printf("find %s store %s", find, store)

	txn.Commit()
}

func (db *DB) DumpIndex() {

	// delete this id from all indices
	err := db.store.ForEach(&bolthold.Query{}, func(wi *entity.WordIndex) error {
		log.Printf("%10s: %v", wi.Word, wi.Bitmap)
		return nil
	})
	if err != nil {
		panic(err)
	}

}
