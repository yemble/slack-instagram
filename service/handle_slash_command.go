package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

func getUsageString() string {
	return fmt.Sprintf("Usage: `%s [instagram-url]`", slashCommand)
}

func (h *handler) handleSlashCommand(ctx context.Context, body url.Values) *slack.Msg {
	var (
		text        = strings.TrimSpace(body.Get("text"))
		responseURL = body.Get("response_url")
		userID      = body.Get("user_id")
	)

	log.Printf("Got slash command '%s' from %s", text, userID)

	if !strings.HasPrefix(responseURL, "https://") {
		log.Printf("Bad response_url %s", responseURL)
		return simpleEphemeralMessage("bad request (response_url)")
	}

	if !strings.HasPrefix(text, "https://www.instagram.com/p/") {
		return simpleEphemeralMessage(getUsageString())
	}

	s := strings.Split(text, " ")
	if len(s) < 1 {
		return simpleEphemeralMessage(getUsageString())
	}

	ssMsg := &SQSSlackMessage{
		RequestTimestamp: time.Now().Unix(),
		Type:             SQSMessageTypeSlash,
		SlashMessage: &SlashMessage{
			ResponseURL:  responseURL,
			UserID:       userID,
			InstagramURL: s[0],
		},
	}

	if err := h.enqueueMessage(ctx, ssMsg); err != nil {
		log.Printf("Error enqueueing slash message: %s", err)
		return simpleEphemeralMessage("Failed to enqueue request")
	}

	return simpleEphemeralMessage(fmt.Sprintf("Fetching %s ...", s[0]))
}

func (h *handler) processSQSSlashMessage(ctx context.Context, msg *SQSSlackMessage) {
	meta, err := h.fetchInsta(ctx, msg.SlashMessage.InstagramURL)

	if err != nil {
		log.Printf("Error while fetching data from %s: %s", msg.SlashMessage.InstagramURL, err)
		h.postSlashResponse(ctx, msg.SlashMessage.ResponseURL, simpleEphemeralMessage("Error fetching data from instagram"))
		return
	}

	log.Printf("Fetched %s; meta: %#v", msg.SlashMessage.InstagramURL, meta)

	h.postSlashResponse(ctx, msg.SlashMessage.ResponseURL, h.slashResponse(msg.SlashMessage.UserID, meta))
}

func (h *handler) slashResponse(userID string, meta *InstaMeta) *slack.Msg {
	text := fmt.Sprintf("<@%s> shared this instagram post", userID)

	if meta.HasVideo {
		text = fmt.Sprintf("%s (contains video)", text)
	}

	return &slack.Msg{
		ResponseType: slack.ResponseTypeInChannel,
		Text:         text,
		Attachments: []slack.Attachment{
			slack.Attachment{
				Title:     meta.Title,
				TitleLink: meta.URL,
				ImageURL:  meta.ImageURL,
			},
		},
	}
}

func (h *handler) postSlashResponse(ctx context.Context, responseURL string, msg *slack.Msg) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("postSlashResponse marshal error: %s", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responseURL, bytes.NewReader(data))
	if err != nil {
		log.Printf("postSlashResponse request error: %s", err)
		return
	}
	req.Header.Set("content-type", "application/json")

	client := &http.Client{
		Timeout: externalTimeout,
	}

	log.Printf("Sending slash command response: %#v", msg)

	_, err = client.Do(req)
	if err != nil {
		log.Printf("postSlashResponse execute error: %s", err)
		return
	}
}
