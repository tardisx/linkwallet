package main

import (
	"flag"
	"log"
	"time"

	"github.com/tardisx/linkwallet/db"
	v "github.com/tardisx/linkwallet/version"
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
	rescrape, err := dbh.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	bmm := db.NewBookmarkManager(&dbh)
	cmm := db.NewConfigManager(&dbh)

	go func() {
		for {
			v.VersionInfo.UpdateVersionInfo()
			time.Sleep(time.Hour * 6)
		}
	}()

	// update stats every 5 minutes
	go func() {
		for {
			err := dbh.UpdateBookmarkStats()
			if err != nil {
				panic(err)
			}
			time.Sleep(time.Minute * 5)
		}
	}()

	log.Printf("linkwallet version %s starting", v.VersionInfo.Local.Version)

	server := web.Create(bmm, cmm)
	go bmm.RunQueue()
	go bmm.UpdateContent()

	if rescrape {
		log.Printf("queueing all bookmarks for rescraping, as index was just created")
		bookmarks, err := bmm.AllBookmarks()
		if err != nil {
			log.Printf("could not load all bookmarks: %s", err.Error())
		} else {
			for _, bm := range bookmarks {
				bmm.QueueScrape(&bm)
			}
		}
		log.Printf("queued %d bookmarks for scraping", len(bookmarks))
	}

	server.Start()
}
