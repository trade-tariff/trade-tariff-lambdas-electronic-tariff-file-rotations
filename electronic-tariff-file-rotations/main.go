package main

import (
	"fmt"
	"os"
	// "time"

	"github.com/aws/aws-lambda-go/lambda"
	// "github.com/joho/godotenv"
)

type LambdaEvent struct {
	Date string `json:"date"`
}

func handler(event *LambdaEvent) {
	fmt.Println("Hello, world!")
	return
}

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_VERSION") != "" {
		// we are in AWS
		lambda.Start(handler)
	} else {
		handler(nil)
	}
}
