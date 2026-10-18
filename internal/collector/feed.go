package collector

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"gonews/internal/model"
	"gonews/internal/sources"
	"gonews/internal/textutil"
)

var reImgSrc = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)

func (c *Collector) fetchFeed(ctx context.Context, src *sources.Source) ([]*model.Article, error) {
	body, err := c.fetch(ctx, src, src.FeedURL, acceptFeed)
	if err != nil {
		return nil, err
	}

	// Парсер создаём на каждый вызов: так он точно безопасен при параллельной работе.
	feed, err := gofeed.NewParser().Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("разбор RSS: %w", err)
	}

	var fullText map[string]string
	if src.FullTextTag != "" {
		fullText = extractFullText(body, src.FullTextTag)
	}

	var out []*model.Article
	for _, it := range feed.Items {
		id := textutil.FirstNonEmpty(it.GUID, it.Link)
		if id == "" {
			continue
		}
		var pub time.Time
		if it.PublishedParsed != nil {
			pub = *it.PublishedParsed
		} else if it.UpdatedParsed != nil {
			pub = *it.UpdatedParsed
		}
		content := it.Content
		if ft := fullText[it.Link]; ft != "" {
			content = ft
		}
		out = append(out, &model.Article{
			Src: src, ID: id, Title: textutil.CleanHTML(it.Title), Link: it.Link,
			Image:     extractImage(it),
			Published: pub, RSSContent: content, RSSDesc: it.Description,
		})
	}
	return out, nil
}

// extractImage ищет картинку: image-тег ленты → enclosure → media:* → первый <img> в HTML.
func extractImage(it *gofeed.Item) string {
	if it.Image != nil && it.Image.URL != "" {
		return it.Image.URL
	}
	for _, e := range it.Enclosures {
		if e.URL != "" && strings.HasPrefix(e.Type, "image/") {
			return e.URL
		}
	}
	if media, ok := it.Extensions["media"]; ok {
		for _, name := range []string{"content", "thumbnail"} {
			for _, ext := range media[name] {
				if u := ext.Attrs["url"]; u != "" && ext.Attrs["medium"] != "video" &&
					!strings.HasPrefix(ext.Attrs["type"], "video/") {
					return u
				}
			}
		}
	}
	for _, h := range []string{it.Content, it.Description} {
		for _, m := range reImgSrc.FindAllStringSubmatch(h, -1) {
			if u := html.UnescapeString(m[1]); !strings.Contains(u, "s.w.org/images/core/emoji") {
				return u
			}
		}
	}
	return ""
}

// extractFullText достаёт текст из нестандартного тега (например <full-text>) по ссылке статьи.
func extractFullText(body []byte, tag string) map[string]string {
	out := map[string]string{}
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.Strict = false

	var inItem bool
	var link, text string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "item":
				inItem, link, text = true, "", ""
			case "link":
				if inItem {
					var s string
					_ = dec.DecodeElement(&s, &t)
					link = strings.TrimSpace(s)
				}
			case tag:
				if inItem {
					var s string
					_ = dec.DecodeElement(&s, &t)
					text = s
				}
			}
		case xml.EndElement:
			if t.Name.Local == "item" {
				if link != "" && text != "" {
					out[link] = text
				}
				inItem = false
			}
		}
	}
	return out
}
