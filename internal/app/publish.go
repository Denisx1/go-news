package app

import (
	"context"
	"log"
	"time"

	"gonews/internal/parallel"
	"gonews/internal/store"
)

// publishPending отправляет на сайт всё из news_rewritten.json, что ещё не отправлено.
func (a *App) publishPending(ctx context.Context) {
	if a.site == nil {
		return
	}
	items, err := store.Read(a.cfg.NewsFile)
	if err != nil {
		log.Printf("публикация: чтение %s: %v", a.cfg.RewrittenFile, err)
		return
	}
	for _, it := range items {
		if ctx.Err() != nil {
			return
		}
		if a.state.IsPublished(it.ID) {
			continue
		}
	
		if err := a.site.Publish(ctx, it); err != nil {
			log.Printf("публикация %q: %v (повторю в следующем цикле)", it.Title, err)
			continue
		}
		a.state.MarkPublished(it.ID)
		a.state.Save()
		log.Printf("[✅] отправлено на cherkasy.biz: %q", it.Title)
		parallel.Sleep(ctx, 3*time.Second)
	}
}
