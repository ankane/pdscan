.PHONY: install test lint format release snapshot

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
