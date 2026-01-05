package postgres

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.DistributedLock = (*AdvisoryLock)(nil)

// AdvisoryLock implements DistributedLock using PostgreSQL advisory locks.
//
// IMPORTANT LIMITATIONS:
// - Advisory locks are connection-scoped, not TTL-based
// - If the connection is lost, the lock is automatically released
// - TTL parameter is ignored (locks don't expire automatically)
// - Extend is a no-op since locks don't have TTL
//
// For production multi-worker deployments, Redis locks are recommended.
// This is provided as a fallback when Redis is unavailable.
type AdvisoryLock struct {
	db *DB
}

// NewAdvisoryLock creates a new PostgreSQL advisory lock adapter.
func NewAdvisoryLock(db *DB) *AdvisoryLock {
	return &AdvisoryLock{db: db}
}

// hashLockName converts a string lock name to a 64-bit integer for PostgreSQL advisory locks.
// Uses FNV-1a hash for consistent, well-distributed values.
func hashLockName(name string) int64 {
	h := fnv.New64a()
	h.Write([]byte("sercha:lock:" + name))
	return int64(h.Sum64())
}

// Acquire attempts to acquire a named advisory lock.
// Uses pg_try_advisory_lock which returns immediately without blocking.
//
// Note: The TTL parameter is ignored - PostgreSQL advisory locks don't have TTL.
// The lock is held until explicitly released or the connection closes.
func (l *AdvisoryLock) Acquire(ctx context.Context, name string, ttl time.Duration) (bool, error) {
	lockID := hashLockName(name)

	var acquired bool
	err := l.db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired)
	if err != nil {
		return false, err
	}
	return acquired, nil
}

// Release releases a named advisory lock.
// Uses pg_advisory_unlock to release the lock.
// Safe to call even if the lock is not held (returns false but no error).
func (l *AdvisoryLock) Release(ctx context.Context, name string) error {
	lockID := hashLockName(name)

	var released bool
	err := l.db.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", lockID).Scan(&released)
	if err != nil {
		return err
	}
	// Note: released=false means lock wasn't held, but that's not an error
	return nil
}

// Extend is a no-op for PostgreSQL advisory locks since they don't have TTL.
// Advisory locks are held until explicitly released or the connection closes.
func (l *AdvisoryLock) Extend(ctx context.Context, name string, ttl time.Duration) error {
	// No-op: PostgreSQL advisory locks don't expire
	// We could check if we hold the lock, but that adds complexity
	// for little benefit in the scheduler use case
	return nil
}

// Ping checks if the PostgreSQL backend is healthy.
func (l *AdvisoryLock) Ping(ctx context.Context) error {
	return l.db.PingContext(ctx)
}
