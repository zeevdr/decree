# Central Config Service

Schema-driven business configuration management for multi-tenant services.

## What is this?

Central Config Service manages **business-oriented configuration** — approval rules, fee structures, settlement windows, feature parameters — the kind of config that lives between your infrastructure settings and your application code.

Unlike tools like etcd, Consul, or Spring Cloud Config (which focus on system/infrastructure configuration), Central Config Service provides:

- **Typed schemas** — define your config structure with types, validation, and constraints
- **Multi-tenancy** — apply schemas to tenants with role-based access and field-level locking
- **Versioned configs** — every change creates a version; rollback to any previous state
- **Real-time subscriptions** — gRPC streaming pushes changes to consumers instantly
- **Audit trail** — full history of who changed what, when, and why
- **Import/Export** — portable schemas and configs in YAML format
- **Optimistic concurrency** — safe read-modify-write with checksum validation

## Features

### Schema Management

- Define config schemas with typed fields: `int`, `string`, `time`, `duration`, `url`, `json`
- Hierarchical field namespacing (e.g., `payments.settlement.window`)
- Field validation: min/max, regex, enum, JSON Schema for complex types
- Nullable/required fields
- Schema versioning with immutable published versions and parent version lineage
- Schema and version descriptions for documentation
- Field deprecation with read redirects for migrations
- Schema import/export (YAML)

### Config Management

- Read and write config values with schema validation
- Batch updates — modify multiple fields in a single version
- Version descriptions — annotate changes with context
- Value descriptions — document the meaning of individual values
- Rollback to any previous version
- Request a specific version for consistent reads within a flow
- Config import/export (YAML)

### Access Control

| Role | Capabilities |
|------|-------------|
| **SuperAdmin** | Manage schemas, tenants, lock fields, full config access |
| **Admin** | Read/write config values (unlocked fields) for assigned tenant |
| **User** | Read-only config access for assigned tenant |

### Observability

- OpenTelemetry tracing and metrics
- Usage statistics — track which fields are read and how often
- gRPC health checks for Kubernetes probes

## Quick Start

### Docker Compose (local development)

```bash
# Clone the repository
git clone https://github.com/zeevdr/central-config-service.git
cd central-config-service

# Start the service with PostgreSQL and Redis
docker compose up -d

# Run database migrations
make migrate

# The gRPC service is now available at localhost:9090
```

### Helm (Kubernetes)

```bash
helm install central-config-service deploy/helm/central-config-service \
  --set database.writeUrl="postgres://user:pass@pg-primary:5432/centralconfig?sslmode=require" \
  --set database.readUrl="postgres://user:pass@pg-replica:5432/centralconfig?sslmode=require" \
  --set redis.url="redis://redis:6379" \
  --set auth.jwksUrl="https://your-idp/.well-known/jwks.json"
```

## Architecture

```
┌──────────┐     gRPC      ┌────────────────────────┐
│  Clients ├──────────────►│  Central Config Service │
└──────────┘               │                        │
                           │  ┌── SchemaService     │
                           │  ├── ConfigService     │
                           │  └── AuditService      │
                           └───┬──────────┬─────────┘
                               │          │
                          ┌────▼───┐  ┌───▼────┐
                          │ Postgres│  │ Redis  │
                          │        │  │ Cache + │
                          │        │  │ Pub/Sub │
                          └────────┘  └────────┘
```

Single binary exposing three gRPC services. Deploy with `--enable-services` to control which services run on each instance — scale read-heavy config instances independently from schema management.

See [system design docs](.claude/efforts/01-system-design.md) for detailed architecture.

## Configuration

The service is configured via environment variables:

| Variable | Description | Default |
|----------|------------|---------|
| `GRPC_PORT` | gRPC listen port | `9090` |
| `DB_WRITE_URL` | PostgreSQL primary connection string | required |
| `DB_READ_URL` | PostgreSQL read replica connection string | `DB_WRITE_URL` |
| `REDIS_URL` | Redis connection string | required |
| `ENABLE_SERVICES` | Services to enable: `schema`, `config`, `audit` | all |
| `JWT_JWKS_URL` | JWKS endpoint for JWT validation | required |
| `JWT_ISSUER` | Expected JWT issuer | optional |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | optional |

## API

The API is defined in Protocol Buffers under [`proto/`](proto/). Three gRPC services:

- **SchemaService** — create, version, and manage config schemas and tenants
- **ConfigService** — read/write config values, subscribe to changes, version management
- **AuditService** — query change history and usage statistics

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, build instructions, and contribution guidelines.

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.