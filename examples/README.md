# Deployment Examples

Ready-to-use Docker Compose configurations for different deployment scenarios.

## Examples

| Example | Description | Use Case |
|---------|-------------|----------|
| [quickstart](./quickstart/) | Single container with Redis | Development, small teams |
| [multinode](./multinode/) | Split API + Worker (no Redis) | Simple production |
| [multinode-ha](./multinode-ha/) | HA with load balancing + Redis | Scalable production |

## Quick Comparison

| Feature | quickstart | multinode | multinode-ha |
|---------|------------|-----------|--------------|
| API containers | 1 | 1 | 2+ |
| Worker containers | 0 (combined) | 1 | 2+ |
| Load balancer | No | No | Yes (nginx) |
| Redis required | Yes | No | Yes |
| Horizontal scaling | No | Limited | Yes |

## Prerequisites

- Docker and Docker Compose
- 4GB+ RAM (Vespa requires ~2GB)

## Usage

```bash
# Navigate to desired example
cd examples/quickstart

# Start services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f

# Stop
docker compose down

# Stop and remove volumes
docker compose down -v
```

## Helm Charts

Each example includes a `helm/` directory for Kubernetes deployments (coming soon).

## Documentation

- [Run Modes Overview](../docs/core/architecture/run-modes/overview.md)
- [Configuration Reference](../docs/core/architecture/run-modes/configuration.md)
- [Scaling Patterns](../docs/core/architecture/run-modes/scaling.md)
