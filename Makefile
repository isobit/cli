.PHONY: all fmt test lint vet

all: fmt lint test

fmt:
	go fmt ./...

lint:
	@test -z $(shell gofmt -l . | tee /dev/stderr) || { echo "files above are not go fmt"; exit 1; }
	golangci-lint run

vet:
	# This is also run by golangci-lint (make lint)
	go vet ./...

test:
	go test ./...
