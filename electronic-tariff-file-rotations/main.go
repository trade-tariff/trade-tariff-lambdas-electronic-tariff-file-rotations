package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/joho/godotenv"
)

var defaultConfig = map[string]string{
	"ETF_BUCKET":              "trade-tariff-reporting",
	"S3_PREFIX":               "uk/reporting/",
	"S3_SEARCH_TERM":          "electronic_tariff_file",
	"DELETION_CANDIDATE_DAYS": "42", // 6 weeks
	"DEBUG":                   "false",
}

type LambdaEvent struct {
	Date string `json:"date"`
}

type S3File struct {
	key *s3.ObjectIdentifier
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

func isDeletionCandidate(file S3File) bool {
	const layout = "2006-01-02"
	deletionDays, _ := strconv.ParseInt(os.Getenv("DELETION_CANDIDATE_DAYS"), 10, 64)

	curtime := time.Now()
	date, _ := time.Parse(layout, file.age)
	dayDiff := int64((curtime.Sub(date)).Hours() / 24)

	if dayDiff >= deletionDays {
		return true
	} else {
		return false
	}
}

func handler(event *LambdaEvent) {
	initializeEnvironment()

	sess := getAWSSession()
	s3svc := s3.New(sess)

	var deletionList []S3File

	debug := os.Getenv("DEBUG") == "true"

	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(os.Getenv("ETF_BUCKET")),
		Prefix: aws.String(os.Getenv("S3_PREFIX")),
	})

	if err != nil {
		log.Println("Error getting bucket files.")
		log.Fatal(err)
	}

	for _, item := range resp.Contents {
		file := S3File{&s3.ObjectIdentifier{Key: item.Key}, item.LastModified.Format("2006-01-02")}

		if (strings.Contains(*item.Key, os.Getenv("S3_SEARCH_TERM"))) && isDeletionCandidate(file) {
			deletionList = append(deletionList, file)

			if debug {
				log.Printf("Deletion candidate found: %s\n", file.key)
			}
		}
	}

	if len(deletionList) != 0 {
		deleteKeys := make([]*s3.ObjectIdentifier, len(deletionList))

		i := 0
		for _, file := range deletionList {
			deleteKeys[i] = file.key
			i++
		}

		if debug {
			log.Println("Debug mode active, forgoing file deletion.")
		} else {
			_, err := s3svc.DeleteObjects(&s3.DeleteObjectsInput{
				Bucket: aws.String(os.Getenv("ETF_BUCKET")),
				Delete: &s3.Delete{
					Objects: deleteKeys,
					Quiet:   aws.Bool(false),
				},
			})

			if err != nil {
				log.Println("Error deleting files.")
				log.Fatal(err)
			}
		}

	} else {
		log.Printf("No candidates for deletion. Exiting!\n")
	}

	os.Exit(0)
}
