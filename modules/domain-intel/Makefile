.PHONY: build test test-short fmt vet clean

build:
	go build -o bin/domain-intel ./cmd/domain-intel

test:
	go test ./...

test-short:
	go test -short ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
