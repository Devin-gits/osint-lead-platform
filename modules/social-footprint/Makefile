.PHONY: build test test-short fmt vet pydeps clean

build:
	go build -o bin/social-footprint ./cmd/social-footprint

test:
	go test ./...

test-short:
	go test -short ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

pydeps:
	python3 -m pip install -r requirements.txt

clean:
	rm -rf bin/
