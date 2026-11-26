// Package app — основной цикл: сбор → рерайт → публикация.
package app

import (
	"context"
	"log"
	"time"

	"gonews/internal/collector"
	"gonews/internal/config"
	"gonews/internal/rewriter"
	"gonews/internal/site"
	"gonews/internal/sources"
	"gonews/internal/state"
)

type App struct {
	cfg     config.Config
	sources []sources.Source
	col     *collector.Collector
	state   *state.State
	rw      *rewriter.Client // nil — рерайт выключен
	site    *site.Client     // nil — публикация выключена
}

func New(cfg config.Config, srcs []sources.Source, col *collector.Collector,
	st *state.State, rw *rewriter.Client, siteClient *site.Client) *App {
	return &App{cfg: cfg, sources: srcs, col: col, state: st, rw: rw, site: siteClient}
}

// Run запускает цикл и работает до отмены контекста.
func (a *App) Run(ctx context.Context) {
	log.Printf("сервис запущен: источников %d, интервал %s, рерайт: %v, публикация: %v",
		len(a.sources), a.cfg.PollInterval, a.rw != nil, a.site != nil)
	a.runCycle(ctx)

	t := time.NewTicker(a.cfg.PollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("остановлен")
			return
		case <-t.C:
			a.runCycle(ctx)
		}
	}
}

func (a *App) runCycle(ctx context.Context) {
	a.collect(ctx)        // 1. сбор → news.json
	a.rewritePending(ctx) // 2. рерайт → news_rewritten.json
	a.publishPending(ctx) // 3. отправка на cherkasy.biz
}
