package driven

import (
	"context"
	"time"
)

// DistributedLock provides distributed locking for coordinating work across instances.
// This is used to prevent duplicate execution in multi-instance deployments.
type DistributedLock interface {
	// Acquire attempts to acquire a named lock with the given TTL.
	// Returns true if the lock was successfully acquired, false if already held by another instance.
	// The lock will automatically expire after TTL (implementation dependent).
	Acquire(ctx context.Context, name string, ttl time.Duration) (acquired bool, err error)

	// Release releases a named lock.
	// This is best-effort; implementations with TTL will auto-expire anyway.
	// Safe to call even if the lock is not held or has expired.
	Release(ctx context.Context, name string) error

	// Extend extends the TTL of a currently held lock.
	// Returns error if the lock is not held by this instance.
	// Note: Not all implementations support TTL extension (e.g., PostgreSQL advisory locks).
	Extend(ctx context.Context, name string, ttl time.Duration) error

	// Ping checks if the lock backend is healthy.
	Ping(ctx context.Context) error
}
