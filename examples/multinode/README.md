# Multinode

Split API and Worker deployment using PostgreSQL for all storage. No Redis dependency.

## Architecture

```
┌─────────────┐
│  API        │
│  (HTTP)     │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│ PostgreSQL  │◄────│   Worker    │
│ (queue/lock)│     │ (scheduler) │
└──────┬──────┘     └─────────────┘
       │
       ▼
┌─────────────┐
│    Vespa    │
└─────────────┘
```

## Usage

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f sercha-api sercha-worker

# Stop
docker compose down
```

## Services

| Service | Port | Purpose |
|---------|------|---------|
| sercha-api | 8080 | HTTP API server |
| sercha-worker | - | Background task processing |
| postgres | 5432 | Database, queue, locks |
| vespa | 19071 | Search engine |

## Scaling

This example runs a single API and single Worker. For horizontal scaling with multiple workers, use [multinode-redis](../multinode-redis/) instead.

PostgreSQL advisory locks work for distributed locking, but Redis provides more robust coordination for multi-worker deployments.

## When to Use

- Simple production deployments
- When Redis is not available
- Single worker is sufficient

## Configuration

| Variable | Value | Description |
|----------|-------|-------------|
| `RUN_MODE` | `api` / `worker` | Container role |
| `SCHEDULER_ENABLED` | `true` | Worker runs scheduler |
| `DB_MAX_OPEN_CONNS` | `10` | Connections per container |

See [Configuration Reference](../../docs/core/architecture/run-modes/configuration.md) for all options.
