// Package config loads the poller's runtime configuration from the
// environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultInterval = 20 * time.Second
	defaultWorkers  = 2
	defaultBoards   = "g:20s,sci:20s,diy:20s,3:20s"
)

// Board is one watched board and how often to poll its catalog.
type Board struct {
	Name     string
	Interval time.Duration
}

// Config is the poller's full runtime configuration.
type Config struct {
	DatabaseURL string
	Boards      []Board
	Workers     int
}

// Load reads Config from the environment:
//
//	DATABASE_URL   postgres connection string (required)
//	POLLER_BOARDS  comma-separated board:interval pairs, e.g. "g:20s,sci:20s" (default "g:20s,sci:20s,diy:20s,3:20s")
//	POLLER_WORKERS worker pool size (default 2)
func Load() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	boardsEnv := os.Getenv("POLLER_BOARDS")
	if boardsEnv == "" {
		boardsEnv = defaultBoards
	}
	boards, err := parseBoards(boardsEnv)
	if err != nil {
		return Config{}, err
	}

	workers := defaultWorkers
	if v := os.Getenv("POLLER_WORKERS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("POLLER_WORKERS: %w", err)
		}
		workers = n
	}

	return Config{DatabaseURL: dbURL, Boards: boards, Workers: workers}, nil
}

func parseBoards(s string) ([]Board, error) {
	var boards []Board
	for _, part := range strings.Split(s, ",") {
		nameInterval := strings.SplitN(strings.TrimSpace(part), ":", 2)
		name := nameInterval[0]
		interval := defaultInterval
		if len(nameInterval) == 2 {
			d, err := time.ParseDuration(nameInterval[1])
			if err != nil {
				return nil, fmt.Errorf("board %q: bad interval: %w", name, err)
			}
			interval = d
		}
		boards = append(boards, Board{Name: name, Interval: interval})
	}
	return boards, nil
}
