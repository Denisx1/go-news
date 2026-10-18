// Package model — общие типы данных.
package model

import (
	"time"

	"gonews/internal/sources"
)

// Article — статья в процессе обработки.
type Article struct {
	Src        *sources.Source
	ID         string
	Title      string
	Link       string
	Image      string
	Published  time.Time
	RSSContent string
	RSSDesc    string
	Text       string
}

// NewsItem — запись в JSON-файлах (news.json, news_rewritten.json).
type NewsItem struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Title       string `json:"title"`
	Link        string `json:"link,omitempty"`
	Image       string `json:"image,omitempty"`
	Description string `json:"description"`
	Published   string `json:"published,omitempty"` // RFC3339
}
