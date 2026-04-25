package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// KillSwitchKey is the single Redis key agents consult before executing
// any chaos action.
const KillSwitchKey = "havoc:killswitch"

// KillSwitchEngaged reports whether the global kill switch is currently set.
func (c *Client) KillSwitchEngaged(ctx context.Context) (bool, error) {
	_, err := c.rdb.Get(ctx, KillSwitchKey).Result()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// EngageKillSwitch sets the global kill switch. A zero ttl means permanent
// until cleared.
func (c *Client) EngageKillSwitch(ctx context.Context, ttl time.Duration) error {
	return c.rdb.Set(ctx, KillSwitchKey, "1", ttl).Err()
}

// DisengageKillSwitch clears the global kill switch.
func (c *Client) DisengageKillSwitch(ctx context.Context) error {
	return c.rdb.Del(ctx, KillSwitchKey).Err()
}
