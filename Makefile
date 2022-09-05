install:
	go install

test:
	cd internal && go test -v

lint:
	staticcheck ./...

format:
	go fmt
	cd cmd && go fmt
	cd internal && go fmt

release:
	goreleaser --rm-dist --skip-publish

snapshot:
	goreleaser --rm-dist --snapshot
