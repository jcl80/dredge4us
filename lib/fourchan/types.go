// Package fourchan is a minimal client for 4chan's read API
// (a.4cdn.org): catalog and thread endpoints only.
//
// Image fields (tim, ext, md5, fsize, w, h, tn_w, tn_h, filename) are
// deliberately absent from these structs. Nothing in this package can
// construct an i.4cdn.org URL, because nothing here decodes the data an
// i.4cdn.org URL is built from.
package fourchan

import "time"

// Catalog is the full response from GET /{board}/catalog.json: a list of
// pages, each holding a page of threads.
type Catalog []Page

// Page is one page of a board's catalog.
type Page struct {
	Page    int      `json:"page"`
	Threads []Thread `json:"threads"`
}

// Thread is a catalog entry for one thread. Com is the OP's comment as
// returned by the catalog endpoint, which 4chan truncates — it must never
// be treated as the full post body.
type Thread struct {
	No           int    `json:"no"`
	Sub          string `json:"sub"`
	Com          string `json:"com"`
	Replies      int    `json:"replies"`
	LastModified int64  `json:"last_modified"`
	Time         int64  `json:"time"`
	Sticky       int    `json:"sticky"`
	Closed       int    `json:"closed"`
	Archived     int    `json:"archived"`
}

// PostTime returns the thread's OP timestamp.
func (t Thread) PostTime() time.Time { return time.Unix(t.Time, 0).UTC() }

// ThreadResponse is the response from GET /{board}/thread/{no}.json.
type ThreadResponse struct {
	Posts []Post `json:"posts"`
}

// Post is a single post's text content, as returned by the thread
// endpoint. Com is HTML-fragment text (entity-escaped, <br>-newlined) —
// see Detector implementations for normalization before matching.
type Post struct {
	No       int    `json:"no"`
	Resto    int    `json:"resto"`
	Time     int64  `json:"time"`
	Sub      string `json:"sub"`
	Com      string `json:"com"`
	Sticky   int    `json:"sticky"`
	Closed   int    `json:"closed"`
	Archived int    `json:"archived"`
}

// PostTime returns the post's timestamp.
func (p Post) PostTime() time.Time { return time.Unix(p.Time, 0).UTC() }
