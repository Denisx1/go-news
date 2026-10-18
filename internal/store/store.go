// Package store читает и пишет JSON-файлы с новостями.
package store

import (
	"bytes"
	"encoding/json"
	"os"

	"gonews/internal/model"
)

// Read возвращает записи из файла. Нет файла или он пуст — пустой список.
func Read(path string) ([]model.NewsItem, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, nil
	}
	var items []model.NewsItem
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Write атомарно записывает записи: сначала во временный файл, затем rename.
func Write(path string, items []model.NewsItem) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // не превращать <, >, & в \u003c
	enc.SetIndent("", "  ")
	if err := enc.Encode(items); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
