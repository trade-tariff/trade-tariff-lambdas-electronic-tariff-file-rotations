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

type Config struct {
	ETF_BUCKET              string
	S3_PREFIXES             []string
	DELETION_CANDIDATE_DAYS int64
	DEBUG                   bool
}

var config = Config{
	ETF_BUCKET:              "trade-tariff-reporting",
	S3_PREFIXES:             []string{"uk/reporting/", "xi/reporting/"},
	DELETION_CANDIDATE_DAYS: 42, // 6 weeks
	DEBUG:                   false,
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

	config.ETF_BUCKET = os.Getenv("ETF_BUCKET")

	if prefix, exists := os.LookupEnv("S3_PREFIX"); exists {
		config.S3_PREFIXES = strings.Split(prefix, ",")
	}

	if days, err := strconv.ParseInt(os.Getenv("DELETION_CANDIDATE_DAYS"), 10, 64); err == nil {
		config.DELETION_CANDIDATE_DAYS = days
	}
	config.DEBUG, _ = strconv.ParseBool(os.Getenv("DEBUG"))

	return ok
}

func initializeLogger() {
	logLevel := slog.LevelInfo

	if config.DEBUG {
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
	date, timeParseErr := time.Parse("2006-01-02", file.age)

	if timeParseErr != nil {
		slog.Error("Error parsing datetime.", "trace", timeParseErr)
		os.Exit(1)
	}

	curtime := time.Now()
	dayDiff := int64((curtime.Sub(date)).Hours() / 24)

	if dayDiff >= config.DELETION_CANDIDATE_DAYS {
		return true
	} else {
		return false
	}
}

func handler(event *LambdaEvent) {
	sess := getAWSSession()
	s3svc := s3.New(sess)

	for _, prefix := range config.S3_PREFIXES {
		handle_prefix(prefix, s3svc)
	}
}

func handle_prefix(prefix string, s3svc *s3.S3) {
	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.ETF_BUCKET),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		slog.Error("Error getting bucket files.", "trace", err)
		os.Exit(1)
	}

	var deletionList []S3File

	for _, item := range resp.Contents {
		file := S3File{&s3.ObjectIdentifier{Key: item.Key}, item.LastModified.Format("2006-01-02")}

		if isDeletionCandidate(file) {
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

		if config.DEBUG {
			slog.Debug("Debug mode active, forgoing file deletion.")
		} else {
			deleted, err := s3svc.DeleteObjects(&s3.DeleteObjectsInput{
				Bucket: aws.String(config.ETF_BUCKET),
				Delete: &s3.Delete{
					Objects: deleteKeys,
				},
			})

			if err != nil {
				slog.Error("Error deleting files.", "trace", err)
				os.Exit(1)
			}

			slog.Info("Deleted items from S3.", "items", len(deleted.Deleted))

			if len(deleted.Errors) > 0 {
				slog.Warn("Encountered errors deleting items.", "errors", deleted.Errors)
			}
		}

	} else {
		slog.Info("No candidates for deletion. Moving on!")
	}
}
