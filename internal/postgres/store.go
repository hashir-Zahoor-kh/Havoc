// Package postgres wraps the pgx connection pool and exposes the queries
// Havoc needs. Schema migrations live under /migrations and are applied on
// startup via Migrate.
package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the typed wrapper around the pgx pool used by all components.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a connection pool against dsn and pings it before returning.
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the connection pool.
func (s *Store) Close() { s.pool.Close() }

// Pool exposes the underlying pgxpool. Reserved for ad-hoc queries.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// MigrationsFS returns a read-only filesystem containing the baked-in
// migrations. Tests and alternate callers can override by passing their
// own fs.FS to MigrateFS.
func MigrationsFS() fs.FS {
	sub, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		// The Sub call can only fail if the embed directive changes.
		panic(err)
	}
	return sub
}

// Migrate applies every *.sql file in the embedded migrations FS in
// lexicographic order. Each file is executed as a single statement batch.
func (s *Store) Migrate(ctx context.Context) error {
	return s.MigrateFS(ctx, MigrationsFS())
}

// MigrateFS applies every *.sql file in the provided filesystem in
// lexicographic order.
func (s *Store) MigrateFS(ctx context.Context, mfs fs.FS) error {
	var files []string
	err := fs.WalkDir(mfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk migrations: %w", err)
	}
	sort.Strings(files)
	for _, f := range files {
		body, err := fs.ReadFile(mfs, f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := s.pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", f, err)
		}
	}
	return nil
}
