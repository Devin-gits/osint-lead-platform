.PHONY: build test clean fmt vet

build:
	go build -o bin/email-validate ./cmd/email-validate

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
