package db

func (db *DB) InitIndices() {
	panic("unimplemented")
	// wi := entity.WordIndex{}
	// db.store.DeleteMatching(wi, &bolthold.Query{})
}

// func (db *DB) IndexDocument(id uint64, info entity.PageInfo) {
// 	log.Printf("I am indexing!")
// 	err := db.bleve.Index(fmt.Sprint(id), info)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// func (db *DB) UpdateIndexForWordsByID(words []string, id uint64) {
// 	panic("I should not be called")
// }

func (db *DB) DumpIndex() {
	panic("unimplemented")

}
