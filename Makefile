.PHONY: release
SHELL := /bin/bash
VERSION := $(shell head -n 1 VERSION)

release:
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o="artifacts/goci-$(VERSION)-linux-amd64" ./cmd/goci
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o="artifacts/goci-$(VERSION)-darwin-amd64" ./cmd/goci
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o="artifacts/goci-$(VERSION)-darwin-arm64" ./cmd/goci

install_deps:
	go mod vendor
