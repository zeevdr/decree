BINARY_NAME := central-config-service
BUILD_DIR := bin
TOOLS_IMAGE := central-config-tools
TOOLS_SENTINEL := .tools-image-built
DOCKER_RUN_TOOLS := docker run --rm -u $(shell id -u):$(shell id -g) -e HOME=/tmp -v $(CURDIR):/workspace -w /workspace $(TOOLS_IMAGE)

.PHONY: all generate generate-proto generate-sqlc test lint build image migrate e2e clean tools help

all: generate lint test build

## tools: Build the tools Docker image (only when Dockerfile.tools changes)
tools: $(TOOLS_SENTINEL)
$(TOOLS_SENTINEL): build/Dockerfile.tools
	docker build -t $(TOOLS_IMAGE) -f build/Dockerfile.tools .
	@touch $(TOOLS_SENTINEL)

## generate: Generate code from protobuf and SQL specs
generate: $(TOOLS_SENTINEL)
	$(DOCKER_RUN_TOOLS) sh -c 'buf generate && cd db && sqlc generate'

## generate-proto: Generate Go code from protobuf definitions
generate-proto: $(TOOLS_SENTINEL)
	$(DOCKER_RUN_TOOLS) buf generate

## generate-sqlc: Generate Go code from SQL queries
generate-sqlc: $(TOOLS_SENTINEL)
	$(DOCKER_RUN_TOOLS) bash -c "cd db && sqlc generate"

## test: Run unit tests
test:
	go test ./... -race -count=1

## lint: Run linters
lint: lint-go lint-proto

## lint-go: Run Go linter
lint-go:
	golangci-lint run ./...

## lint-proto: Run protobuf linter and breaking change detection
lint-proto: $(TOOLS_SENTINEL)
	$(DOCKER_RUN_TOOLS) sh -c 'buf lint && buf breaking --against ".git#branch=main"'

## build: Build the service binary
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

## image: Build the Docker image
image:
	docker build -t $(BINARY_NAME) -f build/Dockerfile .

## migrate: Run database migrations
migrate: $(TOOLS_SENTINEL)
	$(DOCKER_RUN_TOOLS) goose -dir db/migrations postgres "$${DB_WRITE_URL:-postgres://centralconfig:localdev@localhost:5432/centralconfig?sslmode=disable}" up

## migrate-down: Roll back the last migration
migrate-down: $(TOOLS_SENTINEL)
	$(DOCKER_RUN_TOOLS) goose -dir db/migrations postgres "$${DB_WRITE_URL:-postgres://centralconfig:localdev@localhost:5432/centralconfig?sslmode=disable}" down

## e2e: Run end-to-end tests (docker compose → migrate → test → teardown)
e2e:
	docker compose up -d --wait service
	go test ./e2e/... -tags=e2e -v -race -count=1 || (docker compose down -v && exit 1)
	docker compose down -v

## clean: Remove build artifacts and generated code
clean:
	rm -rf $(BUILD_DIR)
	$(DOCKER_RUN_TOOLS) rm -rf api/centralconfig/ internal/storage/dbstore/

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
