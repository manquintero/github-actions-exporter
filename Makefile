CGO_ENABLED?=0
GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
ifneq ($(GOOS),darwin)
	GOOS=linux
endif
GOARCH=$(shell uname -m)
ifeq ($(GOARCH),x86_64)
	# map x86_64 arch. to use amd64
	GOARCH=amd64
endif
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

.PHONY: get
get:
	go get ./...

.PHONY: deps
deps: get
	go mod tidy -v

.PHONY: lint
lint:
	@[[ -z "$$(gofmt -l .)" ]] || echo '::error::need to run `make fmt` to format code';

.PHONY: clean
clean:
	rm -rf ./bin/
