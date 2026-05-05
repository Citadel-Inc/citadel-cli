# citadel-cli — top-level developer ergonomics.

.PHONY: help build build-all test vet lint golangci fmt verify clean coverage-check

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*## "} /^[-a-zA-Z0-9_]+:.*## / { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the citadel-cli binary into ./citadel-cli
	go build -o ./citadel-cli .

build-all: ## Cross-compile for linux-amd64, linux-arm64, darwin-arm64 into dist/
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/citadel-cli-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/citadel-cli-linux-arm64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/citadel-cli-darwin-arm64 .

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
