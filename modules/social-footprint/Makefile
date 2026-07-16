.PHONY: build test test-short test-integration fmt vet pydeps clean

build:
	go build -o bin/social-footprint ./cmd/social-footprint

test:
	go test -count=1 ./...

test-short:
	go test -count=1 -short ./...

test-integration:
	go test -count=1 -tags integration ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

pydeps:
	python3 -m pip install -r requirements.txt

clean:
	rm -rf bin/
