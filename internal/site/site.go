// Package site — клиент сайта cherkasy.biz: вход по cookie-сессии, загрузка фото, отправка новости.
package site

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"gonews/internal/config"
	"gonews/internal/httpx"
	"gonews/internal/model"
	"gonews/internal/textutil"
)

type Client struct {
	cfg      config.SiteConfig
	http     *http.Client
	loggedIn bool
}

func New(cfg config.SiteConfig) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 60 * time.Second, Jar: jar},
	}
}

type uploadResult struct {
	Error   int    `json:"error"`
	Img     string `json:"img"`
	Preview string `json:"preview"`
	ID      int    `json:"id"`
}

func (c *Client) doPost(ctx context.Context, u string, form url.Values) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", httpx.UserAgent)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", c.cfg.AddURL)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(b), nil
}

// Login входит на сайт; cookie сохраняются в jar.
func (c *Client) Login(ctx context.Context) error {
	code, body, err := c.doPost(ctx, c.cfg.LoginURL, url.Values{
		"ajax": {"login"},
		"user": {c.cfg.Login},
		"pass": {c.cfg.Password},
	})
	if err != nil {
		return err
	}
	if code >= 400 {
		return fmt.Errorf("логин: HTTP %d: %s", code, textutil.Truncate(body, 200))
	}
	log.Printf("  ответ логина: %s", textutil.Truncate(strings.Join(strings.Fields(body), " "), 200))
	c.loggedIn = true
	return nil
}

// UploadPhoto скачивает картинку по ссылке и загружает её на сайт.
func (c *Client) UploadPhoto(ctx context.Context, imgURL string) (uploadResult, error) {
	// 1. скачиваем
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return uploadResult{}, err
	}
	req.Header.Set("User-Agent", httpx.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return uploadResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return uploadResult{}, fmt.Errorf("скачивание фото: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return uploadResult{}, err
	}

	// 2. собираем multipart-форму
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	name := path.Base(strings.Split(imgURL, "?")[0])
	if path.Ext(name) == "" {
		name += ".jpg"
	}
	_ = mw.WriteField("name", name)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		return uploadResult{}, err
	}
	if _, err := fw.Write(data); err != nil {
		return uploadResult{}, err
	}
	mw.Close()

	// 3. отправляем
	up, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.UploadURL, &buf)
	if err != nil {
		return uploadResult{}, err
	}
	up.Header.Set("Content-Type", mw.FormDataContentType())
	up.Header.Set("User-Agent", httpx.UserAgent)
	up.Header.Set("X-Requested-With", "XMLHttpRequest")
	up.Header.Set("Referer", c.cfg.AddURL)
	r2, err := c.http.Do(up)
	if err != nil {
		return uploadResult{}, err
	}
	defer r2.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(r2.Body, 1<<20))
	if r2.StatusCode >= 400 {
		return uploadResult{}, fmt.Errorf("загрузка фото: HTTP %d: %s", r2.StatusCode, textutil.Truncate(string(b), 200))
	}
	var res uploadResult
	if err := json.Unmarshal(b, &res); err != nil {
		return uploadResult{}, fmt.Errorf("ответ загрузки: %v: %s", err, textutil.Truncate(string(b), 200))
	}
	if res.Error != 0 || res.ID == 0 {
		return uploadResult{}, fmt.Errorf("сайт отклонил фото: %s", textutil.Truncate(string(b), 200))
	}
	return res, nil
}

// Publish отправляет одну новость в форму «Відправити редактору».
func (c *Client) Publish(ctx context.Context, it model.NewsItem) error {
	if !c.loggedIn {
		if err := c.Login(ctx); err != nil {
			return err
		}
	}
	editor := ""
	if it.Link != "" {
		editor = "Джерело: " + it.Link
	}
	form := url.Values{
		"ajax":     {"add"},
		"resource": {c.cfg.ResourceID},
		"type":     {"news"},
		"name":     {it.Title},
		"txt":      {it.Description},
		"from":     {""},
		"till":     {""},
		"editor":   {editor},
	}

	if it.Image != "" {
		res, err := c.UploadPhoto(ctx, it.Image)
		if err != nil {
			log.Printf("  фото не загружено, публикую без него: %v", err)
		} else {
			form.Add("photos[]", strconv.Itoa(res.ID))
		}
	}

	code, body, err := c.doPost(ctx, c.cfg.AddURL, form)
	if err != nil {
		return err
	}
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		c.loggedIn = false
	}
	if code >= 400 {
		return fmt.Errorf("HTTP %d: %s", code, textutil.Truncate(body, 200))
	}
	log.Printf("  ответ сайта: %s", textutil.Truncate(strings.Join(strings.Fields(body), " "), 200))
	return nil
}
