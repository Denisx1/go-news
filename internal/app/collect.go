package app

import (
	"context"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"gonews/internal/collector"
	"gonews/internal/model"
	"gonews/internal/parallel"
	"gonews/internal/store"
)

// collect: опрос лент (параллельно) → отбор новых → скачивание текстов (параллельно)
// → фильтр по теме → запись в news.json.
func (a *App) collect(ctx context.Context) {
	collected, ok := a.fetchAll(ctx)
	fresh := a.pickFresh(collected, ok)
	a.state.Save()

	if len(fresh) == 0 {
		log.Println("новых статей нет")
		return
	}
	log.Printf("✨ новых статей: %d", len(fresh))

	parallel.ForEach(ctx, a.cfg.FetchWorkers, len(fresh), func(i int) {
		fresh[i].Text = a.col.ArticleText(ctx, fresh[i])
	})

	fresh = a.filterRelevant(fresh)
	if len(fresh) == 0 {
		log.Println("после фильтров новых статей нет")
		return
	}
	a.saveToFile(fresh)
}

func (a *App) fetchAll(ctx context.Context) ([][]*model.Article, []bool) {
	collected := make([][]*model.Article, len(a.sources))
	ok := make([]bool, len(a.sources))
	parallel.ForEach(ctx, a.cfg.FeedWorkers, len(a.sources), func(i int) {
		src := &a.sources[i]
		log.Printf("📥 опрашиваю: %s", src.Name)
		arts, err := a.col.FetchSource(ctx, src)
		if err != nil {
			log.Printf("  [ошибка] %s: %v", src.Name, err)
			return
		}
		collected[i], ok[i] = arts, true
	})
	return collected, ok
}

// pickFresh отбирает ещё не обработанные статьи (состояние трогаем только здесь).
func (a *App) pickFresh(collected [][]*model.Article, ok []bool) []*model.Article {
	var fresh []*model.Article
	picked := map[string]bool{} // защита от одной и той же статьи в разных лентах
	for i := range a.sources {
		if !ok[i] {
			continue
		}
		count := 0
		for _, art := range collected[i] {
			if a.state.IsSeen(art) || picked[art.ID] || (art.Link != "" && picked[art.Link]) {
				continue
			}
			if a.cfg.OnlyToday && !art.Published.IsZero() && !collector.IsToday(art.Published) {
				continue
			}
			if count >= a.cfg.LimitPerSource {
				break // остальное подберём в следующем цикле
			}
			fresh = append(fresh, art)
			picked[art.ID] = true
			if art.Link != "" {
				picked[art.Link] = true
			}
			count++
		}
	}
	return fresh
}

// filterRelevant оставляет статьи с ключевым словом в заголовке или тексте.
// Неподходящие помечаются обработанными, чтобы не скачивать их снова каждый цикл.
func (a *App) filterRelevant(arts []*model.Article) []*model.Article {
	var out []*model.Article
	for _, art := range arts {
		if art.Src.ExcludeTitle != "" {
			if re, err := regexp.Compile(art.Src.ExcludeTitle); err != nil {
				log.Printf("ExcludeTitle %q: %v", art.Src.ExcludeTitle, err)
			} else if re.MatchString(art.Title) {
				a.state.Mark(art)
				log.Printf("исключено по заголовку: %q", art.Title)
				continue
			}
		}

		if len(a.cfg.Keywords) == 0 || art.Src.SkipKeywords {
			out = append(out, art)
			continue
		}
		hay := strings.ToLower(art.Title + "\n" + art.Text)
		matched := ""
		for _, k := range a.cfg.Keywords {
			if strings.Contains(hay, k) {
				matched = k
				break
			}
		}
		if matched == "" {
			if art.Text != "" {
				a.state.Mark(art)
			}
			log.Printf("не по теме, пропуск: %q", art.Title)
			continue
		}
		log.Printf("по теме (%q): %q", matched, art.Title)
		out = append(out, art)
	}
	a.state.Save()
	return out
}

// saveToFile дописывает новые статьи в news.json.
func (a *App) saveToFile(arts []*model.Article) {
	sort.SliceStable(arts, func(i, j int) bool { return arts[i].Published.Before(arts[j].Published) })

	// Если файл битый — не трогаем его, чтобы не потерять данные.
	items, err := store.Read(a.cfg.NewsFile)
	if err != nil {
		log.Printf("файл %s не является корректным JSON: %v", a.cfg.NewsFile, err)
		return
	}
	exists := make(map[string]bool, len(items))
	for _, it := range items {
		exists[it.ID] = true
	}

	var saved []*model.Article
	for _, art := range arts {
		if art.Text == "" {
			log.Printf("пропуск %q: не удалось получить текст (повторю в следующем цикле)", art.Title)
			continue
		}
		if exists[art.ID] {
			saved = append(saved, art) // уже в файле — просто отметим как обработанную
			continue
		}
		item := model.NewsItem{
			ID:          art.ID,
			Source:      art.Src.Name,
			Title:       art.Title,
			Link:        art.Link,
			Image:       art.Image,
			Description: art.Text,
		}
		if !art.Published.IsZero() {
			item.Published = art.Published.UTC().Format(time.RFC3339)
		}
		items = append(items, item)
		exists[art.ID] = true
		saved = append(saved, art)
	}
	if len(saved) == 0 {
		return
	}
	if err := store.Write(a.cfg.NewsFile, items); err != nil {
		log.Printf("запись %s: %v", a.cfg.NewsFile, err)
		return
	}

	for _, art := range saved {
		a.state.Mark(art) // отмечаем только после успешной записи
	}
	a.state.Save()
	log.Printf("💾 записано новостей: %d → %s (всего в файле: %d)", len(saved), a.cfg.NewsFile, len(items))
}
