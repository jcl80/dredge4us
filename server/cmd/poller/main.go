// Command poller runs the live catalog-polling loop against local
// Postgres. See the README for what it deliberately does not do.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jcl80/dredge4us/lib/detect"
	"github.com/jcl80/dredge4us/lib/fourchan"
	"github.com/jcl80/dredge4us/server/internal/config"
	"github.com/jcl80/dredge4us/server/internal/migrate"
	"github.com/jcl80/dredge4us/server/internal/scheduler"
	pgstore "github.com/jcl80/dredge4us/server/internal/store"
	"github.com/jcl80/dredge4us/server/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("poller exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := pgstore.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pg.Close()

	if err := migrate.Run(ctx, pg.Pool(), migrations.Files); err != nil {
		return err
	}

	limiter := fourchan.NewLimiter()
	client := fourchan.NewClient(limiter)

	sched := &scheduler.Scheduler{
		Client:    client,
		Store:     pg,
		Detectors: []detect.Detector{detect.NewArtifactDetector()},
		Boards:    cfg.Boards,
		Workers:   cfg.Workers,
	}

	slog.Info("poller starting", "boards", len(cfg.Boards), "workers", cfg.Workers)
	sched.Run(ctx)

	return nil
}
