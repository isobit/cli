.PHONY: all fmt vet test

all: fmt vet test

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...
