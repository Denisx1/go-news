package features

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// Browser — один настоящий Chrome для сайтов с защитой от ботов (Akamai и т.п.).
// Запросы выполняются через fetch() внутри страницы того же сайта, поэтому
// используются настоящий TLS-отпечаток и cookies браузера.
//
// Написано под chromedp v0.20+ (API на дженериках: Do / Run / Call).
type Browser struct {
	mu      sync.Mutex // запросы через браузер идут по одному
	ctx     context.Context
	warm    map[string]bool // сайты, на которых уже открыта главная
	closers []func()
}

func NewBrowser(parent context.Context, headless bool) (*Browser, error) {
	// Копия стандартных опций (глобальный список менять нельзя).
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)

	opts = append(opts,
		chromedp.Flag("enable-automation", false), // убираем признак «браузером управляет программа»
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)
	if p := findBrowser(); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	} else {
		return nil, fmt.Errorf("не найден Brave/Chrome/Chromium: укажите BROWSER_PATH")
	}
	if !headless {
		opts = append(opts, chromedp.VisibleWindow)
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	fail := func(err error) (*Browser, error) {
		cancelCtx()
		cancelAlloc()
		return nil, err
	}

	// Запускаем браузер (без таймаута: он привязан к этому контексту).
	if err := chromedp.Do(ctx); err != nil {
		return fail(fmt.Errorf("не удалось запустить Chrome (он установлен?): %w", err))
	}

	// Убираем слово "Headless" из User-Agent.
	ua, err := chromedp.Run(ctx, chromedp.Evaluate[string](`navigator.userAgent`))
	if err != nil {
		return fail(err)
	}
	ua = strings.ReplaceAll(ua, "HeadlessChrome", "Chrome")
	if _, err := chromedp.Call(ctx, emulation.SetUserAgentOverride,
		emulation.SetUserAgentOverrideParams{UserAgent: ua}); err != nil {
		return fail(err)
	}

	return &Browser{
		ctx:     ctx,
		warm:    map[string]bool{},
		closers: []func(){cancelCtx, cancelAlloc},
	}, nil
}

func (b *Browser) Close() {
	for _, c := range b.closers {
		c()
	}
}

// Fetch скачивает URL через браузер и возвращает тело ответа.
func (b *Browser) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	origin := u.Scheme + "://" + u.Host

	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.warm[origin] {
		if err := b.warmUp(origin); err != nil {
			return nil, err
		}
	}

	// Таймаут только на этот запрос: браузер и вкладка остаются открытыми.
	runCtx, cancel := context.WithTimeout(b.ctx, 45*time.Second)
	defer cancel()
	go func() { // остановка сервиса прерывает запрос
		select {
		case <-ctx.Done():
			cancel()
		case <-runCtx.Done():
		}
	}()

	jsURL, _ := json.Marshal(rawURL)
	js := fmt.Sprintf(`fetch(%s, {credentials: "include"}).then(r => r.ok ? r.text() : Promise.reject("HTTP " + r.status))`, jsURL)

	res, err := chromedp.Run(runCtx, chromedp.Evaluate[string](js, chromedp.EvalAwaitPromise))
	if err != nil {
		delete(b.warm, origin) // сессия могла протухнуть — в следующий раз откроем главную заново
		return nil, err
	}
	return []byte(res), nil
}

// warmUp открывает главную страницу сайта, чтобы защита выдала браузеру свои cookies.
func (b *Browser) warmUp(origin string) error {
	ctx, cancel := context.WithTimeout(b.ctx, 60*time.Second)
	defer cancel()

	if err := chromedp.Do(ctx,
		chromedp.Navigate(origin+"/"),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	title, err := chromedp.Run(ctx, chromedp.Title())
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(title), "access denied") {
		return fmt.Errorf("сайт заблокировал и браузер (заголовок страницы: %q)", title)
	}
	b.warm[origin] = true
	return nil
}

func findBrowser() string {
	if p := os.Getenv("BROWSER_PATH"); p != "" {
		return p
	}
	for _, name := range []string{"brave-browser", "brave", "google-chrome", "chromium", "chromium-browser", "microsoft-edge"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}
