CGO_ENABLED?=0
GOOS?=linux # dawin for macos
GOARCH?=amd64
BIN_OUT?=bin/app

VERSION?=$(shell git describe --tags --abbrev=0)

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	  go build -a -installsuffix cgo -ldflags="-X 'main.version=$(VERSION)'" -o $(BIN_OUT)

.PHONY: run
run:
	./$(BIN_OUT)

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	@[[ -z "$$(gofmt -l .)" ]] || echo '::error::need to run `make fmt` to format code';
