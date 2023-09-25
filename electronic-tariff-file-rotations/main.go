package main

import (
	// "encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/joho/godotenv"
)

var defaultConfig = map[string]string{
	"ETF_BUCKET":              "my-cool-test-bucket-844815912454",
	"DELETION_CANDIDATE_DAYS": "42", // 6 weeks
	"DEBUG":                   "false",
}

type LambdaEvent struct {
	Date string `json:"date"`
}

type S3File struct {
	key string
	age string
}

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_VERSION") != "" {
		// we are in AWS
		lambda.Start(handler)
	} else {
		handler(nil)
	}
}

func initializeEnvironment() {
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()

		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	for key, defaultValue := range defaultConfig {
		if val, exists := os.LookupEnv(key); !exists || val == "" {
			err := os.Setenv(key, defaultValue)

			if err != nil {
				log.Fatal("Error occurred while setting environment variable")
			}
		}
	}
}

func getAWSSession() *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-2"),
	})

	if err != nil {
		log.Fatal("Error while creating AWS session.")
	}

	return sess
}

func isDeletionCandidate(S3File) bool {
	const layout = "2006-01-02"

	curtime := time.Now()
	fmtCurtime := curtime.Format(layout)

	date, _ := time.Parse(layout, S3File.age)
	dayDiff := int64((curtime.Sub(date)).Hours() / 24)

	if dayDiff >= os.Getenv("DELETION_CANDIDATE_DAYS") {
		return true
	} else {
		return false
	}
}

func handler(event *LambdaEvent) {
	initializeEnvironment()

	sess := getAWSSession()

	s3svc := s3.New(sess)

	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(os.Getenv("ETF_BUCKET"))})

	if err != nil {
		log.Println("Error getting bucket files.")
		log.Fatal(err)
	}

	for _, item := range resp.Contents {
		fmt.Println("Name: ", *item.Key)
		fmt.Println("Last modified", item.LastModified.Format("2006-01-02"))
	}

	return
}
