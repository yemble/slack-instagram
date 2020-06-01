package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"strings"
)

type InstaMeta struct {
	Username     string
	UserPicURL   string
	Title        string
	URL          string
	ImageURL     string
	ImageIsVideo bool
	PartCount    int
}

func getOGMeta(data []byte) (*InstaMeta, error) {
	metaPattern := regexp.MustCompile(`<meta property="(.+?)" content="(.+?)" />`)

	matches := metaPattern.FindAllSubmatch(data, -1)
	if matches == nil {
		log.Printf("Failed to match metadata in: %s", string(data)[0:300])
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
		return nil, fmt.Errorf("No image url in og meta")
	}

	return meta, nil
}

func getMetaFromResponse(resp *http.Response, fetchedURL string, instaOffset int) (*InstaMeta, error) {
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Unexpected status: %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading data: %w", err)
	}

	/*m, err := getOGMeta(data)
	if err == nil {
		return m, nil
	}*/

	meta := &InstaMeta{}

	title, _ := getDocTitle(data)
	titleLines := strings.Split(title, "\n")
	meta.Title = titleLines[0]

	meta.URL = fetchedURL

	// get json from body, parse and extract data

	ad, err := parseAdditionalData(data)
	if err != nil {
		return meta, fmt.Errorf("Error parsing instagram response, no metadata or additional data")
	}

	scm := ad.GraphQL.ShortcodeMedia

	meta.Username = scm.Owner.Username
	meta.UserPicURL = scm.Owner.ProfilePicURL

	if scm.EdgeSideCarToChildren == nil {
		meta.PartCount = 1
	} else {
		meta.PartCount = len(scm.EdgeSideCarToChildren.Edges)
	}

	if scm.EdgeSideCarToChildren == nil || instaOffset >= len(scm.EdgeSideCarToChildren.Edges) || instaOffset < 0 {
		meta.ImageURL = scm.DisplayURL
		meta.ImageIsVideo = scm.IsVideo
	} else {
		node := scm.EdgeSideCarToChildren.Edges[instaOffset].Node
		meta.ImageURL = node.DisplayURL
		meta.ImageIsVideo = node.IsVideo
	}

	return meta, nil
}

func parseAdditionalData(data []byte) (*InstagramAdditionalData, error) {
	p := regexp.MustCompile(`(?s)({"graphql":{"shortcode_media":.+?)\);?</script>`)

	if match := p.FindSubmatch(data); match != nil {
		ad := &InstagramAdditionalData{}
		if err := json.Unmarshal(match[1], ad); err != nil {
			return nil, err
		}

		return ad, nil
	}

	return nil, fmt.Errorf("json not found")
}

func getDocTitle(data []byte) (string, error) {
	p := regexp.MustCompile(`(?s)<title>(.+?)</title>`)

	if match := p.FindSubmatch(data); match != nil {
		raw := string(match[1])
		return strings.TrimSpace(raw), nil
	}

	return "", fmt.Errorf("not found")
}

func (h *handler) fetchInsta(ctx context.Context, instaURL string, instaOffset int) (*InstaMeta, error) {
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

	meta, err := getMetaFromResponse(resp, instaURL, instaOffset)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
