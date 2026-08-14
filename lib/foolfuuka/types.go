// Package foolfuuka is a minimal client for FoolFuuka-based 4chan archives
// (desuarchive.org, archive.palanq.win, and any other instance of the same
// software): board index and thread endpoints only. Responses are
// translated into fourchan.Catalog/fourchan.Post so callers (diff,
// scheduler, detect) work against archive sources exactly as they do
// against live 4chan.
//
// Like lib/fourchan, image fields are deliberately absent — this is a
// text-only client.
package foolfuuka

// apiPost is one post as FoolFuuka's JSON API returns it, from either the
// board index or the thread endpoint. Numeric-looking fields other than
// timestamp are strings in the wire format.
type apiPost struct {
	Num              string `json:"num"`
	ThreadNum        string `json:"thread_num"`
	IsOP             string `json:"op"`
	Timestamp        int64  `json:"timestamp"`
	TimestampExpired string `json:"timestamp_expired"`
	Title            string `json:"title"`
	Comment          string `json:"comment"`
	Sticky           string `json:"sticky"`
	Locked           string `json:"locked"`
	Deleted          string `json:"deleted"`
}

// apiIndexEntry is one thread as returned by the board index endpoint
// (/_/api/chan/index/): the OP plus whatever trailing replies FoolFuuka
// includes inline, with Omitted counting the rest.
type apiIndexEntry struct {
	Omitted int       `json:"omitted"`
	Op      apiPost   `json:"op"`
	Posts   []apiPost `json:"posts"`
}

// apiIndexResponse is the board index response: threads keyed by thread
// number, unordered.
type apiIndexResponse map[string]apiIndexEntry

// apiThreadEntry is a full thread as returned by the thread endpoint
// (/_/api/chan/thread/): the OP plus every reply, keyed by post number.
type apiThreadEntry struct {
	Op    apiPost            `json:"op"`
	Posts map[string]apiPost `json:"posts"`
}

// apiThreadResponse is the thread endpoint's response: a single entry
// keyed by thread number. A request for a thread that doesn't exist
// returns HTTP 200 with {"error": "..."} instead of a 404, so callers
// must check for that shape before decoding this one — see
// decodeThreadResponse.
type apiThreadResponse map[string]apiThreadEntry
