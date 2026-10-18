package collector

import (
	"context"
	"log"
	"regexp"

	"gonews/internal/model"
	"gonews/internal/textutil"
)

// ArticleText возвращает полный очищенный текст статьи.
func (c *Collector) ArticleText(ctx context.Context, art *model.Article) string {
	text := ""
	if art.Src.FetchFullPage && art.Link != "" {
		txt, err := c.fetchPageText(ctx, art.Src, art.Link, art.Src.HTMLSelector)
		if err == nil {
			text = txt
		} else {
			log.Printf("  ⚠️ страница %q: %v — беру текст из RSS", art.Title, err)
		}
	}
	if text == "" {
		text = textutil.FirstNonEmpty(textutil.CleanHTML(art.RSSContent), textutil.CleanHTML(art.RSSDesc))
	}
	for _, pat := range art.Src.CleanRegexes {
		re, err := regexp.Compile(pat)
		if err != nil {
			log.Printf("CleanRegexes %q: %v", pat, err)
			continue
		}
		text = re.ReplaceAllString(text, "")
	}
	return textutil.Truncate(textutil.CollapseBlank(text), maxArticleChars)
}
