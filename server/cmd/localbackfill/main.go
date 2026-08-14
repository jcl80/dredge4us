// Command localbackfill is a one-shot, time-boxed pull of a single
// board's recent history via an archive's search API, run from a
// developer machine against the same DATABASE_URL the deployed app
// uses (see repo-root .env) — for a quick local test without waiting
// on a redeploy. Stores raw post text only, same as /debug/backfill;
// pair with a /debug/classify call (or another local run) to detect.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jcl80/dredge4us/lib/foolfuuka"
	"github.com/jcl80/dredge4us/lib/fourchan"
	"github.com/jcl80/dredge4us/server/internal/migrate"
	pgstore "github.com/jcl80/dredge4us/server/internal/store"
	"github.com/jcl80/dredge4us/server/migrations"
)

// archiveHosts mirrors server/internal/config's table — kept separate
// since this is a standalone dev tool, not part of the app's config.
var archiveHosts = map[string]string{
	"desuarchive": "https://desuarchive.org",
	"palanq":      "https://archive.palanq.win",
}

func main() {
	board := flag.String("board", "g", "board to pull")
	source := flag.String("source", "desuarchive", "archive: desuarchive or palanq")
	minutes := flag.Int("minutes", 2, "time budget")
	flag.Parse()

	if err := run(*board, *source, *minutes); err != nil {
		slog.Error("localbackfill failed", "error", err)
		os.Exit(1)
	}
}

func run(board, source string, minutes int) error {
	host, ok := archiveHosts[source]
	if !ok {
		return fmt.Errorf("unknown source %q", source)
	}

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

	if err := migrate.Run(ctx, pg.Pool(), migrations.Files); err != nil {
		return err
	}

	client := foolfuuka.NewClient(host, fourchan.NewLimiter())

	var posts []pgstore.RawPost
	fetched := 0
	page := 1
	for {
		if ctx.Err() != nil {
			slog.Info("time budget hit, stopping", "page", page)
			break
		}

		fcPosts, totalFound, err := client.Search(ctx, board, page)
		if err != nil {
			slog.Error("search failed", "page", page, "error", err)
			break
		}
		if len(fcPosts) == 0 {
			slog.Info("no more results", "page", page)
			break
		}

		for _, p := range fcPosts {
			threadNo := p.Resto
			if threadNo == 0 {
				threadNo = p.No
			}
			posts = append(posts, pgstore.RawPost{
				Board:    board,
				Source:   host,
				ThreadNo: threadNo,
				PostNo:   p.No,
				PostTime: p.PostTime(),
				Sub:      p.Sub,
				Com:      p.Com,
				Sticky:   p.Sticky != 0,
				Closed:   p.Closed != 0,
				Archived: p.Archived != 0,
			})
		}

		fetched += len(fcPosts)
		slog.Info("page fetched", "page", page, "posts", len(fcPosts), "total_so_far", fetched, "total_found", totalFound)
		page++
		if fetched >= totalFound {
			break
		}
	}

	// Independent of ctx — same reason as the deployed backfill: don't
	// let an expired time budget also kill the save of what we already
	// have.
	saveCtx, saveCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer saveCancel()
	if err := pg.SaveRawPosts(saveCtx, posts); err != nil {
		return fmt.Errorf("save raw posts: %w", err)
	}

	slog.Info("done", "board", board, "source", source, "posts_saved", len(posts))
	return nil
}
