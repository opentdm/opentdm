# opentdm developer entrypoints.
.DEFAULT_GOAL := help
SERVER := ./server

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the server binary into bin/
	go -C $(SERVER) build -o ../bin/opentdm-server ./cmd/opentdm-server

.PHONY: test
test: ## Run all Go tests with the race detector + coverage
	go -C $(SERVER) test -race -cover ./...

.PHONY: vet
vet: ## go vet the server
	go -C $(SERVER) vet ./...

.PHONY: fmt
fmt: ## gofmt the server
	gofmt -w $(SERVER)

.PHONY: tidy
tidy: ## go mod tidy the server
	go -C $(SERVER) mod tidy

.PHONY: gen-key
gen-key: ## Print a fresh base64 32-byte key
	@go -C $(SERVER) run ./cmd/opentdm-server gen-key

.PHONY: up
up: ## Start the docker-compose stack (app + postgres)
	docker compose up -d --build

.PHONY: down
down: ## Stop the docker-compose stack
	docker compose down

.PHONY: logs
logs: ## Tail app logs
	docker compose logs -f app
