.PHONY: build test test-short fmt vet pydeps clean

build:
	go build -o bin/extraction ./cmd/extraction

test:
	go test ./...

test-short:
	go test -short ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

pydeps:
	pip install -r requirements.txt

clean:
	rm -rf bin/
