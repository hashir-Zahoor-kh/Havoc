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
// Prefer ReleaseLockIfOwner when an experiment id is available — it
// avoids unlocking a newer experiment that has already taken over the
// slot.
func (c *Client) ReleaseLock(ctx context.Context, service string) error {
	return c.rdb.Del(ctx, activeLockKey(service)).Err()
}

// releaseIfOwnerScript is a compare-and-delete: only DELs the key if its
// value equals the expected experiment id. Atomic on the Redis side.
var releaseIfOwnerScript = goredis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`)

// ReleaseLockIfOwner releases the active-experiment lock only if the
// stored value matches experimentID. Returns true when this caller's
// experiment owned the lock and it was released.
func (c *Client) ReleaseLockIfOwner(ctx context.Context, service, experimentID string) (bool, error) {
	res, err := releaseIfOwnerScript.Run(ctx, c.rdb, []string{activeLockKey(service)}, experimentID).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
