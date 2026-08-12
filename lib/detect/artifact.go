package detect

import (
	"html"
	"regexp"
	"strings"

	"github.com/jcl80/dredge4us/lib/fourchan"
)

// ArtifactDetector flags posts referencing offsite artifacts worth
// following up on: model weights, source repos, and the dump sites people
// use to route around 4chan's own attachment limits. Regex-only — no
// judgement about whether the artifact is legitimate, no LLM.
type ArtifactDetector struct{}

// NewArtifactDetector returns a ready-to-use ArtifactDetector.
func NewArtifactDetector() *ArtifactDetector { return &ArtifactDetector{} }

// Name identifies this detector in Finding.Detector.
func (d *ArtifactDetector) Name() string { return "artifact" }

var artifactPatterns = map[string]*regexp.Regexp{
	"magnet_uri":      regexp.MustCompile(`magnet:\?xt=urn:[a-zA-Z0-9]+:[a-zA-Z0-9]+[^\s"'<>]*`),
	"huggingface_url": regexp.MustCompile(`https?://huggingface\.co/[^\s"'<>]+`),
	"github_url":      regexp.MustCompile(`https?://(gist\.)?github\.com/[^\s"'<>]+`),
	"dump_site_url":   regexp.MustCompile(`https?://(www\.)?(pastebin\.com|catbox\.moe|files\.catbox\.moe|litterbox\.catbox\.moe|anonfiles\.com)/[^\s"'<>]+`),
	"sha256_hex":      regexp.MustCompile(`\b[a-fA-F0-9]{64}\b`),
	"model_filename":  regexp.MustCompile(`\b[\w-]+\.(safetensors|gguf|ckpt|pt)\b`),
}

var (
	htmlTagPattern    = regexp.MustCompile(`<[^>]+>`)
	whitespacePattern = regexp.MustCompile(`\s+`)
)

// plainText strips 4chan's HTML fragment markup (entity-escaped text,
// <br>-newlines, quote spans) down to matchable text. Tags are replaced
// with a space, not deleted outright — 4chan escapes any literal '<'/'>'
// a user types, so every real tag here is 4chan's own line/formatting
// markup, and deleting it instead would glue adjacent lines together
// (e.g. "foo<br>https://...") into one run the URL regex then over-matches
// into. It's a simplification, not a full HTML parse — good enough for
// regex matching, not for reconstructing exact post formatting.
func plainText(com string) string {
	text := htmlTagPattern.ReplaceAllString(com, " ")
	text = html.UnescapeString(text)
	return strings.TrimSpace(whitespacePattern.ReplaceAllString(text, " "))
}

// Detect implements Detector.
func (d *ArtifactDetector) Detect(board string, thread fourchan.Thread, posts []fourchan.Post) []Finding {
	var findings []Finding

	for _, p := range posts {
		text := plainText(p.Com)
		for kind, re := range artifactPatterns {
			for _, match := range re.FindAllString(text, -1) {
				findings = append(findings, Finding{
					Board:         board,
					ThreadNo:      thread.No,
					PostNo:        p.No,
					PostTime:      p.PostTime(),
					Detector:      d.Name(),
					Kind:          kind,
					MatchedValue:  match,
					ThreadSubject: thread.Sub,
					ThreadReplies: thread.Replies,
				})
			}
		}
	}

	return findings
}
