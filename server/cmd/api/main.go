// Command api serves the poller's findings as read-only JSON. Kept as a
// separate binary from poller so ingestion and querying can restart,
// scale, and fail independently of each other.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jcl80/dredge4us/server/internal/api"
	pgstore "github.com/jcl80/dredge4us/server/internal/store"
)

const shutdownTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("api exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := pgstore.NewPostgres(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pg.Close()

	srv := &http.Server{Addr: addr, Handler: api.New(pg)}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("api starting", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
