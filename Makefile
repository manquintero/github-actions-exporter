CGO_ENABLED?=0
GOOS?=linux
VERSION?=$(shell git describe --tags --abbrev=0)

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) \
	  go build -a -installsuffix cgo -ldflags="-X 'main.version=$(VERSION)'" -o bin/app

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...
