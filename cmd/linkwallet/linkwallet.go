package main

import (
	"log"

	"github.com/tardisx/linkwallet/db"
	"github.com/tardisx/linkwallet/version"
	"github.com/tardisx/linkwallet/web"
)

func main() {

	dbh := db.DB{}
	dbh.Open("badger")
	bmm := db.NewBookmarkManager(&dbh)

	go func() { version.UpdateVersionInfo() }()

	log.Printf("linkallet verison %s starting", version.Is())

	server := web.Create(bmm)
	go bmm.RunQueue()
	go bmm.UpdateContent()
	server.Start()
}
