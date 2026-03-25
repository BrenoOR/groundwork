package main

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	_ = s3.New(s3.Options{})
	_ = dynamodb.New(dynamodb.Options{})
	_ = sqs.New(sqs.Options{})
}