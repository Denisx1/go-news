// Package sources — список сайтов, откуда собираем новости.
package sources

// Source описывает один сайт.
type Source struct {
	Name          string
	FeedURL       string
	FetchFullPage bool   // false — текст из RSS, true — качать страницу статьи
	HTMLSelector  string // CSS-селектор абзацев (если FetchFullPage == true)
	ListSelector  string // CSS-селектор элементов списка новостей (если нет RSS)
	UseBrowser    bool   // true — запросы через настоящий Chrome (сайты с защитой от ботов)
	// CleanRegexes — регулярки, совпадения которых вырезаются из текста.
	// (?m) делает ^ и $ границами строк.
	CleanRegexes []string
	SkipKeywords bool   // true — не применять фильтр KEYWORDS для этого сайта
	ExcludeTitle string // регулярка: совпадение с заголовком исключает статью
	FullTextTag  string // тег RSS с полным текстом (если не content:encoded)
	ItemTitleSel string // заголовок внутри карточки (по умолчанию .news-item-title)
	ItemDateSel  string // дата (по умолчанию .news-item-meta)
	ItemDescSel  string // описание (по умолчанию .news-item-description)
	ItemImgSel   string // картинка (по умолчанию .news-item-image img)
	JSONAPI      string // "theatre" — источник это JSON API, а не RSS/HTML
	LinkTemplate string // шаблон ссылки на статью, %s подставляется slug
}

// All — 🚀 сюда добавляй новые сайты.
func All() []Source {
	return []Source{
		{
			Name:    "Черкаський театр ім. Шевченка",
			FeedURL: "https://theatre.pp.ua/api/posts?limit=20&page=1&locale=uk",

			LinkTemplate:  "https://theatre-shevchenko.ck.ua/uk/news/%s",
			FetchFullPage: false,
			SkipKeywords:  true,
			UseBrowser:    true,
		},
		{
			Name:          "ГУ ДПС у Черкаській області",
			FeedURL:       "https://ck.tax.gov.ua/media-ark/news-ark/",
			ListSelector:  ".news__item",
			ItemTitleSel:  ".news__title",
			ItemDateSel:   ".shortnews__date",
			ItemImgSel:    ".news__image img",
			FetchFullPage: true,
			HTMLSelector:  "main.news .container__body p, main.news .container__body li",
			SkipKeywords:  true,
			UseBrowser:    true,
		},
		{
			Name:          "Черкаський обласний центр зайнятості",
			FeedURL:       "https://chk.dcz.gov.ua/rss.xml",
			FetchFullPage: false,
			SkipKeywords:  true, // сайт региональный
		},
		{
			Name:          "Черкасиобленерго",
			FeedURL:       "https://www.cherkasyoblenergo.com/news?lang=uk",
			ListSelector:  ".media-news-item",
			ItemTitleSel:  ".media-news-title",
			ItemDateSel:   ".media-news-date",
			ItemImgSel:    ".media-news-thumb img",
			FetchFullPage: true,
			HTMLSelector:  ".post-article-body p, .post-article-body li, .post-article-body h2, .post-article-body h3",
			SkipKeywords:  true, // компания и так про Черкащину
		},
		{
			Name:          "Міністерство економіки України",
			FeedURL:       "https://me.gov.ua/News/List?lang=uk-UA&tag=News",
			ListSelector:  ".news-item",
			FetchFullPage: false,
			SkipKeywords:  false,
		},
		{
			Name:          "Успіх in UA",
			FeedURL:       "https://uspih.in.ua/feed",
			FetchFullPage: false,
			CleanRegexes: []string{
				`(?m)^Запис\s.*вперше опубліковано на.*$`,
				`(?m)^Читайте також:.*$`,
			},
		},
		{
			Name:          "MVS (Головний сервісний центр)",
			FeedURL:       "https://vin.hsc.gov.ua/feed/",
			FetchFullPage: true,
			HTMLSelector:  ".entry-content p, .entry-content li, .entry-content h2, .entry-content h3",
			UseBrowser:    true, // у сайта Akamai: обычные запросы получают 403
		},
		{
			Name:          "Черкаська міська рада",
			FeedURL:       "https://cherkasy-rada.gov.ua/feed/",
			FetchFullPage: false,
			SkipKeywords:  true, // сайт и так про Черкаси
			// ExcludeTitle: `(?i)^Міські новини \(ОПЕРАТИВНО\)`, // ежедневные списки отключений
			CleanRegexes: []string{
				`(?m)^The post .* appeared first on .*$`,
			},
		},
		{
			Name:          "Кам'янська громада",
			FeedURL:       "https://gromada.org.ua/rss/17283/",
			FetchFullPage: false,
			FullTextTag:   "full-text",
			SkipKeywords:  true,
		},
		{
			Name:          "Центр громадського моніторингу та контролю",
			FeedURL:       "https://naglyad.org/uk/feed/",
			FetchFullPage: false,
			SkipKeywords:  false,
			CleanRegexes: []string{
				`(?m)^The post .* appeared first on .*$`,
				`(?m)^Запис\s.*вперше опубліковано на.*$`,
			},
		},
	}
}
