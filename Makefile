.PHONY: all fmt test lint vet

all: fmt lint test

fmt:
	go fmt ./...

lint:
	golangci-lint run

vet:
	# This is also run by golangci-lint (make lint)
	go vet ./...

test:
	go test ./...
