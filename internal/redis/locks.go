package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// activeLockKey is the Redis key used to mark a service as having an
// in-flight experiment. Keyed by the service name so two experiments
// targeting different services can run concurrently.
func activeLockKey(service string) string {
	return fmt.Sprintf("havoc:active:%s", service)
}

// AcquireLock attempts to record an active experiment on the given service.
// Returns (true, nil) if the lock was freshly acquired, (false, nil) if a
// lock is already present, and (false, err) on a Redis failure.
func (c *Client) AcquireLock(ctx context.Context, service, experimentID string, ttl time.Duration) (bool, error) {
	ok, err := c.rdb.SetNX(ctx, activeLockKey(service), experimentID, ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// IsLocked reports whether the given service currently holds an active lock.
func (c *Client) IsLocked(ctx context.Context, service string) (bool, error) {
	_, err := c.rdb.Get(ctx, activeLockKey(service)).Result()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReleaseLock removes the active-experiment lock for the given service.
func (c *Client) ReleaseLock(ctx context.Context, service string) error {
	return c.rdb.Del(ctx, activeLockKey(service)).Err()
}
