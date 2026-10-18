// Package config читает настройки из переменных окружения (и файла .env).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type SiteConfig struct {
	Enabled    bool
	LoginURL   string
	AddURL     string
	UploadURL  string
	Login      string
	Password   string
	ResourceID string // значение поля resource (ID компании)
}

type Config struct {
	// Сбор
	Keywords        []string // пусто = без фильтра
	PollInterval    time.Duration
	FeedWorkers     int
	FetchWorkers    int
	LimitPerSource  int
	OnlyToday       bool
	BrowserHeadless bool

	// Файлы
	DataDir       string
	NewsFile      string
	RewrittenFile string
	StateFile     string

	// Рерайт
	Rewrite      bool
	AnthropicKey string
	Model        string
	Effort       string
	AIWorkers    int

	// Публикация
	Site SiteConfig
}

// Load читает .env (если он есть) и переменные окружения.
func Load() (Config, error) {
	_ = godotenv.Load() // отсутствие .env — не ошибка

	r := &reader{}
	dataDir := r.str("DATA_DIR", "data")

	cfg := Config{
		Keywords:        parseKeywords(r.str("KEYWORDS", "черкас,черкащин")),
		PollInterval:    r.duration("POLL_INTERVAL", 10*time.Minute),
		FeedWorkers:     r.integer("FEED_WORKERS", 5),
		FetchWorkers:    r.integer("FETCH_WORKERS", 5),
		LimitPerSource:  r.integer("LIMIT_PER_SOURCE", 5),
		OnlyToday:       r.boolean("ONLY_TODAY", true),
		BrowserHeadless: r.boolean("BROWSER_HEADLESS", true),

		DataDir:       dataDir,
		NewsFile:      filepath.Join(dataDir, "news.json"),
		RewrittenFile: filepath.Join(dataDir, "news_rewritten.json"),
		StateFile:     filepath.Join(dataDir, "state.json"),

		Rewrite:   r.boolean("REWRITE", true),
		Model:     r.str("ANTHROPIC_MODEL", "claude-opus-5-5"),
		Effort:    r.str("ANTHROPIC_EFFORT", "high"),
		AIWorkers: r.integer("AI_WORKERS", 3),
	}
	if cfg.Rewrite {
		cfg.AnthropicKey = r.required("ANTHROPIC_API_KEY")
	}

	cfg.Site = SiteConfig{
		Enabled:    r.boolean("SITE_ENABLED", true),
		LoginURL:   r.str("SITE_LOGIN_URL", "https://cherkasy.biz/login/"),
		AddURL:     r.str("SITE_ADD_URL", "https://cherkasy.biz/news/add/"),
		UploadURL:  r.str("SITE_UPLOAD_URL", "https://cherkasy.biz/upload/photo-nologo/"),
		ResourceID: r.str("SITE_RESOURCE_ID", "0"),
	}
	if cfg.Site.Enabled {
		cfg.Site.Login = r.required("SITE_LOGIN")
		cfg.Site.Password = r.required("SITE_PASSWORD")
	}
	return cfg, r.err
}

func parseKeywords(v string) []string {
	if strings.TrimSpace(v) == "*" {
		return nil
	}
	var out []string
	for _, k := range strings.Split(v, ",") {
		if k = strings.ToLower(strings.TrimSpace(k)); k != "" {
			out = append(out, k)
		}
	}
	return out
}

// reader собирает значения и запоминает первую ошибку.
type reader struct{ err error }

func (r *reader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
}

func (r *reader) str(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (r *reader) required(k string) string {
	v := os.Getenv(k)
	if v == "" {
		r.fail("не задана переменная %s", k)
	}
	return v
}

func (r *reader) integer(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		r.fail("%s: нужно целое число >= 1", k)
		return def
	}
	return n
}

func (r *reader) boolean(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		r.fail("%s: нужно true или false", k)
		return def
	}
	return b
}

func (r *reader) duration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		r.fail("%s: %v", k, err)
		return def
	}
	return d
}
