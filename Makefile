.PHONY: build clean lint deploy-production

build:
	cd electronic-tariff-file-rotations && env GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bootstrap

clean:
	rm -rf ./bin

lint:
	cd electronic-tariff-file-rotations && golangci-lint run

deploy-development: clean build
	STAGE=development \
		DELETION_CANDIDATE_DAYS=14 \
		serverless deploy --verbose

deploy-staging: clean build
	STAGE=staging
		DELETION_CANDIDATE_DAYS=91 \
		serverless deploy --verbose

deploy-production: clean build
	STAGE=production
		DELETION_CANDIDATE_DAYS=91 \
		serverless deploy --verbose
