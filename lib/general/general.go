// Package general detects and normalizes 4chan "general" threads — the
// recurring, long-running thread convention (e.g. "/lmg/ - Local Models
// General") where a thread hits its reply cap, dies, and gets reborn as
// a new thread with a near-identical subject to continue the
// conversation. Heuristic only, regex-based — no LLM, consistent with
// the rest of lib.
package general

import (
	"regexp"
	"strings"
)

var generalWordPattern = regexp.MustCompile(`(?i)\bgeneral\b`)

// IsGeneral reports whether subject looks like a general thread: it
// contains the word "general" as a whole word, case-insensitive. This
// will have false positives/negatives — it's a cheap heuristic, not a
// classifier.
func IsGeneral(subject string) bool {
	return generalWordPattern.MatchString(subject)
}

var (
	editionPattern     = regexp.MustCompile(`(?i)[-#(]*\s*(edition|part|ed\.?)\s*[:#]?\s*\d+\s*\)?`)
	trailingNumPattern = regexp.MustCompile(`#?\d+\s*$`)
	punctPattern       = regexp.MustCompile(`[^\w\s]`)
	whitespacePattern  = regexp.MustCompile(`\s+`)
)

// NormalizeSubject reduces a thread subject to a stable key used to
// stitch successive instances of the same general together — stripping
// edition numbers and punctuation that vary between otherwise-identical
// reposts. Two subjects with the same normalized key, in the same
// board, are treated as the same general lineage.
func NormalizeSubject(subject string) string {
	s := strings.ToLower(subject)
	s = editionPattern.ReplaceAllString(s, " ")
	s = trailingNumPattern.ReplaceAllString(s, " ")
	s = punctPattern.ReplaceAllString(s, " ")
	s = whitespacePattern.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
