package main

import (
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/yemble/slack-instagram/service"
)

func main() {
	log.Printf("Starting lambda handler")

	cfgString := os.Getenv("CONFIG_JSON")
	cfg, err := service.NewConfigFromJSON([]byte(cfgString))
	if err != nil {
		log.Fatalf("Loading config: %s", err)
	}

	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	h := service.NewHandler(cfg, sqs.New(awsSession))

	lambda.StartHandler(h)

	log.Printf("main exit")
}
