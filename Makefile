.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary
	go build -o azureschema ./cmd/azureschema

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: install
install: ## Install the binary
	go install ./cmd/azureschema

.PHONY: lint
lint: ## Run linter
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix ./...
