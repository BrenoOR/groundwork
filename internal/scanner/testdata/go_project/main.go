package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	_ = s3.New(s3.Options{})
	_ = context.Background()
}