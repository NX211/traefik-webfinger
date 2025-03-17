.PHONY: build clean test lint vendor release

VERSION ?= $(shell git describe --tags --always || echo "dev")
GO_LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION)"

export GO111MODULE=on

default: lint test build

build:
	go build $(GO_LDFLAGS) -o ./bin/webfinger-plugin .

install:
	go install $(GO_LDFLAGS)

test:
	go test -v -cover ./...

yaegi_test:
	yaegi test -v .

lint:
	golangci-lint run

vendor:
	go mod vendor
	go mod tidy

clean:
	rm -rf ./bin/ ./vendor/

release:
	goreleaser release --rm-dist

# Maintain compatibility with Mikefile
mike-lint: lint
mike-test: test
mike-vendor: vendor
mike-clean: clean 