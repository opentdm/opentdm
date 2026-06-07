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
test: ## Run all Go tests with the race detector + coverage (-p 1: store+httpapi e2e share one DB)
	go -C $(SERVER) test -race -cover -p 1 ./...

.PHONY: test-e2e
test-e2e: ## Spin up a throwaway Postgres and run the full race+cover e2e suite (-p 1), then tear it down
	@bash -euo pipefail -c '\
	  name=otdm-e2e-pg; port=5433; \
	  trap "docker rm -f $$name >/dev/null 2>&1 || true" EXIT; \
	  docker rm -f $$name >/dev/null 2>&1 || true; \
	  echo "starting throwaway postgres ($$name) on :$$port..."; \
	  docker run -d --name $$name -e POSTGRES_USER=opentdm -e POSTGRES_PASSWORD=opentdm \
	    -e POSTGRES_DB=opentdm_test -p $$port:5432 postgres:16-alpine >/dev/null; \
	  for i in $$(seq 1 30); do docker exec $$name pg_isready -U opentdm -d opentdm_test >/dev/null 2>&1 && break; sleep 1; done; \
	  TEST_DATABASE_URL="postgres://opentdm:opentdm@localhost:$$port/opentdm_test?sslmode=disable" \
	    go test -race -cover -p 1 ./server/... ./cli/... ./apiclient/...; \
	'

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
