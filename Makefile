CGO_ENABLED?=0
GOOS?=linux
VERSION?=$(shell git describe --tags --abbrev=0)
BIN_OUT?=bin/app

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) \
	  go build -a -installsuffix cgo -ldflags="-X 'main.version=$(VERSION)'" -o $(BIN_OUT)

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...
