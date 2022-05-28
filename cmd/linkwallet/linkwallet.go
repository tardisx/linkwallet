package main

import (
	"flag"
	"log"

	"github.com/tardisx/linkwallet/db"
	"github.com/tardisx/linkwallet/version"
	"github.com/tardisx/linkwallet/web"
)

func main() {

	var dbPath string
	flag.StringVar(&dbPath, "db-path", "", "path to the database file")
	flag.Parse()

	if dbPath == "" {
		log.Fatal("You need to specify the path to the database file with -db-path")
	}

	dbh := db.DB{}
	err := dbh.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	bmm := db.NewBookmarkManager(&dbh)
	cmm := db.NewConfigManager(&dbh)

	go func() { version.UpdateVersionInfo() }()

	log.Printf("linkwallet version %s starting", version.Is())

	server := web.Create(bmm, cmm)
	go bmm.RunQueue()
	go bmm.UpdateContent()
	server.Start()
}
