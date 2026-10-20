.PHONY: all help fmt fix tidy vet lint test test-race test\:redis test\:conformance vulncheck check build run

PWD := $(shell pwd)

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

NAME := korda
VERSION := $(shell git describe --tags 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --verify HEAD 2>/dev/null || echo "none")
DATE := $(shell date)

STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.7.0-0.dev.0.20261009230814-452d5bb86b45
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.9.0

TAGS ?=
ALL_TAGS := heic,tzf

TEST_REDIS ?= 127.0.0.1:6379
REFERENCE_URL ?= http://localhost:2283
REFERENCE_EMAIL ?= admin@example.com
REFERENCE_PASSWORD ?=

all: build

help: ## print this help
	@grep -E '^[a-zA-Z_:\\-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

fmt: ## fail if gofmt would change a file
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "$$files"; exit 1; fi

fix: ## fail if go fix would change a file
	go fix -diff ./...

tidy: ## fail if go mod tidy would change go.mod or go.sum
	go mod tidy -diff

vet: ## vet, without build tags and with all of them
	go vet ./...
	go vet -tags $(ALL_TAGS) ./...

lint: ## staticcheck, without build tags and with all of them
	go run $(STATICCHECK) ./...
	go run $(STATICCHECK) -tags $(ALL_TAGS) ./...

test: ## test
	go test -v ./...

test-race: ## test with the race detector, without build tags and with all of them
	go test -race ./...
	go test -race -tags $(ALL_TAGS) ./...

test\:redis: ## test with the race detector, including the jobs test against the Redis in TEST_REDIS
	KORDA_TEST_REDIS="$(TEST_REDIS)" go test -race -count=1 ./...

test\:conformance: ## diff korda built with TAGS against the Immich in REFERENCE_URL (REFERENCE_EMAIL, REFERENCE_PASSWORD)
	KORDA_REFERENCE_URL="$(REFERENCE_URL)" KORDA_REFERENCE_EMAIL="$(REFERENCE_EMAIL)" KORDA_REFERENCE_PASSWORD="$(REFERENCE_PASSWORD)" go test -tags "$(TAGS)" -count=1 -run '^TestConformance' -v .

vulncheck: ## govulncheck
	go run $(GOVULNCHECK) ./...

check: fmt fix tidy vet lint test-race vulncheck ## everything CI runs, use make -k to see all failures

build: ## build with the build tags in TAGS, such as TAGS=heic,tzf, see the notices in the documentation
	@echo "Building with the following parameters:"
	@echo "VERSION = $(VERSION)"
	@echo "COMMIT  = $(COMMIT)"
	@echo "DATE    = $(DATE)"
	@echo "TAGS    = $(TAGS)"
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags "$(TAGS)" -ldflags "-s -w -X \"main.Version=${VERSION}\" -X \"main.Commit=${COMMIT}\" -X \"main.Date=${DATE}\"" -o $(PWD)/build/$(NAME)

run: build ## build and run with korda.toml
	./build/korda -c "file://$(PWD)/korda.toml"
