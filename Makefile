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

export GRAPH_DRIVER_TYPE?=neptune
export GRAPH_ADDR?=ws://localhost:8182/gremlin

build:
	@mkdir -p $(BUILD_ARCH)/$(BIN_DIR)
	go build $(LDFLAGS) -o $(BUILD_ARCH)/$(BIN_DIR)/dp-hierarchy-api cmd/dp-hierarchy-api/main.go
debug: build
	HUMAN_LOG=1 go run $(LDFLAGS) cmd/dp-hierarchy-api/main.go
test:
	go test -cover -race ./...

.PHONY: build debug test