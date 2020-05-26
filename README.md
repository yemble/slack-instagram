# slack-instagram
Lambda function to post instagrams to Slack

## Deployment

1. Compile with `GOOS=linux`, add binary to zip, create a lambda function using Go engine.
1. Attach an API gateway endpoint (POST method)
1. Attach an SQS queue
1. Configure a slack custom integration with a slash-command (eg `/insta`) pointing to the API gateway endpoint

Add an environment var for the function named `CONFIG_JSON` (see `service/config.go` for structure).
