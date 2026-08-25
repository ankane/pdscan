.PHONY: install test lint format release snapshot docker

VERSION = 0.2.0

install:
	go install

test:
	go test ./... -v

lint:
	staticcheck -checks "inherit,-ST1005" ./...

format:
	go fmt ./...

release:
	goreleaser --clean

snapshot:
	goreleaser --clean --snapshot

docker:
	docker build --push --pull --no-cache --platform linux/amd64 --build-arg VERSION=$(VERSION) -t ankane/pdscan:latest -t ankane/pdscan:v$(VERSION) .
