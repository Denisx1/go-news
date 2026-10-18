// Package textutil — очистка и обрезка текста.
package textutil

import (
	"html"
	"regexp"
	"strings"
)

var (
	reScript = regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	reBlock  = regexp.MustCompile(`(?i)</?(p|div|br|h[1-6]|li)\b[^>]*>`)
	// только настоящие теги, чтобы не съедать "a < b"
	reTags  = regexp.MustCompile(`(?is)</?[a-z!][^>]*>`)
	reBlank = regexp.MustCompile(`\n\s*\n`)
	reStray = regexp.MustCompile(`(?is)</?[a-z][^>]*>`)
)

// CleanHTML превращает HTML (в том числе многократно экранированный) в чистый текст.
func CleanHTML(s string) string {
	// до 3 проходов: &amp;lt;p&amp;gt; → &lt;p&gt; → <p> → убран
	for i := 0; i < 3; i++ {
		prev := s
		s = reScript.ReplaceAllString(s, "")
		s = reStyle.ReplaceAllString(s, "")
		s = reBlock.ReplaceAllString(s, "\n")
		s = reTags.ReplaceAllString(s, "")
		s = html.UnescapeString(s)
		if s == prev {
			break
		}
	}
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = reBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// StripTags убирает остатки HTML-тегов (например <img ... />).
func StripTags(s string) string {
	return strings.TrimSpace(reStray.ReplaceAllString(s, ""))
}

// CollapseBlank схлопывает пустые строки и обрезает пробелы по краям.
func CollapseBlank(s string) string {
	return strings.TrimSpace(reBlank.ReplaceAllString(s, "\n\n"))
}

func FirstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// Truncate обрезает строку до n символов (не байт).
func Truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
