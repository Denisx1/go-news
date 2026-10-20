// Package rewriter делает рерайт новостей через Anthropic API.
package rewriter

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gonews/internal/parallel"
	"gonews/internal/textutil"
)

const (
	apiURL      = "https://api.anthropic.com/v1/messages"
	maxAttempts = 3
	maxTokens   = 8000 // с запасом: размышления тоже расходуют лимит
)

//go:embed prompt.txt
var systemPrompt string

type Client struct {
	http   *http.Client
	key    string
	model  string
	effort string
}

func New(client *http.Client, key, model, effort string) *Client {
	return &Client{http: client, key: key, model: model, effort: effort}
}

type apiError struct {
	status int
	wait   time.Duration // из заголовка Retry-After
	body   string
}

func (e *apiError) Error() string { return fmt.Sprintf("anthropic %d: %s", e.status, e.body) }

// Rewrite переписывает текст; при временных ошибках (429, 5xx) повторяет запрос.
func (c *Client) Rewrite(ctx context.Context, text string) (string, error) {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var out string
		if out, err = c.request(ctx, text); err == nil {
			return out, nil
		}
		wait := time.Duration(attempt) * 2 * time.Second
		var ae *apiError
		if errors.As(err, &ae) {
			if ae.status != http.StatusTooManyRequests && ae.status < 500 {
				return "", err // 400/401/403: повтор бессмысленен
			}
			if ae.wait > 0 {
				wait = ae.wait
			}
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		log.Printf("  рерайт, попытка %d: %v", attempt, err)
		if attempt < maxAttempts {
			parallel.Sleep(ctx, wait)
		}
	}
	return "", err
}

func (c *Client) request(ctx context.Context, text string) (string, error) {

	systemBlocks := []map[string]any{
		{
			"type": "text",
			"text": strings.TrimSpace(systemPrompt),
			"cache_control": map[string]any{
				"type": "ephemeral", // Включает кэширование этой части
			},
		},
	}

	body, _ := json.Marshal(map[string]any{
		"model":         c.model,
		"max_tokens":    maxTokens,
		"system":        systemBlocks, // Передаем массив вместо строки
		"output_config": map[string]string{"effort": c.effort},
		"messages":      []map[string]string{{"role": "user", "content": text}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.key)
	req.Header.Set("anthropic-version", "2023-06-01")

	// 2. ОБЯЗАТЕЛЬНЫЙ заголовок для работы Prompt Caching
	req.Header.Set("anthropic-beta", "prompt-caching-2024-07-28")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		e := &apiError{status: resp.StatusCode, body: textutil.Truncate(string(raw), 300)}
		if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
			e.wait = time.Duration(s) * time.Second
		}
		return "", e
	}

	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		// (Опционально) Можете добавить структуру usage, чтобы логировать экономию токенов:
		// Usage struct {
		//     CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		//     CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		// } `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, part := range out.Content {
		if part.Type == "text" {
			sb.WriteString(part.Text)
		}
	}
	res := strings.TrimSpace(sb.String())
	if res == "" {
		return "", fmt.Errorf("пустой ответ от ИИ")
	}
	return res, nil
}
