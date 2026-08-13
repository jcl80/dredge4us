// Package narrate generates prose narrative summaries of a trailing
// time window's findings/generals activity, via one LLM call per
// window. Unlike detect.LLMClassifier (one call per thread, run inline
// during polling), this is meant to be called on its own schedule —
// see server/cmd/summarizer.
package narrate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// Finding is one detector match, as input to Summarize.
type Finding struct {
	Board         string
	Kind          string
	MatchedValue  string
	Note          string
	ThreadSubject string
}

// KindCount is a finding kind's total count across the whole window —
// separate from Findings, which callers cap to a handful of examples.
type KindCount struct {
	Kind  string
	Count int
}

// GeneralActivity is one general thread that started or ended within
// the window.
type GeneralActivity struct {
	Board   string
	Subject string
	Replies int
}

// WindowActivity is everything observed within one trailing time
// window, as input to Summarize.
type WindowActivity struct {
	Label         string // e.g. "the last hour"
	PeriodStart   time.Time
	PeriodEnd     time.Time
	Findings      []Finding   // examples only — callers should cap this
	ByKind        []KindCount // the full-window breakdown, uncapped
	NewGenerals   []GeneralActivity
	EndedGenerals []GeneralActivity
}

// TotalFindings sums ByKind — the true count, independent of how many
// example Findings were included.
func (w WindowActivity) TotalFindings() int {
	n := 0
	for _, kc := range w.ByKind {
		n += kc.Count
	}
	return n
}

// Summarizer generates prose narrative summaries via an LLM.
type Summarizer struct {
	client openai.Client
	model  string
}

// NewSummarizer returns a Summarizer against apiKey. model is typically
// "gpt-5.5".
func NewSummarizer(apiKey, model string) *Summarizer {
	return &Summarizer{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// noActivityText is returned without calling the LLM when a window has
// nothing to report — no reason to pay for a summary of silence.
const noActivityText = "No notable activity."

// Summarize returns a short prose narrative of w.
func (s *Summarizer) Summarize(ctx context.Context, w WindowActivity) (string, error) {
	if w.TotalFindings() == 0 && len(w.NewGenerals) == 0 && len(w.EndedGenerals) == 0 {
		return noActivityText, nil
	}

	resp, err := s.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: s.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(buildPrompt(w)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("narrate: summarize request failed: %w", err)
	}

	text := strings.TrimSpace(resp.OutputText())
	if text == "" {
		return "", fmt.Errorf("narrate: summarize returned empty text")
	}
	return text, nil
}

const systemPrompt = `You are writing a short executive briefing for an analyst monitoring 4chan boards for tech/AI intelligence signals: artifact drops, capability claims, leaks, vulnerability disclosures, insider tips, and community sentiment shifts. You'll be given detector findings and general-thread activity for a trailing time window. Write 2-4 sentences of plain prose (no markdown, no bullet points, no headers) summarizing what happened. Call out anything genuinely notable (leaks, credible capability claims, vuln disclosures, insider tips) by name and thread. Roll up routine/repetitive matches (e.g. many github_url or huggingface_url hits) into a brief mention rather than listing each one. If nothing stands out, say so plainly, in one sentence. Do not editorialize beyond what the data supports.`

func buildPrompt(w WindowActivity) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", systemPrompt)
	fmt.Fprintf(&b, "Window: %s (%s to %s)\n\n",
		w.Label, w.PeriodStart.Format(time.RFC3339), w.PeriodEnd.Format(time.RFC3339))

	fmt.Fprintf(&b, "Finding counts by kind:\n")
	for _, kc := range w.ByKind {
		fmt.Fprintf(&b, "- %s: %d\n", kc.Kind, kc.Count)
	}

	if len(w.Findings) > 0 {
		fmt.Fprintf(&b, "\nExample findings (board, kind, thread, match/note):\n")
		for _, f := range w.Findings {
			detail := f.MatchedValue
			if detail == "" {
				detail = f.Note
			}
			fmt.Fprintf(&b, "- /%s/ %s in %q: %s\n", f.Board, f.Kind, f.ThreadSubject, detail)
		}
	}

	if len(w.NewGenerals) > 0 {
		fmt.Fprintf(&b, "\nNew general threads started:\n")
		for _, g := range w.NewGenerals {
			fmt.Fprintf(&b, "- /%s/ %q (%d replies)\n", g.Board, g.Subject, g.Replies)
		}
	}

	if len(w.EndedGenerals) > 0 {
		fmt.Fprintf(&b, "\nGeneral threads that ended (pruned/died):\n")
		for _, g := range w.EndedGenerals {
			fmt.Fprintf(&b, "- /%s/ %q (%d replies)\n", g.Board, g.Subject, g.Replies)
		}
	}

	return b.String()
}
