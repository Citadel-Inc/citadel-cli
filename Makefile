# citadel-cli — top-level developer ergonomics.

.PHONY: help build build-all test vet lint golangci fmt verify clean coverage-check

VERSION ?= dev
LDFLAGS = -X github.com/Rethunk-Tech/citadel-cli/cmd.Version=$(VERSION)

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*## "} /^[-a-zA-Z0-9_]+:.*## / { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the citadel-cli binary into ./citadel-cli
	go build -ldflags "$(LDFLAGS)" -o ./citadel-cli .

build-all: ## Cross-compile for linux-amd64, linux-arm64, darwin-arm64, windows-amd64 into dist/
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS) -s -w" -o dist/citadel-cli-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS) -s -w" -o dist/citadel-cli-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS) -s -w" -o dist/citadel-cli-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS) -s -w" -o dist/citadel-cli-windows-amd64.exe .

test: ## Run go test with race detector across all packages
	go test -race ./...

vet: ## Run go vet
	go vet ./...

lint: golangci ## Alias for golangci

golangci: ## Run golangci-lint (stricter pass than vet)
	golangci-lint run

fmt: ## Run go fmt across all packages
	go fmt ./...

verify: vet test golangci ## Pre-push gate: vet + race tests + golangci

coverage-check: ## Enforce COVERAGE_MIN per package (default 75%); see scripts/check-package-coverage.sh
	bash scripts/check-package-coverage.sh

clean: ## Remove build artefacts
	rm -f ./citadel-cli
	rm -rf ./dist
