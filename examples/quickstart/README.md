# Quickstart

Single container deployment with all dependencies. Ideal for development and small teams.

## Architecture

```
┌─────────────────────────────────────────┐
│            sercha (combined)            │
│         API + Worker + Scheduler        │
└──────────────────┬──────────────────────┘
                   │
     ┌─────────────┼─────────────┐
     │             │             │
┌────▼────┐  ┌─────▼─────┐  ┌────▼────┐
│PostgreSQL│  │   Redis   │  │  Vespa  │
└─────────┘  └───────────┘  └─────────┘
```

## Usage

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f sercha

# Stop
docker compose down
```

## Services

| Service | Port | Purpose |
|---------|------|---------|
| sercha | 8080 | API server |
| postgres | 5432 | Database |
| redis | 6379 | Sessions, queue |
| vespa | 19071 | Search engine |

## Initial Setup

Once running, create the first admin user:

```bash
curl -X POST http://localhost:8080/api/v1/setup \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "changeme", "name": "Admin"}'
```

## Configuration

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | - | Token signing secret (change in production) |
| `PORT` | 8080 | API port |

See [Configuration Reference](../../docs/core/architecture/run-modes/configuration.md) for all options.
