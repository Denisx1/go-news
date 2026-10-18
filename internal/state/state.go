// Package state хранит, какие статьи уже обработаны и опубликованы.
package state

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"gonews/internal/model"
)

// PublishedPrefix — префикс ключей «уже отправлено на сайт».
// Такие ключи не удаляются по сроку давности (иначе старые новости
// из news_rewritten.json отправились бы повторно).
const PublishedPrefix = "pub:"

const seenTTL = 30 * 24 * time.Hour

type State struct {
	Seen map[string]time.Time `json:"seen"`
	path string
}

func Load(path string) *State {
	st := &State{path: path}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, st)
	}
	if st.Seen == nil {
		st.Seen = map[string]time.Time{}
	}
	return st
}

// IsSeen: статья уже обрабатывалась (проверяем и ID, и ссылку).
func (s *State) IsSeen(a *model.Article) bool {
	if _, ok := s.Seen[a.ID]; ok {
		return true
	}
	if a.Link != "" {
		if _, ok := s.Seen[a.Link]; ok {
			return true
		}
	}
	return false
}

// Mark запоминает статью, чтобы больше её не обрабатывать.
func (s *State) Mark(a *model.Article) {
	now := time.Now()
	s.Seen[a.ID] = now
	if a.Link != "" {
		s.Seen[a.Link] = now
	}
}

func (s *State) IsPublished(id string) bool {
	_, ok := s.Seen[PublishedPrefix+id]
	return ok
}

func (s *State) MarkPublished(id string) {
	s.Seen[PublishedPrefix+id] = time.Now()
}

func (s *State) Save() {
	cutoff := time.Now().Add(-seenTTL)
	for k, t := range s.Seen {
		if !strings.HasPrefix(k, PublishedPrefix) && t.Before(cutoff) {
			delete(s.Seen, k)
		}
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		log.Printf("сохранение состояния: %v", err)
		return
	}
	if err := os.Rename(tmp, s.path); err != nil {
		log.Printf("сохранение состояния: %v", err)
	}
}
