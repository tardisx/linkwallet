# linkwallet

A self-hosted bookmark database with full-text page content search.

# Feature list

* Simple cross-platform single binary deployment
  * or docker if you prefer
* Bookmarklet, single click to add a bookmark from any webpage
* Full-text search
  * Bookmark content is scraped and indexed locally
  * Page content periodically refreshed automatically
  * Interactively search across titles and content
  * Rippingly fast results, as you type
    * full text search ~60ms (over full text content of 600 bookmarks)
  * No need to remember how you filed something, you just need a keyword
    or two to discover it again
* Embedded database, no separate database required
* Light on resources
  * ~21Mb binary
  * ~40Mb memory
  * ~24Mb database (600 bookmarks, full text content indexed)
* Easily export your bookmarks to a plain text file - your data is yours

# Installation

## Docker

* Copy the `docker-compose.yml-sample` to a directory somewhere
* Rename to `docker-compose.yml` and edit to your needs
  * In most cases, you only need to change the path to the `/data`
    mountpoint.
* Run `docker-compose up -d`

## Packages (deb/rpm)

* Download the .deb or .rpm from the releases
* Install using apt/dpkg/rpm
  * Automatically creates a systemd service, enabled and started
  * Runs as user `linkwallet`
  * Database stored in `/var/lib/linkwallet`

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