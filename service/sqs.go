package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	SQSMessageTypeSlash  = "slash_command"
	SQSMessageTypeUnfurl = "unfurl_event"
)

type SQSSlackMessage struct {
	RequestTimestamp int64         `json:"request_timestamp"`
	Type             string        `json:"type"`
	SlashMessage     *SlashMessage `json:"slash_message,omitempty"`
	UnfurlEvent      *UnfurlEvent  `json:"unfurl_message,omitempty"`
}

type SlashMessage struct {
	InstagramURL string `json:"instagram_url"`
	ResponseURL  string `json:"response_url"`
	UserID       string `jsoin:"user_id"`
}

func (h *handler) enqueueMessage(ctx context.Context, ssMsg *SQSSlackMessage) error {
	data, err := json.Marshal(ssMsg)
	if err != nil {
		return err
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(h.config.QueueURL),
		MessageBody: aws.String(string(data)),
	}

	log.Printf("Enqueueing message %#v", ssMsg)

	_, err = h.sqs.SendMessageWithContext(ctx, input)
	return err
}

func (h *handler) handleSQSMessage(ctx context.Context, sqsMsg events.SQSMessage) {
	log.Printf("Handling sqs message..")

	ssMsg := &SQSSlackMessage{}
	err := json.Unmarshal([]byte(sqsMsg.Body), ssMsg)
	if err != nil {
		log.Printf("Error unmarshaling sqs message body (%s) %s", sqsMsg.Body, err)
	} else {

		log.Printf("Got a message from SQS with type %s, created %ds ago", ssMsg.Type, time.Now().Unix()-ssMsg.RequestTimestamp)

		if time.Unix(ssMsg.RequestTimestamp, 0).Add(maxLag).Before(time.Now()) {
			log.Printf("SQS message too old %#v", ssMsg)
		} else {
			switch ssMsg.Type {
			case SQSMessageTypeSlash:
				h.processSQSSlashMessage(ctx, ssMsg)
			case SQSMessageTypeUnfurl:
				h.processSQSUnfurlMessage(ctx, ssMsg)
			}
		}
	}

	// delete
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(h.config.QueueURL),
		ReceiptHandle: aws.String(sqsMsg.ReceiptHandle),
	}
	if _, err = h.sqs.DeleteMessageWithContext(ctx, input); err != nil {
		log.Println("Error deleting sqs message %s: %s", sqsMsg.ReceiptHandle, err)
	}
}
