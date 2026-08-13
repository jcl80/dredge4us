// Package detect defines the detector boundary: implementations read
// posts held in memory by the caller and return only classified matches.
// No raw post text is allowed to leave this package.
package detect

import (
	"time"

	"github.com/jcl80/dredge4us/lib/fourchan"
)

// Finding is the only artifact a Detector may produce. Note is optional
// detector-supplied rationale — distinct from MatchedValue, which stays
// URL/hash only, never prose.
type Finding struct {
	Board         string
	ThreadNo      int
	PostNo        int
	PostTime      time.Time
	Detector      string
	Kind          string
	MatchedValue  string
	Note          string
	ThreadSubject string
	ThreadReplies int
}

// Detector scans a thread's posts and returns whatever it found.
// Implementations must not retain posts, or any text derived from them,
// beyond the call.
type Detector interface {
	Name() string
	Detect(board string, thread fourchan.Thread, posts []fourchan.Post) []Finding
}
