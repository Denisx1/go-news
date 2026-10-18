package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gonews/internal/app"
	"gonews/internal/collector"
	"gonews/internal/config"
	"gonews/internal/features"
	"gonews/internal/rewriter"
	"gonews/internal/site"
	"gonews/internal/sources"
	"gonews/internal/state"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if loc, err := time.LoadLocation("Europe/Kyiv"); err == nil {
		time.Local = loc
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("конфиг: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("папка данных: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srcs := sources.All()
	httpClient := &http.Client{Timeout: 90 * time.Second}

	// Chrome запускаем только если он нужен хотя бы одному источнику.
	var browser collector.Browser
	for _, s := range srcs {
		if s.UseBrowser {
			b, err := features.NewBrowser(ctx, cfg.BrowserHeadless)
			if err != nil {
				return fmt.Errorf("Chrome: %w", err)
			}
			defer b.Close()
			browser = b
			break
		}
	}

	var rw *rewriter.Client
	if cfg.Rewrite {
		rw = rewriter.New(httpClient, cfg.AnthropicKey, cfg.Model, cfg.Effort)
	}
	var st *site.Client
	if cfg.Site.Enabled {
		st = site.New(cfg.Site)
	}

	a := app.New(cfg, srcs, collector.New(httpClient, browser), state.Load(cfg.StateFile), rw, st)
	a.Run(ctx)
	return nil
}
