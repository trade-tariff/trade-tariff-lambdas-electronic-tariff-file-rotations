.PHONY: build clean

build:
	cd electronic-tariff-file-rotations && env GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/handler

clean:
	rm -rf ./bin
