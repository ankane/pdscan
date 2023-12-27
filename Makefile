install:
	go install

test:
	go test ./... -v

lint:
	staticcheck -checks "inherit,-ST1005" ./...

format:
	go fmt
	cd cmd && go fmt
	cd internal && go fmt

release:
	goreleaser --clean --skip-publish

snapshot:
	goreleaser --clean --snapshot
