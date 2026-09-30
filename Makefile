BINARY  := custom-docker-db
VERSION ?= dev
LDFLAGS := -s -w -X github.com/pvfm/custom-docker-db/internal/cli.Version=$(VERSION)

.PHONY: build test vet fmt lint clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

lint: vet
	@test -z "$$(gofmt -l .)" || (echo "gofmt necessário:"; gofmt -l .; exit 1)

clean:
	rm -rf bin
