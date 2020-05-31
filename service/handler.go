package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	externalTimeout = 20 * time.Second
	maxLag          = 30 * time.Second

	slashCommand = "/insta"
)

func getMapValueInsensitive(m map[string]string, k string) string {
	lk := strings.ToLower(k)
	for mk, mv := range m {
		if strings.ToLower(mk) == lk {
			return mv
		}
	}
	return ""
}

type ChallengeRequest struct {
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Type      string `json:"type"`
}

type handler struct {
	config *Config
	sqs    *sqs.SQS
}

func NewHandler(config *Config, sqs *sqs.SQS) lambda.Handler {
	return &handler{
		config: config,
		sqs:    sqs,
	}
}

func (h *handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	// try http first
	apiReq := &events.APIGatewayProxyRequest{}
	if err := json.Unmarshal(payload, apiReq); err == nil && len(apiReq.Body) > 0 {
		resp, err := h.handleAPIRequest(ctx, apiReq)
		if err != nil {
			log.Printf("Error from api request handler: %w", err)
			return nil, err
		}

		data, err := json.Marshal(resp)
		return data, err
	}

	// try sqs next
	sqsEvent := &events.SQSEvent{}
	if err := json.Unmarshal(payload, sqsEvent); err == nil && len(sqsEvent.Records) > 0 {
		err := h.handleSQSEvent(ctx, sqsEvent)
		if err != nil {
			log.Printf("Error from sqs event handler: %w", err)
		}

		return nil, err
	}

	// unsupported
	return nil, fmt.Errorf("Unsupported lambda payload")
}

func (h *handler) handleAPIRequest(ctx context.Context, evt *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	var bodyString string

	if evt.IsBase64Encoded {
		dec, derr := base64.StdEncoding.DecodeString(evt.Body)
		if derr != nil {
			return NewSlackTextResponse(400, "Bad api request body (base64 decode fail)"), derr
		}

		bodyString = string(dec)
	} else {
		bodyString = evt.Body
	}

	ct := getMapValueInsensitive(evt.Headers, "content-type")

	log.Printf("API request type %s body %s", ct, bodyString)

	if ct == "application/x-www-form-urlencoded" {
		// slash command
		return h.handleAPIFormRequest(ctx, bodyString)
	} else if ct == "application/json" {
		// challenge or event
		resp, err := h.handleAPIJSONRequest(ctx, bodyString)
		if err == nil {
			return resp, nil
		}

		log.Printf("Error: %s", err)
	}

	log.Printf("Unhandled request")
	return NewAPIResponse(400, "text/plain", "Unsupported request"), nil
}

func (h *handler) handleAPIFormRequest(ctx context.Context, bodyString string) (*events.APIGatewayProxyResponse, error) {
	bodyValues, err := url.ParseQuery(bodyString)
	if err != nil {
		return NewSlackTextResponse(400, "Bad slack api request body"), err
	}

	tkn := bodyValues.Get("token")
	if info := h.config.TeamByRequestToken(tkn); info == nil {
		return NewSlackTextResponse(400, fmt.Sprintf("Bad slack api request token (%s)", tkn)), nil
	}

	cmd := bodyValues.Get("command")
	if cmd != slashCommand {
		return NewSlackTextResponse(400, fmt.Sprintf("Unexpected slack api /command (%s)", cmd)), nil
	}

	msg := h.handleSlashCommand(ctx, bodyValues)

	return NewSlackMessageResponse(200, msg), nil
}

func (h *handler) handleAPIJSONRequest(ctx context.Context, bodyString string) (*events.APIGatewayProxyResponse, error) {
	// challenge?
	challengeReq := &ChallengeRequest{}
	if err := json.Unmarshal([]byte(bodyString), challengeReq); err == nil && challengeReq.Type == "url_verification" {
		tkn := challengeReq.Token
		if info := h.config.TeamByRequestToken(tkn); info == nil {
			return NewAPIResponse(400, "text/plain", fmt.Sprintf("Bad slack api request token (%s)", tkn)), nil
		}

		return NewAPIResponse(200, "text/plain", challengeReq.Challenge), nil
	}

	// event?
	msg := &UnfurlEvent{}
	if err := json.Unmarshal([]byte(bodyString), msg); err == nil && msg.Type == "event_callback" {
		tkn := msg.Token
		if info := h.config.TeamByRequestToken(tkn); info == nil {
			return NewAPIResponse(400, "text/plain", fmt.Sprintf("Bad slack api request token (%s)", tkn)), nil
		}

		if err = h.handleEventCallback(ctx, msg); err != nil {
			return NewAPIResponse(500, "text/plain", "Error handling event"), nil
		}

		return NewAPIResponse(200, "text/plain", "Handling event"), nil
	}

	return nil, fmt.Errorf("Unsupported json request")
}

func (h *handler) handleSQSEvent(ctx context.Context, evt *events.SQSEvent) error {
	for _, r := range evt.Records {
		h.handleSQSMessage(ctx, r)
	}
	return nil
}
