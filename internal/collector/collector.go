// Package collector загружает ленты и страницы новостей.
package collector

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gonews/internal/httpx"
	"gonews/internal/model"
	"gonews/internal/sources"
)

const (
	maxArticleChars = 8000
	maxBodyBytes    = 5 << 20

	acceptFeed = "application/rss+xml, application/xml, text/xml;q=0.9, */*;q=0.8"
	acceptHTML = "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8"
)

// Browser — загрузка страниц через настоящий Chrome (для сайтов с защитой от ботов).
type Browser interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

type Collector struct {
	http    *http.Client
	browser Browser // может быть nil
}

func New(client *http.Client, browser Browser) *Collector {
	return &Collector{http: client, browser: browser}
}

// FetchSource возвращает список статей источника (RSS или HTML-список).
func (c *Collector) FetchSource(ctx context.Context, src *sources.Source) ([]*model.Article, error) {
	if src.ListSelector != "" {
		return c.fetchHTMLList(ctx, src)
	}
	return c.fetchFeed(ctx, src)
}

// fetch выбирает способ загрузки: через браузер или обычный HTTP.
func (c *Collector) fetch(ctx context.Context, src *sources.Source, u, accept string) ([]byte, error) {
	if src.UseBrowser && c.browser != nil {
		return c.browser.Fetch(ctx, u)
	}
	return c.get(ctx, u, accept)
}

func (c *Collector) get(ctx context.Context, u, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", httpx.UserAgent)
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Language", "uk-UA,uk;q=0.9,en;q=0.8")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return nil, fmt.Errorf("HTTP %d (server=%q, cf-ray=%q): %s",
			resp.StatusCode, resp.Header.Get("Server"), resp.Header.Get("Cf-Ray"),
			strings.Join(strings.Fields(string(snippet)), " "))
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
}
