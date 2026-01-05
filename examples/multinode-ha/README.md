# Multinode HA

High availability deployment with multiple API and Worker instances, load balancing, and distributed coordination.

## What's HA

This example makes **Sercha containers** highly available:

| Component | HA Feature |
|-----------|------------|
| API | Multiple instances behind load balancer |
| Workers | Multiple instances processing from shared queue |
| Scheduler | Lock-based coordination prevents duplicates |

**External dependencies** (PostgreSQL, Redis, Vespa) are shown as single instances for simplicity. For full end-to-end HA, configure these separately using managed services or replication.

## Architecture

```
                    ┌─────────────┐
                    │    nginx    │ :8080
                    │ (balancer)  │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
      ┌────▼────┐     ┌────▼────┐
      │  API 1  │     │  API 2  │
      └────┬────┘     └────┬────┘
           │               │
           └───────┬───────┘
                   │
     ┌─────────────┼─────────────┐
     │             │             │
┌────▼────┐  ┌─────▼─────┐  ┌────▼────┐
│PostgreSQL│  │   Redis   │  │  Vespa  │
└─────────┘  │(queue/lock)│  └─────────┘
             └─────┬──────┘
                   │
       ┌───────────┼───────────┐
       │                       │
  ┌────▼────┐             ┌────▼────┐
  │Worker 1 │             │Worker 2 │
  └─────────┘             └─────────┘
```

## Usage

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f

# Check health through load balancer
curl http://localhost:8080/health

# Stop
docker compose down
```

## Failover Behavior

| Failure | Result |
|---------|--------|
| API container dies | nginx routes to remaining instances |
| Worker container dies | Other workers continue processing queue |
| Scheduler holder dies | Lock expires (60s), another worker takes over |

## Distributed Locking

Both workers have `SCHEDULER_ENABLED=true` but only one schedules per cycle:

1. Workers compete for Redis lock `sercha:lock:scheduler`
2. Winner schedules tasks, others skip
3. Lock auto-expires after 60s for crash recovery

## Scaling

Add more instances by duplicating services:

```yaml
sercha-api-3:
  # ... same config as api-1/api-2

sercha-worker-3:
  # ... same config as worker-1/worker-2
```

Update `nginx.conf` upstream to include new API instances.

## External Dependencies HA

For full production HA, configure external services separately:

| Service | HA Options |
|---------|------------|
| PostgreSQL | Managed (RDS, Cloud SQL) or streaming replication |
| Redis | Redis Sentinel, Cluster, or managed (ElastiCache) |
| Vespa | Multi-node Vespa cluster |
| nginx | Multiple instances + keepalived, or managed LB |

## Connection Pool Sizing

With multiple containers, tune database connections:

| Setting | Value | Reason |
|---------|-------|--------|
| `DB_MAX_OPEN_CONNS` | 10 | Per container |
| Total connections | 40 | 4 containers × 10 |

Ensure PostgreSQL `max_connections` exceeds total.

## Configuration

| Variable | Value | Description |
|----------|-------|-------------|
| `RUN_MODE` | `api` / `worker` | Container role |
| `REDIS_URL` | Required | Enables distributed locking |
| `SCHEDULER_LOCK_REQUIRED` | `true` | Prevents duplicate scheduling |

See [Configuration Reference](../../docs/core/architecture/run-modes/configuration.md) for all options.
