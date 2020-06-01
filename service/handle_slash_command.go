package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

func getUsageString() string {
	return fmt.Sprintf("Usage: `%s <instagram-url> [photo-number](optional)`", slashCommand)
}

func (h *handler) handleSlashCommand(ctx context.Context, body url.Values) *slack.Msg {
	var (
		text        = strings.TrimSpace(body.Get("text"))
		responseURL = body.Get("response_url")
		userID      = body.Get("user_id")
		instaURL    string
		instaOffset int
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
	if len(s) < 1 || len(s) > 2 {
		return simpleEphemeralMessage(getUsageString())
	}

	instaURL = s[0]

	if len(s) > 1 {
		n, err := strconv.Atoi(s[1])
		if err != nil {
			return simpleEphemeralMessage(getUsageString())
		}

		if n > 0 {
			instaOffset = n - 1
		}
	}

	ssMsg := &SQSSlackMessage{
		RequestTimestamp: time.Now().Unix(),
		Type:             SQSMessageTypeSlash,
		SlashMessage: &SlashMessage{
			ResponseURL:   responseURL,
			UserID:        userID,
			InstagramURL:  instaURL,
			SelectedIndex: instaOffset,
		},
	}

	if err := h.enqueueMessage(ctx, ssMsg); err != nil {
		log.Printf("Error enqueueing slash message: %s", err)
		return simpleEphemeralMessage("Failed to enqueue request")
	}

	return simpleEphemeralMessage(fmt.Sprintf("Fetching %s ...", s[0]))
}

func (h *handler) processSQSSlashMessage(ctx context.Context, msg *SQSSlackMessage) {
	meta, err := h.fetchInsta(ctx, msg.SlashMessage.InstagramURL, msg.SlashMessage.SelectedIndex)

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

	extra := make([]string, 0, 0)

	if meta.ImageIsVideo {
		extra = append(extra, "video")
	}

	if meta.PartCount > 1 {
		extra = append(extra, fmt.Sprintf("%d parts", meta.PartCount))
	}

	if len(extra) > 0 {
		text = fmt.Sprintf("%s (%s)", text, strings.Join(extra, ", "))
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
