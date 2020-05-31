# slack-instagram
Lambda function to post instagrams to Slack

## Usage

`/insta <url> [image number]`

Will post a large image and a link to the post.

* Optional image selection argument.
* Indicates number of photos and whether post is video or not.

## Test

`go test ./...`

## Deployment

1. Compile with `GOOS=linux`, add binary to zip, create a [Lambda](https://aws.amazon.com/lambda/) function using `Go` engine.
1. Attach an [API gateway](https://aws.amazon.com/api-gateway/) endpoint (POST method)
1. Attach an [SQS queue](https://aws.amazon.com/sqs/) 
1. Configure a slack custom integration with a slash-command (eg `/insta`) pointing to the API gateway endpoint

Add an environment var for the function named `CONFIG_JSON` (see `service/config.go` for structure).
