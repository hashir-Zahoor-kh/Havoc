// Package redis wraps the Redis helpers used by Havoc: the global kill
// switch checked by every agent, and the per-service active-experiment
// lock that prevents two experiments from stacking on the same service.
package redis

import (
	goredis "github.com/redis/go-redis/v9"
)

// Client is a typed wrapper around go-redis exposing only the operations
// Havoc actually uses.
type Client struct {
	rdb *goredis.Client
}

// New constructs a Client for the given address, password, and db index.
func New(addr, password string, db int) *Client {
	return &Client{rdb: goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})}
}

// Close releases the underlying connection pool.
func (c *Client) Close() error { return c.rdb.Close() }

// Raw exposes the underlying go-redis client. Reserved for cases where the
// typed helpers below are insufficient.
func (c *Client) Raw() *goredis.Client { return c.rdb }
