install:
	go install

test:
	go test ./... -v

lint:
	staticcheck ./...

format:
	go fmt
	cd cmd && go fmt
	cd internal && go fmt

release:
	goreleaser --clean --skip-publish

snapshot:
	goreleaser --clean --snapshot
