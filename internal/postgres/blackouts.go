package postgres

import (
	"context"
	"fmt"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

// BlackoutRecord is the Postgres-shaped row for a blackout window. The
// safety package converts these into evaluatable windows via an adapter
// wired up in the control plane — keeping the persistence layer free of
// any dependency on the safety package.
type BlackoutRecord struct {
	ID              domain.ID
	Name            string
	CronExpression  string
	DurationMinutes int
}

// InsertBlackout persists a new blackout window. Name must be unique.
func (s *Store) InsertBlackout(ctx context.Context, r BlackoutRecord) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO blackout_windows (id, name, cron_expression, duration_minutes) VALUES ($1, $2, $3, $4)`,
		r.ID, r.Name, r.CronExpression, r.DurationMinutes,
	)
	if err != nil {
		return fmt.Errorf("insert blackout: %w", err)
	}
	return nil
}

// DeleteBlackout removes a blackout window by name.
func (s *Store) DeleteBlackout(ctx context.Context, name string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM blackout_windows WHERE name = $1`, name)
	if err != nil {
		return false, fmt.Errorf("delete blackout: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ListBlackouts returns every configured blackout window.
func (s *Store) ListBlackouts(ctx context.Context) ([]BlackoutRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, cron_expression, duration_minutes FROM blackout_windows ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list blackouts: %w", err)
	}
	defer rows.Close()
	var out []BlackoutRecord
	for rows.Next() {
		var r BlackoutRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.CronExpression, &r.DurationMinutes); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
