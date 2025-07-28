# Makefile for scifind-backend - Simplified for daily development

# Go configuration
GO := go
BINARY := scifind-backend
SRC := ./cmd/server

# Docker configuration
DOCKER_IMAGE := scifind-backend
DOCKER_TAG := latest

.PHONY: help build test run dev docker-up docker-down migrate clean

##@ Development Commands

help: ## Show this help message
	@echo "SciFind Backend - Development Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Core Commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application binary
	$(GO) build -o $(BINARY) $(SRC)

run: build ## Build and run the application
	./$(BINARY)

dev: ## Run in development mode with hot reload
	$(GO) run $(SRC) -config=./configs/config.dev.yaml

test: ## Run all tests
	$(GO) test ./...

test-watch: ## Run tests in watch mode
	$(GO) test ./... -v -count=1

##@ Docker Commands

docker-up: ## Start development environment with Docker
	docker-compose up -d

docker-down: ## Stop development environment
	docker-compose down

docker-logs: ## View logs from all services
	docker-compose logs -f

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

##@ Database Commands

migrate-up: ## Run database migrations
	$(GO) run ./cmd/migrate up

migrate-down: ## Rollback database migrations
	$(GO) run ./cmd/migrate down

migrate-create: ## Create new migration (usage: make migrate-create NAME=add_users)
	$(GO) run ./cmd/migrate create $(NAME)

##@ Code Quality

fmt: ## Format Go code
	$(GO) fmt ./...

lint: ## Run linter
	golangci-lint run

check: fmt lint ## Format code and run linter

##@ Cleanup

clean: ## Clean build artifacts
	$(GO) clean
	rm -f $(BINARY)

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

# Default target
.DEFAULT_GOAL := help