SHELL=bash

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BIN_DIR?=.

BUILD_TIME=$(shell date +%s)
GIT_COMMIT=$(shell git rev-parse HEAD)
VERSION ?= $(shell git tag --points-at HEAD | grep ^v | head -n 1)
LDFLAGS=-ldflags "-w -s -X 'main.Version=${VERSION}' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

export GOOS?=$(shell go env GOOS)
export GOARCH?=$(shell go env GOARCH)

export GRAPH_DRIVER_TYPE?=neo4j
export GRAPH_ADDR?=bolt://localhost:7687

PHONY: all
all: audit test build

PHONY: audit
audit:
	go list -m all | nancy sleuth

PHONY: build
build:
	@mkdir -p $(BUILD_ARCH)/$(BIN_DIR)
	go build $(LDFLAGS) -o $(BUILD_ARCH)/$(BIN_DIR)/dp-hierarchy-api cmd/dp-hierarchy-api/main.go

PHONY: debug
debug: build
	HUMAN_LOG=1 go run $(LDFLAGS) -race cmd/dp-hierarchy-api/main.go

PHONY: test
test:
	go test -cover -race ./...

PHONY: lint
lint:
	golangci-lint run ./...

test-component:
	exit

.PHONY: build debug test component
