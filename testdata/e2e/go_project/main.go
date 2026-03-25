package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	ctx := context.Background()

	s3Client := s3.New(s3.Options{})
	_, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		log.Fatal(err)
	}

	dynamoClient := dynamodb.New(dynamodb.Options{})
	_, err = dynamoClient.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		log.Fatal(err)
	}

	sqsClient := sqs.New(sqs.Options{})
	_, err = sqsClient.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		log.Fatal(err)
	}

	lambdaClient := lambda.New(lambda.Options{})
	_, err = lambdaClient.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		log.Fatal(err)
	}
}