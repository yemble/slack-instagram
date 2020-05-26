package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/slack-go/slack"
)

type UnfurlEvent struct {
	Token     string            `json:"token"`
	Event     UnfurlEventDetail `json:"event"`
	Type      string            `json:"type"`
	EventID   string            `json:"event_id"`
	EventTime int               `json:"event_time"`
}

type UnfurlEventDetail struct {
	Type             string            `json:"type"`
	Channel          string            `json:"channel"`
	User             string            `json:"user"`
	MessageTimestamp string            `json:"message_ts"`
	ThreadTimestamp  string            `json:"thread_ts,omitempty"`
	Links            []UnfurlEventLink `json:"links"`
}

type UnfurlEventLink struct {
	Domain string `json:"domain"`
	URL    string `json:"url"`
}

type UnfurlBody struct {
	Token     string            `json:"token"`
	Channel   string            `json:"channel"`
	Timestamp string            `json:"ts"`
	Unfurls   map[string]Unfurl `json:"unfurls"`
}
type Unfurl struct {
	Blocks []slack.SectionBlock `json:"blocks"`
}

func (h *handler) handleEventCallback(ctx context.Context, msg *UnfurlEvent) error {
	if msg.Event.Type != "link_shared" {
		log.Printf("Event isn't link_shared")
		return fmt.Errorf("unsupported event type %s", msg.Event.Type)
	}

	ssMsg := &SQSSlackMessage{
		RequestTimestamp: time.Now().Unix(),
		Type:             SQSMessageTypeUnfurl,
		UnfurlEvent:      msg,
	}

	if err := h.enqueueMessage(ctx, ssMsg); err != nil {
		log.Printf("Error enqueueing slash message: %s", err)
		return fmt.Errorf("Failed to enqueue request: %s", err)
	}

	return nil
}

func (h *handler) processSQSUnfurlMessage(ctx context.Context, msg *SQSSlackMessage) {
	unfurls := make(map[string]Unfurl)

	evt := msg.UnfurlEvent.Event

	for _, link := range evt.Links {
		meta, err := h.fetchInsta(ctx, link.URL)

		if err != nil {
			log.Printf("Error while fetching data from %s: %s", link.URL, err)
			return
		}

		log.Printf("Fetched %s; meta: %#v", link.URL, meta)

		unfurls[link.URL] = Unfurl{
			Blocks: []slack.SectionBlock{
				slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: fmt.Sprintf("%s - [link](%s)", meta.Title, meta.URL),
					},
					Accessory: &slack.Accessory{
						ImageElement: &slack.ImageBlockElement{
							Type:     slack.METImage,
							ImageURL: meta.ImageURL,
						},
					},
				},
			},
		}
	}

	if team := h.config.TeamByRequestToken(msg.UnfurlEvent.Token); team != nil {
		unfurlBody := UnfurlBody{
			Token:     team.OauthToken,
			Channel:   evt.Channel,
			Timestamp: evt.MessageTimestamp,
			Unfurls:   unfurls,
		}

		postUnfurlResponse(ctx, team.OauthToken, unfurlBody)
	}
}

func postUnfurlResponse(ctx context.Context, otkn string, msg UnfurlBody) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("postUnfurlResponse marshal error: %s", err)
		return
	}

	const url = "https://slack.com/api/chat.unfurl"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		log.Printf("postUnfurlResponse request error: %s", err)
		return
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", otkn))

	client := &http.Client{
		Timeout: externalTimeout,
	}

	log.Printf("Sending unfurls request: %#v", msg)

	_, err = client.Do(req)
	if err != nil {
		log.Printf("postUnfurlResponse execute error: %s", err)
		return
	}
}
