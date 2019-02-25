install:
	go install

test:
	cd internal && go test -v

lint:
	golint . cmd internal

format:
	go fmt
	cd cmd && go fmt
	cd internal && go fmt

release:
	goreleaser --rm-dist
