APP     := krypt
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
DIST    := dist

.PHONY: all clean build \
        darwin-arm64 darwin-amd64 \
        linux-amd64 linux-arm64 \
        windows-amd64

all: build

## build: compile for all platforms
build: darwin-arm64 darwin-amd64 linux-amd64 linux-arm64 windows-amd64

darwin-arm64:
	@mkdir -p $(DIST)
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(APP)-darwin-arm64      .

darwin-amd64:
	@mkdir -p $(DIST)
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(APP)-darwin-amd64      .

linux-amd64:
	@mkdir -p $(DIST)
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(APP)-linux-amd64       .

linux-arm64:
	@mkdir -p $(DIST)
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(APP)-linux-arm64       .

windows-amd64:
	@mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(APP)-windows-amd64.exe .

## install: install krypt to ~/go/bin (make sure ~/go/bin is in your PATH)
install:
	go install $(LDFLAGS) .

## uninstall: remove krypt from ~/go/bin
uninstall:
	rm -f $(shell go env GOPATH)/bin/$(APP)

## clean: remove dist/
clean:
	rm -rf $(DIST)

## help: show this message
help:
	@echo "Usage: make [target]"
	@grep -E '^## ' Makefile | sed 's/## /  /'
