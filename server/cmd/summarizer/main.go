// Command summarizer periodically generates LLM narrative summaries of
// findings/generals activity over trailing hour/day/week windows and
// stores them for the API to serve. It only touches Postgres and
// OpenAI — no 4chan API dependency — so it deploys, scales, and fails
// independently of poller and api, same reasoning as their own split.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jcl80/dredge4us/lib/narrate"
	"github.com/jcl80/dredge4us/server/internal/store"
)

// maxFindingsPerSummary bounds prompt size/cost — a week can have
// hundreds of findings; only this many recent examples go in verbatim,
// the rest are represented by FindingKindCounts' full-window breakdown.
const maxFindingsPerSummary = 40

const defaultModel = "gpt-5.5"

type window struct {
	Label    string // human-readable, goes straight into the LLM prompt
	Name     string // matches narrative_summaries.window
	Interval time.Duration
}

var windows = []window{
	{"the last hour", "hour", time.Hour},
	{"the last 24 hours", "day", 24 * time.Hour},
	{"the last 7 days", "week", 7 * 24 * time.Hour},
}

func main() {
	if err := run(); err != nil {
		slog.Error("summarizer exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = defaultModel
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := store.NewPostgres(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pg.Close()

	summarizer := narrate.NewSummarizer(apiKey, model)

	var wg sync.WaitGroup
	for _, w := range windows {
		wg.Add(1)
		go func(w window) {
			defer wg.Done()
			runWindow(ctx, pg, summarizer, w)
		}(w)
	}

	slog.Info("summarizer starting", "windows", len(windows))
	wg.Wait()
	return nil
}

// runWindow generates immediately on startup, then again every
// w.Interval — same shape as scheduler.watchBoard's first-cycle-then-
// ticker pattern.
func runWindow(ctx context.Context, pg *store.Postgres, summarizer *narrate.Summarizer, w window) {
	generate(ctx, pg, summarizer, w)

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			generate(ctx, pg, summarizer, w)
		}
	}
}

func generate(ctx context.Context, pg *store.Postgres, summarizer *narrate.Summarizer, w window) {
	end := time.Now()
	start := end.Add(-w.Interval)

	findings, err := pg.FindingsSince(ctx, start, maxFindingsPerSummary)
	if err != nil {
		slog.Error("fetch findings for summary failed", "window", w.Name, "error", err)
		return
	}

	byKind, err := pg.FindingKindCounts(ctx, "", start)
	if err != nil {
		slog.Error("fetch kind counts for summary failed", "window", w.Name, "error", err)
		return
	}

	newGenerals, err := pg.NewGenerals(ctx, start)
	if err != nil {
		slog.Error("fetch new generals for summary failed", "window", w.Name, "error", err)
		return
	}

	endedGenerals, err := pg.EndedGenerals(ctx, start)
	if err != nil {
		slog.Error("fetch ended generals for summary failed", "window", w.Name, "error", err)
		return
	}

	activity := narrate.WindowActivity{
		Label:         w.Label,
		PeriodStart:   start,
		PeriodEnd:     end,
		Findings:      toNarrateFindings(findings),
		ByKind:        toNarrateKindCounts(byKind),
		NewGenerals:   toNarrateGenerals(newGenerals),
		EndedGenerals: toNarrateGenerals(endedGenerals),
	}

	text, err := summarizer.Summarize(ctx, activity)
	if err != nil {
		slog.Error("generate narrative summary failed", "window", w.Name, "error", err)
		return
	}

	err = pg.SaveNarrativeSummary(ctx, store.NarrativeSummary{
		Window:       w.Name,
		PeriodStart:  start,
		PeriodEnd:    end,
		FindingCount: activity.TotalFindings(),
		Summary:      text,
	})
	if err != nil {
		slog.Error("save narrative summary failed", "window", w.Name, "error", err)
		return
	}

	slog.Info("narrative summary generated", "window", w.Name, "findings", activity.TotalFindings())
}

func toNarrateFindings(records []store.FindingRecord) []narrate.Finding {
	out := make([]narrate.Finding, len(records))
	for i, f := range records {
		note := ""
		if f.Note != nil {
			note = *f.Note
		}
		out[i] = narrate.Finding{
			Board:         f.Board,
			Kind:          f.Kind,
			MatchedValue:  f.MatchedValue,
			Note:          note,
			ThreadSubject: f.ThreadSubject,
		}
	}
	return out
}

func toNarrateKindCounts(counts []store.KindCount) []narrate.KindCount {
	out := make([]narrate.KindCount, len(counts))
	for i, kc := range counts {
		out[i] = narrate.KindCount{Kind: kc.Kind, Count: kc.Count}
	}
	return out
}

func toNarrateGenerals(rows []store.GeneralActivity) []narrate.GeneralActivity {
	out := make([]narrate.GeneralActivity, len(rows))
	for i, g := range rows {
		out[i] = narrate.GeneralActivity{Board: g.Board, Subject: g.ThreadSubject, Replies: g.Replies}
	}
	return out
}
