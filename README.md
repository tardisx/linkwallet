
# ARCHIVED - 2025-05-23

Due to GitHub's AI enshittification, this project has been moved to https://code.ppl.town/justin/linkwallet

# linkwallet

[![Go Report Card](https://goreportcard.com/badge/github.com/tardisx/linkwallet)](https://goreportcard.com/report/github.com/tardisx/linkwallet)

A self-hosted bookmark database with full-text page content search.

linkwallet uses the [Bleve](https://blevesearch.com) indexing library, providing
excellent support for free-text queries over the content of all your bookmarked
pages.

![Search][screenshot_search]

linkwallet indexes the page content, and automatically re-scrapes the pages
periodically. Tags can be applied (though with the full-text search they are
often not needed). Bookmarks can be easily managed, and can be imported or
exported in bulk.

![Admin][screenshot_admin]

Bookmarks can be added with two clicks via the bookmarklet.

![Bookmarklet][screenshot_bookmarklet]

# Feature list

* Simple cross-platform single binary deployment
  * or docker if you prefer
* Bookmarklet, single click to add a bookmark from any webpage
* Full-text search
  * Bookmark content is scraped and indexed locally
  * Page content periodically refreshed automatically
  * Interactively search across titles and content
  * Rippingly fast results, as you type
    * full text search ~30ms (over full text content of 600 bookmarks)
  * No need to remember how you filed something, you just need a keyword
    or two to discover it again
* Embedded database, no separate database required
* Extremely light on resources
* Easily export your bookmarks to a plain text file - your data is yours

# Installation

## Docker

* Copy the `docker-compose.yml-sample` to a directory somewhere
* Rename to `docker-compose.yml` and edit to your needs
  * In most cases, you only need to change the path to the `/data`
    mountpoint.
* Run `docker-compose up -d`

To upgrade:

* `docker-compose pull`
* `docker-compose up -d`

## Packages (deb/rpm)

[not yet migrated to new goreleaser - please message me if you need packages]

## Binary

* Download the appropriate binary from the releases page
* Install somewhere on your system
* Run `./linkwallet -db-path /some/path/xxxx.db` where `/some/path/xxxx.db`
  is the location of your bookmarks database (will be created if it does not yet exist)

## Source

* Checkout the code
* `go build cmd/linkwallet/linkwallet.go`

# Using

linkwallet is a 100% web-driven app. After running, hit the web interface
on port 8109 (docker using the sample docker-compose.yml) or 8080 (default
on binary).

Change the port number by setting the PORT environment variable.

If you put linkwallet on a separate machine, or behind a reverse proxy,
go into the config page and set the correct `BaseURL` parameter, or the bookmarklets
will not work.

# Roadmap

* More options when managing links
  * sorting
* More tag options
  * bookmarklet with pre-filled tags
  * search/filter on tags

[screenshot_search]: https://raw.githubusercontent.com/tardisx/linkwallet/main/screenshot_search.png
[screenshot_admin]: https://raw.githubusercontent.com/tardisx/linkwallet/main/screenshot_admin.png
[screenshot_bookmarklet]: https://raw.githubusercontent.com/tardisx/linkwallet/main/screenshot_bookmarklet.png

