package main

import (
	"log/slog"
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

var config = map[string]string{
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
	ok := initializeEnvironment()
	initializeLogger()

	if !ok {
		slog.Error("Error loading .env file.")
		os.Exit(1)
	}

	if os.Getenv("AWS_LAMBDA_FUNCTION_VERSION") != "" {
		// we are in AWS
		lambda.Start(handler)
	} else {
		handler(nil)
	}
}

func initializeEnvironment() (ok bool) {
	ok = true

	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()

		if err != nil {
			return false
		}
	}

	for key := range config {
		if val, exists := os.LookupEnv(key); exists || val != "" {
			config[key] = val
		}
	}

	return ok
}

func initializeLogger() {
	logLevel := slog.LevelInfo

	if config["DEBUG"] == "true" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	slog.SetDefault(logger)
}

func getAWSSession() *session.Session {
	sess, sessErr := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-2"),
	})

	if sessErr != nil {
		slog.Error("Error while creating AWS session.", "trace", sessErr)
		os.Exit(1)
	}

	return sess
}

func isDeletionCandidate(file S3File) bool {
	deletionDays, parseErr := strconv.ParseInt(config["DELETION_CANDIDATE_DAYS"], 10, 64)

	if parseErr != nil {
		slog.Error("Error parsing deletion candidate days.", "trace", parseErr)
		os.Exit(1)
	}

	date, timeParseErr := time.Parse("2006-01-02", file.age)

	if timeParseErr != nil {
		slog.Error("Error parsing datetime.", "trace", timeParseErr)
		os.Exit(1)
	}

	curtime := time.Now()
	dayDiff := int64((curtime.Sub(date)).Hours() / 24)

	if dayDiff >= deletionDays {
		return true
	} else {
		return false
	}
}

func handler(event *LambdaEvent) {
	sess := getAWSSession()
	s3svc := s3.New(sess)

	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config["ETF_BUCKET"]),
		Prefix: aws.String(config["S3_PREFIX"]),
	})

	if err != nil {
		slog.Error("Error getting bucket files.", "trace", err)
		os.Exit(1)
	}

	var deletionList []S3File

	for _, item := range resp.Contents {
		file := S3File{&s3.ObjectIdentifier{Key: item.Key}, item.LastModified.Format("2006-01-02")}

		if (strings.Contains(*item.Key, config["S3_SEARCH_TERM"])) && isDeletionCandidate(file) {
			deletionList = append(deletionList, file)
			slog.Debug("Deletion candidate found!", "file", file.key)
		}
	}

	if len(deletionList) != 0 {
		deleteKeys := make([]*s3.ObjectIdentifier, len(deletionList))

		i := 0
		for _, file := range deletionList {
			deleteKeys[i] = file.key
			i++
		}

		if config["DEBUG"] == "true" {
			slog.Debug("Debug mode active, forgoing file deletion.")
		} else {
			_, err := s3svc.DeleteObjects(&s3.DeleteObjectsInput{
				Bucket: aws.String(config["ETF_BUCKET"]),
				Delete: &s3.Delete{
					Objects: deleteKeys,
					Quiet:   aws.Bool(false),
				},
			})

			if err != nil {
				slog.Error("Error deleting files.", "trace", err)
				os.Exit(1)
			}
		}

	} else {
		slog.Info("No candidates for deletion. Exiting!")
	}
}
