# ==============================================================================
# Users Service Makefile
# ==============================================================================

GO             = go
GOLANGCI_LINT  = golangci-lint
COMPOSE_LOCAL  = docker compose -f docker/docker-compose.local.yml
COMPOSE_FULL   = docker compose -f docker/docker-compose.yml

REGISTRY     ?= ghcr.io
IMAGE_PREFIX ?= factory-chainline/users-service
TAG          ?= latest

CYAN   := \033[36m
GREEN  := \033[32m
YELLOW := \033[33m
RESET  := \033[0m

.DEFAULT_GOAL := help

# ==============================================================================
# Utility
# ==============================================================================

.PHONY: help
help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-30s$(RESET) %s\n", $$1, $$2}'

# ==============================================================================
# Setup & Maintenance
# ==============================================================================

.PHONY: setup
setup: ## Full project orchestration (install -> infra -> generate -> build)
	@echo "$(CYAN)=> Running full project setup...$(RESET)"
	@$(MAKE) install
	@$(MAKE) generate-keys
	@$(MAKE) infra-up
	@$(MAKE) generate-ent
	@$(MAKE) generate-mocks
	@$(MAKE) build
	@echo "$(GREEN)=> Setup complete! Run 'make dev' to start development.$(RESET)"

.PHONY: install
install: ## Download Go modules and dev tool dependencies
	@echo "=> Downloading Go modules..."
	@$(GO) mod download
	@echo "=> Installing dev tools (swag, mockery, ent, air, golangci-lint)..."
	@$(GO) install github.com/swaggo/swag/cmd/swag@latest
	@$(GO) install github.com/vektra/mockery/v2@latest
	@$(GO) install entgo.io/ent/cmd/ent@latest
	@$(GO) install github.com/air-verse/air@latest
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "=> Installation complete."

.PHONY: generate-keys
generate-keys: ## Generate a local Ed25519 JWT keypair (private_key.pem / public_key.pem)
	@echo "=> Generating Ed25519 JWT signing keys..."
	@openssl genpkey -algorithm ED25519 -out private_key.pem
	@openssl pkey -in private_key.pem -pubout -out public_key.pem
	@echo "=> Keys written to private_key.pem and public_key.pem."

.PHONY: clean
clean: ## Remove build artifacts and generated binaries
	@echo "=> Cleaning build artifacts..."
	@rm -rf tmp users-service coverage.out coverage.xml *.cover
	@echo "=> Clean complete."

.PHONY: nuke
nuke: clean ## Radical cleanup (also drops local infra volumes)
	@echo "=> Nuking local infrastructure volumes..."
	@$(COMPOSE_LOCAL) down -v
	@echo "=> Project nuked. Run 'make setup' to start over."

# ==============================================================================
# Development & Build
# ==============================================================================

.PHONY: dev
dev: infra-up ## Start the service locally with hot reload (requires air)
	@echo "$(GREEN)=> Starting development server with hot reload...$(RESET)"
	@air -c .air.toml

.PHONY: build
build: ## Build the service binary
	@echo "=> Building users-service binary..."
	@$(GO) build -o ./users-service ./app/cmd/main.go
	@echo "=> Build complete."

.PHONY: run
run: build ## Build and run the service binary directly
	@./users-service

# ==============================================================================
# Docker (Full Stack)
# ==============================================================================

.PHONY: up
up: ## Start full Docker stack (Postgres, Redis, Kafka, app with hot reload)
	@echo "=> Starting full Docker stack..."
	@$(COMPOSE_FULL) up -d --build
	@echo "=> Full stack is up and running."

.PHONY: down
down: ## Stop full Docker stack
	@echo "=> Stopping full Docker stack..."
	@$(COMPOSE_FULL) down

.PHONY: logs
logs: ## Follow full stack logs
	@$(COMPOSE_FULL) logs -f

# ==============================================================================
# Infrastructure (Postgres, Redis, Kafka only)
# ==============================================================================

.PHONY: infra-up
infra-up: ## Start only local infrastructure (Postgres, Redis, Kafka)
	@echo "=> Starting local infrastructure..."
	@$(COMPOSE_LOCAL) up -d
	@echo "=> Infrastructure is ready."

.PHONY: infra-down
infra-down: ## Stop local infrastructure
	@echo "=> Stopping local infrastructure..."
	@$(COMPOSE_LOCAL) down

.PHONY: infra-logs
infra-logs: ## Follow infrastructure logs
	@$(COMPOSE_LOCAL) logs -f

# ==============================================================================
# Code Generation
# ==============================================================================

.PHONY: generate-ent
generate-ent: ## Generate Ent schema models and Swagger docs
	@echo "=> Generating Ent schema models..."
	@$(GO) run entgo.io/ent/cmd/ent generate ./app/models/ent/schema
	@swag init -g ./app/cmd/main.go -o app/docs

.PHONY: generate-mocks
generate-mocks: ## Generate mocks with mockery and refresh Swagger docs
	@echo "=> Generating mocks with mockery..."
	@mockery
	@swag init -g app/cmd/main.go -o app/docs

.PHONY: generate-swagger
generate-swagger: ## Generate Swagger documentation
	@echo "=> Generating Swagger documentation..."
	@swag init -g ./app/cmd/main.go -o ./app/docs

# ==============================================================================
# Quality
# ==============================================================================

.PHONY: lint
lint: ## Run golangci-lint across the project
	@echo "=> Running linter..."
	@$(GOLANGCI_LINT) run ./...
	@echo "=> Lint complete."

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with autofix where supported
	@echo "=> Running linter with autofix..."
	@$(GOLANGCI_LINT) run --fix ./...
	@echo "=> Lint fix complete."

# ==============================================================================
# Testing Targets
# ==============================================================================

.PHONY: test
test: test-unit test-integration test-e2e ## Run unit, integration, and e2e tests

.PHONY: test-unit
test-unit: ## Run unit tests (app/ and tests/)
	@echo "=> Running Standard Unit Tests (in app/)..."
	@$(GO) test -v -tags=unit ./app/...
	@echo "=> Running Data-Driven Unit Tests (in tests/)..."
	@$(GO) test -v -tags=unit ./tests -args -test_type=unit

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	@echo "=> Running Integration Tests (requires Docker)..."
	@$(GO) test -v -tags=integration ./app/...
	@$(GO) test -v -tags=integration ./tests -args -test_type=integration

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests (requires Docker)
	@echo "=> Running End-to-End Tests (requires Docker)..."
	@$(GO) test -v -tags=e2e ./app/...
	@$(GO) test -v -tags=e2e ./tests -args -test_type=e2e

# ==============================================================================
# CI/CD & Registry (GitHub Actions)
# ==============================================================================

.PHONY: build-docker
build-docker: ## Build the production Docker image
	@echo "=> Building Docker image..."
	@docker build -t $(REGISTRY)/$(IMAGE_PREFIX):$(TAG) -f ./docker/Dockerfile .

.PHONY: docker-push
docker-push: ## Push the production Docker image to the registry
	@echo "=> Pushing image to $(REGISTRY)..."
	@docker push $(REGISTRY)/$(IMAGE_PREFIX):$(TAG)

.PHONY: docker-login
docker-login: ## Login to GitHub Container Registry
	@echo "=> Logging in to $(REGISTRY)..."
	@echo $${GITHUB_TOKEN} | docker login $(REGISTRY) -u $${GITHUB_ACTOR} --password-stdin
