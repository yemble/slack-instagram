package service

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/slack-go/slack"
)

func NewAPIResponse(code int, ctype string, data string) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers: map[string]string{
			"content-type": ctype,
		},
		Body: data,
	}
}

func NewSlackMessageResponse(code int, body *slack.Msg) *events.APIGatewayProxyResponse {
	data, err := json.Marshal(body)
	if err != nil {
		log.Printf("NewSlackMessageResponse marshal error: %s", err)
		return &events.APIGatewayProxyResponse{StatusCode: 500}
	}

	return NewAPIResponse(code, "application/json", string(data))
}

func NewSlackTextResponse(code int, text string) *events.APIGatewayProxyResponse {
	return NewSlackMessageResponse(code, &slack.Msg{
		ResponseType: slack.ResponseTypeEphemeral,
		Text:         text,
	})
}

func simpleEphemeralMessage(text string) *slack.Msg {
	return &slack.Msg{
		ResponseType: slack.ResponseTypeEphemeral,
		Text:         text,
	}
}
