package collector

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"gonews/internal/model"
	"gonews/internal/sources"
	"gonews/internal/textutil"
)

func pick(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

// fetchHTMLList разбирает страницу со списком новостей (для сайтов без RSS).
func (c *Collector) fetchHTMLList(ctx context.Context, src *sources.Source) ([]*model.Article, error) {
	body, err := c.fetch(ctx, src, src.FeedURL, acceptHTML)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("разбор HTML: %w", err)
	}
	base, err := url.Parse(src.FeedURL)
	if err != nil {
		return nil, err
	}

	titleSel := pick(src.ItemTitleSel, ".news-item-title")
	dateSel := pick(src.ItemDateSel, ".news-item-meta")
	descSel := pick(src.ItemDescSel, ".news-item-description")
	imgSel := pick(src.ItemImgSel, ".news-item-image img")

	var out []*model.Article
	seen := map[string]bool{}
	doc.Find(src.ListSelector).Each(func(_ int, s *goquery.Selection) {
		// ссылка: сам элемент или первая <a> внутри
		link := s
		if goquery.NodeName(s) != "a" {
			link = s.Find("a[href]").First()
		}
		href, ok := link.Attr("href")
		if !ok {
			return
		}
		u, err := base.Parse(href)
		if err != nil {
			return
		}
		l := u.String()

		title := textutil.CleanHTML(s.Find(titleSel).First().Text())
		if title == "" {
			title = textutil.CleanHTML(link.Text())
		}
		if title == "" || seen[l] {
			return
		}
		seen[l] = true

		art := &model.Article{
			Src: src, ID: l, Title: title, Link: l,
			Published: parseDMY(s.Find(dateSel).First().Text()),
			RSSDesc:   s.Find(descSel).First().Text(),
		}
		if img, ok := s.Find(imgSel).First().Attr("src"); ok && img != "" {
			if iu, err := base.Parse(img); err == nil {
				art.Image = iu.String()
			}
		}
		out = append(out, art)
	})
	if len(out) == 0 {
		return nil, fmt.Errorf("по селектору %q ничего не найдено", src.ListSelector)
	}
	return out, nil
}

// fetchPageText скачивает страницу статьи и собирает текст по CSS-селектору.
func (c *Collector) fetchPageText(ctx context.Context, src *sources.Source, pageURL, selector string) (string, error) {
	if selector == "" {
		selector = "article p"
	}
	body, err := c.fetch(ctx, src, pageURL, acceptHTML)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		name := goquery.NodeName(s)
		// абзац внутри пункта списка уже входит в текст самого пункта
		if name == "p" && s.ParentsFiltered("li").Length() > 0 {
			return
		}
		t := textutil.StripTags(s.Text())
		if t == "" {
			return
		}
		if name == "li" {
			t = "• " + t
		}
		sb.WriteString(t + "\n\n")
	})
	res := strings.TrimSpace(sb.String())
	if res == "" {
		return "", fmt.Errorf("текст по селектору %q не найден", selector)
	}
	return res, nil
}
