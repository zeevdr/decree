# Contributing to Central Config Service

Thank you for your interest in contributing! This guide covers how to set up your development environment, build, test, and submit changes.

## Prerequisites

- **Go** (1.24+)
- **Docker** and **Docker Compose**
- **Make**

That's it. All other tools (buf, sqlc, goose, protoc plugins) run inside Docker — no local installation needed.

## Getting Started

```bash
# Clone the repository
git clone https://github.com/zeevdr/central-config-service.git
cd central-config-service

# Generate code from protobuf and SQL specs
make generate

# Run tests
make test

# Build the binary
make build
```

## Development Cycle

The development workflow follows a specs-first approach:

```
modify specs → generate code → test → lint → build → deploy → e2e test
```

### Makefile Targets

| Target | Description | Runs in |
|--------|------------|---------|
| `make generate` | Generate Go code from protobuf (buf) and SQL (sqlc) | Docker |
| `make test` | Run unit tests | Local |
| `make lint` | Run golangci-lint, buf lint, buf breaking | Mixed |
| `make build` | Build the service binary to `bin/` | Local |
| `make image` | Build the Docker image | Docker |
| `make migrate` | Run database migrations | Docker |
| `make e2e` | Spin up full stack and run e2e tests | Docker |
| `make clean` | Remove build artifacts and generated code | Local |
| `make all` | generate → lint → test → build | Mixed |

### Specs-First Workflow

1. **Protobuf** — edit `.proto` files under `proto/centralconfig/v1/`, then run `make generate`
2. **SQL queries** — edit `.sql` files under `db/queries/`, then run `make generate`
3. **Migrations** — add new migration files under `db/migrations/`, then run `make migrate`

Generated code is **checked into git** — `go build` works immediately after cloning. Run `make generate` after modifying proto or SQL specs.

## Project Structure

```
cmd/server/          # Application entry point
proto/               # Protobuf definitions (API source of truth)
db/                  # SQL queries and migration files (DB source of truth)
gen/                 # Generated proto code (committed)
internal/
├── server/          # gRPC server setup and interceptors
├── schema/          # SchemaService implementation
├── config/          # ConfigService implementation
├── audit/           # AuditService implementation
├── auth/            # JWT validation
├── validation/      # Field validation logic
├── pubsub/          # Change propagation abstraction (Redis impl)
├── cache/           # Config cache abstraction (Redis impl)
└── storage/         # Database layer
build/               # Dockerfiles
deploy/helm/         # Helm chart
e2e/                 # End-to-end tests
```

## Running Locally

### Start dependencies

```bash
docker compose up -d postgres redis
```

### Run migrations

```bash
make migrate
```

### Run the service

```bash
make build
./bin/central-config-service \
  --db-write-url="postgres://centralconfig:localdev@localhost:5432/centralconfig?sslmode=disable" \
  --db-read-url="postgres://centralconfig:localdev@localhost:5432/centralconfig?sslmode=disable" \
  --redis-url="redis://localhost:6379" \
  --enable-services=schema,config,audit
```

### Run the full stack (service in Docker)

```bash
docker compose up -d
make migrate
```

## Testing

### Unit Tests

```bash
make test
```

Unit tests use testify for assertions and mock interfaces for dependencies (storage, cache, pubsub).

### End-to-End Tests

```bash
make e2e
```

This starts PostgreSQL, Redis, and the service via Docker Compose, runs migrations, executes e2e tests against the running service, and tears everything down.

## Code Style

- Follow standard Go conventions
- Run `make lint` before submitting — it runs golangci-lint with the project's configuration
- Proto files are linted with `buf lint` and checked for breaking changes with `buf breaking`

## Submitting Changes

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes following the specs-first workflow
4. Ensure `make all` passes (generate → lint → test → build)
5. Open a pull request against `main`

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.