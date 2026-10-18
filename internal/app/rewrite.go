package app

import (
	"context"
	"log"
	"strings"

	"gonews/internal/model"
	"gonews/internal/parallel"
	"gonews/internal/store"
)

// rewritePending переписывает всё из news.json, чего ещё нет в news_rewritten.json.
func (a *App) rewritePending(ctx context.Context) {
	if a.rw == nil {
		return
	}
	src, err := store.Read(a.cfg.NewsFile)
	if err != nil {
		log.Printf("рерайт: чтение %s: %v", a.cfg.NewsFile, err)
		return
	}
	done, err := store.Read(a.cfg.RewrittenFile)
	if err != nil {
		log.Printf("рерайт: файл %s битый, не трогаю: %v", a.cfg.RewrittenFile, err)
		return
	}
	have := make(map[string]bool, len(done))
	for _, it := range done {
		have[it.ID] = true
	}
	var pending []model.NewsItem
	for _, it := range src {
		if !have[it.ID] && strings.TrimSpace(it.Description) != "" {
			pending = append(pending, it)
		}
	}
	if len(pending) == 0 {
		return
	}
	log.Printf("🤖 к рерайту: %d", len(pending))

	results := make([]string, len(pending))
	parallel.ForEach(ctx, a.cfg.AIWorkers, len(pending), func(i int) {
		out, err := a.rw.Rewrite(ctx, pending[i].Description)
		if err != nil {
			log.Printf("  рерайт %q: %v (повторю в следующем цикле)", pending[i].Title, err)
			return
		}
		results[i] = out
	})

	added := 0
	for i, it := range pending {
		if results[i] == "" {
			continue
		}
		it.Description = results[i] // title, image, link и т.д. остаются как были
		done = append(done, it)
		added++
	}
	if added == 0 {
		return
	}
	if err := store.Write(a.cfg.RewrittenFile, done); err != nil {
		log.Printf("рерайт: запись %s: %v", a.cfg.RewrittenFile, err)
		return
	}
	log.Printf("✅ переписано: %d → %s (всего: %d)", added, a.cfg.RewrittenFile, len(done))
}
