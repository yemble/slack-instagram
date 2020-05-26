package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"strings"
)

type InstaMeta struct {
	Title    string
	URL      string
	ImageURL string
	HasVideo bool
}

func getMetaFromResponse(resp *http.Response, fetchedURL string) (*InstaMeta, error) {
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Unexpected status: %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading data: %w", err)
	}

	metaPattern := regexp.MustCompile(`<meta property="(.+?)" content="(.+?)" />`)

	matches := metaPattern.FindAllSubmatch(data, -1)
	if matches == nil {
		log.Println("Failed to match metadata in: %s", string(data)[0:300])
		return nil, fmt.Errorf("Error parsing instagram response, no metadata found")
	}

	meta := &InstaMeta{}

	for _, m := range matches {
		switch string(m[1]) {
		case "og:title":
			meta.Title = string(m[2])
		case "og:url":
			meta.URL = string(m[2])
		case "og:image":
			meta.ImageURL = string(m[2])
		}
	}

	if meta.ImageURL == "" {
		// fall back to scraping javascript

		meta.ImageURL, err = getDisplayURL(data)
		if err != nil {
			return meta, fmt.Errorf("Error parsing instagram response, no meta or display_url")
		}

		meta.URL = fetchedURL

		title, _ := getDocTitle(data)
		titleLines := strings.Split(title, "\n")
		meta.Title = titleLines[0]

	}

	meta.HasVideo = hasVideo(data)

	return meta, nil
}

func getDisplayURL(data []byte) (string, error) {
	p := regexp.MustCompile(`(?s)"display_url":"(.+?)"`)

	if match := p.FindSubmatch(data); match != nil {
		raw := string(match[1])
		return strings.Replace(raw, `\u0026`, `&`, -1), nil
	}

	return "", fmt.Errorf("not found")
}

func getDocTitle(data []byte) (string, error) {
	p := regexp.MustCompile(`(?s)<title>(.+?)</title>`)

	if match := p.FindSubmatch(data); match != nil {
		raw := string(match[1])
		return strings.TrimSpace(raw), nil
	}

	return "", fmt.Errorf("not found")
}

func hasVideo(data []byte) bool {
	p := regexp.MustCompile(`"is_video":true`)

	if match := p.FindSubmatch(data); match != nil {
		return true
	}

	return false
}

func (h *handler) fetchInsta(ctx context.Context, instaURL string) (*InstaMeta, error) {
	client := &http.Client{
		Timeout: externalTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, instaURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-agent", "private instagram slack expander <zach@y3m.net>")
	req.Header.Set("X-requested-with", runtime.Version())
	req.Header.Set("Cookie", h.config.CookieString)

	log.Printf("Fetching %s", instaURL)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Request for %s done, parsing meta data..", instaURL)

	meta, err := getMetaFromResponse(resp, instaURL)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
