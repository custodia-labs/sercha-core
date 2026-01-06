# Quickstart

Single container deployment with all dependencies. Ideal for development and small teams.

## Architecture

```
┌─────────────────────────────────────────┐
│            sercha (combined)            │
│         API + Worker + Scheduler        │
└──────────────────┬──────────────────────┘
                   │
          ┌────────┴────────┐
          │                 │
    ┌─────▼─────┐     ┌─────▼─────┐
    │ PostgreSQL │     │   Vespa   │
    └───────────┘     └───────────┘
```

## Quick Start

```bash
# Start all services
docker compose up -d

# Wait for services to be healthy (1-2 minutes for Vespa)
docker compose ps

# Run the interactive setup script
./quickstart.sh
```

The `quickstart.sh` script will guide you through:
- Creating an admin user
- Configuring GitHub OAuth
- Connecting a repository
- Running your first search

### Environment Variables

You can pre-set these to skip the prompts:

```bash
export GITHUB_CLIENT_ID="your-client-id"
export GITHUB_CLIENT_SECRET="your-client-secret"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="your-password"
./quickstart.sh
```

## Services

| Service | Port | Purpose |
|---------|------|---------|
| sercha | 8080 | API server |
| postgres | 5432 | Database |
| vespa | 19071 | Search engine |

## Full Documentation

For detailed documentation including API reference, see the **[Quickstart Guide](https://docs.sercha.dev/core/quickstart)**.

## Stopping Services

```bash
# Stop services (preserves data)
docker compose down

# Stop and remove all data
docker compose down -v
```
