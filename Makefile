APP_NAME := monkeyrun
VERSION ?= dev

.PHONY: build build-all clean test fmt lint vet

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(APP_NAME) .

build-all:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/$(APP_NAME)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/$(APP_NAME)-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/$(APP_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/$(APP_NAME)-windows-amd64.exe .

test:
	CGO_ENABLED=0 go test ./... -v -count=1

fmt:
	gofmt -w .

lint: vet
	@echo "Lint passed (go vet)"

vet:
	go vet ./...

clean:
	rm -rf dist/ $(APP_NAME)
