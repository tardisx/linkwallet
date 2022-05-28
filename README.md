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

# Roadmap

* More options when managing links
  * delete
  * sorting
* More tag options
  * search for tags
  * bookmarklet with pre-filled tags