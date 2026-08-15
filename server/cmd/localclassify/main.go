// Command localclassify runs detectors (regex + LLM, if OPENAI_API_KEY
// is set) over raw_posts rows no classify pass has touched yet, against
// the same DATABASE_URL the deployed app uses (repo-root .env) — the
// local-runner counterpart to cmd/localbackfill. Mirrors
// server/internal/api's classifyHandler/runClassify; kept as a
// standalone copy since that logic is unexported from package api.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jcl80/dredge4us/lib/detect"
	"github.com/jcl80/dredge4us/lib/fourchan"
	pgstore "github.com/jcl80/dredge4us/server/internal/store"
)

func main() {
	board := flag.String("board", "", "board to classify (empty = every board)")
	minutes := flag.Int("minutes", 10, "safety time budget (LLM calls aren't individually cancellable mid-call)")
	flag.Parse()

	if err := run(*board, *minutes); err != nil {
		slog.Error("localclassify failed", "error", err)
		os.Exit(1)
	}
}

func run(board string, minutes int) error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required (source .env first)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(minutes)*time.Minute)
	defer cancel()

	pg, err := pgstore.NewPostgres(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pg.Close()

	detectors := []detect.Detector{detect.NewArtifactDetector()}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-5.5"
		}
		detectors = append(detectors, detect.NewLLMClassifier(apiKey, model))
		slog.Info("llm classification enabled", "model", model)
	} else {
		slog.Info("OPENAI_API_KEY not set — regex only")
	}

	raw, err := pg.UnclassifiedRawPosts(ctx, board)
	if err != nil {
		return fmt.Errorf("list unclassified raw posts: %w", err)
	}
	if len(raw) == 0 {
		slog.Info("nothing unclassified", "board", board)
		return nil
	}
	slog.Info("loaded unclassified posts", "board", board, "posts", len(raw))

	type threadKey struct {
		board    string
		threadNo int
	}
	groups := make(map[threadKey][]pgstore.RawPost)
	for _, p := range raw {
		k := threadKey{p.Board, p.ThreadNo}
		groups[k] = append(groups[k], p)
	}

	var findings []detect.Finding
	ids := make([]int64, 0, len(raw))
	done := 0
	for k, group := range groups {
		if ctx.Err() != nil {
			slog.Info("time budget hit, stopping", "threads_done", done, "threads_total", len(groups))
			break
		}

		sub := ""
		fcPosts := make([]fourchan.Post, 0, len(group))
		for _, p := range group {
			if p.Sub != "" {
				sub = p.Sub
			}
			resto := 0
			if p.PostNo != p.ThreadNo {
				resto = p.ThreadNo
			}
			fcPosts = append(fcPosts, fourchan.Post{
				No:       p.PostNo,
				Resto:    resto,
				Time:     p.PostTime.Unix(),
				Sub:      p.Sub,
				Com:      p.Com,
				Sticky:   boolToFlag(p.Sticky),
				Closed:   boolToFlag(p.Closed),
				Archived: boolToFlag(p.Archived),
			})
			ids = append(ids, p.ID)
		}

		th := fourchan.Thread{No: k.threadNo, Sub: sub, Replies: len(group)}
		for _, d := range detectors {
			fs := d.Detect(k.board, th, fcPosts)
			findings = append(findings, fs...)
		}
		done++
		if done%20 == 0 {
			slog.Info("progress", "threads_done", done, "threads_total", len(groups), "findings_so_far", len(findings))
		}
	}

	saveCtx, saveCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer saveCancel()
	if err := pg.SaveFindings(saveCtx, findings); err != nil {
		return fmt.Errorf("save findings: %w", err)
	}
	if err := pg.MarkClassified(saveCtx, ids); err != nil {
		return fmt.Errorf("mark classified: %w", err)
	}

	slog.Info("done", "board", board, "posts", len(raw), "threads", done, "findings", len(findings))
	return nil
}

func boolToFlag(b bool) int {
	if b {
		return 1
	}
	return 0
}
