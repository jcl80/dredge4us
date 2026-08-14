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
	defaultInterval    = 20 * time.Second
	defaultWorkers     = 2
	defaultBoards      = "g:20s"
	defaultOpenAIModel = "gpt-5.5"
)

// archiveHosts maps a POLLER_BOARDS source name to the FoolFuuka archive
// base URL it selects. See docs/archive-sources.md for why only these two
// are wired up — 4plebs and Archived.Moe sit behind an active Cloudflare
// challenge that isn't in scope to solve.
var archiveHosts = map[string]string{
	"desuarchive": "https://desuarchive.org",
	"palanq":      "https://archive.palanq.win",
}

// Board is one watched board and how often to poll its catalog. Source is
// "" for live 4chan (a.4cdn.org), or a FoolFuuka archive's base URL when
// the board should be pulled from an archive instead — see
// docs/archive-sources.md. A board polls exactly one of the two, never
// both.
type Board struct {
	Name     string
	Interval time.Duration
	Source   string
}

// Config is the poller's full runtime configuration.
type Config struct {
	DatabaseURL  string
	Boards       []Board
	Workers      int
	OpenAIAPIKey string
	OpenAIModel  string
}

// Load reads Config from the environment:
//
//	DATABASE_URL    postgres connection string (required)
//	POLLER_BOARDS   comma-separated board[:source]:interval entries, e.g.
//	                "his:desuarchive:20s,g:20s" (default "g:20s"). The
//	                source segment is optional and selects a FoolFuuka
//	                archive (see archiveHosts) instead of live 4chan.
//	POLLER_WORKERS  worker pool size (default 2)
//	OPENAI_API_KEY  enables LLM thread classification when set (optional)
//	OPENAI_MODEL    model for classification (default "gpt-5.5")
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

	openAIModel := os.Getenv("OPENAI_MODEL")
	if openAIModel == "" {
		openAIModel = defaultOpenAIModel
	}

	return Config{
		DatabaseURL:  dbURL,
		Boards:       boards,
		Workers:      workers,
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:  openAIModel,
	}, nil
}

func parseBoards(s string) ([]Board, error) {
	var boards []Board
	for _, part := range strings.Split(s, ",") {
		fields := strings.SplitN(strings.TrimSpace(part), ":", 3)
		name := fields[0]
		interval := defaultInterval
		source := ""

		switch len(fields) {
		case 1:
		case 2:
			d, err := time.ParseDuration(fields[1])
			if err != nil {
				return nil, fmt.Errorf("board %q: bad interval: %w", name, err)
			}
			interval = d
		case 3:
			host, ok := archiveHosts[fields[1]]
			if !ok {
				return nil, fmt.Errorf("board %q: unknown source %q", name, fields[1])
			}
			source = host

			d, err := time.ParseDuration(fields[2])
			if err != nil {
				return nil, fmt.Errorf("board %q: bad interval: %w", name, err)
			}
			interval = d
		}

		boards = append(boards, Board{Name: name, Interval: interval, Source: source})
	}
	return boards, nil
}
