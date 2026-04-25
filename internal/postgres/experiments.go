package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
	"github.com/jackc/pgx/v5"
)

// InsertExperiment persists a new experiment record.
func (s *Store) InsertExperiment(ctx context.Context, e domain.Experiment) error {
	selector, err := json.Marshal(e.TargetSelector)
	if err != nil {
		return fmt.Errorf("marshal selector: %w", err)
	}
	params, err := json.Marshal(e.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	var scheduledFor any
	if !e.ScheduledFor.IsZero() {
		scheduledFor = e.ScheduledFor
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO experiments (
    id, created_at, scheduled_for, action_type, target_selector,
    target_namespace, duration_seconds, parameters, status, rejection_reason
) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8::jsonb, $9, NULLIF($10, ''))`,
		e.ID, e.CreatedAt, scheduledFor, string(e.ActionType), selector,
		e.TargetNamespace, e.DurationSeconds, params, string(e.Status), e.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("insert experiment: %w", err)
	}
	return nil
}

// UpdateStatus transitions an experiment to a new status. An empty reason
// clears any prior rejection reason.
func (s *Store) UpdateStatus(ctx context.Context, id domain.ID, status domain.Status, reason string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE experiments SET status = $2, rejection_reason = NULLIF($3, '') WHERE id = $1`,
		id, string(status), reason,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// ListExperiments returns experiments newest-first, up to limit rows.
func (s *Store) ListExperiments(ctx context.Context, limit int) ([]domain.Experiment, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, created_at, scheduled_for, action_type, target_selector,
       target_namespace, duration_seconds, parameters, status,
       COALESCE(rejection_reason, '')
FROM experiments
ORDER BY created_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	defer rows.Close()
	return scanExperiments(rows)
}

// GetExperiment fetches a single experiment by id. Returns (nil, nil) if
// no row matches.
func (s *Store) GetExperiment(ctx context.Context, id domain.ID) (*domain.Experiment, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, created_at, scheduled_for, action_type, target_selector,
       target_namespace, duration_seconds, parameters, status,
       COALESCE(rejection_reason, '')
FROM experiments WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
	}
	defer rows.Close()
	list, err := scanExperiments(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

func scanExperiments(rows pgx.Rows) ([]domain.Experiment, error) {
	var out []domain.Experiment
	for rows.Next() {
		var (
			e            domain.Experiment
			selectorRaw  []byte
			paramsRaw    []byte
			scheduledFor *time.Time
			actionType   string
			status       string
		)
		if err := rows.Scan(
			&e.ID, &e.CreatedAt, &scheduledFor, &actionType, &selectorRaw,
			&e.TargetNamespace, &e.DurationSeconds, &paramsRaw, &status,
			&e.RejectionReason,
		); err != nil {
			return nil, err
		}
		e.ActionType = domain.ActionType(actionType)
		e.Status = domain.Status(status)
		if scheduledFor != nil {
			e.ScheduledFor = *scheduledFor
		}
		if len(selectorRaw) > 0 {
			if err := json.Unmarshal(selectorRaw, &e.TargetSelector); err != nil {
				return nil, fmt.Errorf("unmarshal selector: %w", err)
			}
		}
		if len(paramsRaw) > 0 {
			if err := json.Unmarshal(paramsRaw, &e.Parameters); err != nil {
				return nil, fmt.Errorf("unmarshal parameters: %w", err)
			}
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
