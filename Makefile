# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

unit: ### Run unit tests
	go test ./...
.PHONY: unit

test: ### Run yandex tests
	go build -o shortener ./cmd/shortener/*.go &&\
	./shortenertest -test.v -test.run=^TestIteration4$$ -binary-path=./shortener -source-path=. -test.v
.PHONY: test

lint: ### Run linters
	golangci-lint run
.PHONY: lint

checks: unit test lint ### Run all checks
	@echo ""
	@echo "All good!"
.PHONY: checks
