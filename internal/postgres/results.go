package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
	"github.com/jackc/pgx/v5"
)

// InsertResult persists an agent-reported experiment result. The insert
// is idempotent on the result id so duplicate Kafka deliveries do not
// produce duplicate rows. Returns true if a new row was inserted, false
// if the id was already present.
func (s *Store) InsertResult(ctx context.Context, r domain.ExperimentResult) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
INSERT INTO experiment_results (
    id, experiment_id, agent_node, affected_pods, started_at,
    completed_at, outcome, error_message
) VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))
ON CONFLICT (id) DO NOTHING`,
		r.ID, r.ExperimentID, r.AgentNode, r.AffectedPods, r.StartedAt,
		r.CompletedAt, string(r.Outcome), r.ErrorMessage,
	)
	if err != nil {
		return false, fmt.Errorf("insert result: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ListResults returns the most recent results for an experiment.
func (s *Store) ListResults(ctx context.Context, experimentID domain.ID) ([]domain.ExperimentResult, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, experiment_id, agent_node, affected_pods, started_at,
       completed_at, outcome, COALESCE(error_message, '')
FROM experiment_results
WHERE experiment_id = $1
ORDER BY started_at ASC`, experimentID)
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}
	defer rows.Close()
	return scanResults(rows)
}

func scanResults(rows pgx.Rows) ([]domain.ExperimentResult, error) {
	var out []domain.ExperimentResult
	for rows.Next() {
		var (
			r           domain.ExperimentResult
			outcome     string
			startedAt   time.Time
			completedAt time.Time
		)
		if err := rows.Scan(&r.ID, &r.ExperimentID, &r.AgentNode, &r.AffectedPods,
			&startedAt, &completedAt, &outcome, &r.ErrorMessage); err != nil {
			return nil, err
		}
		r.StartedAt = startedAt
		r.CompletedAt = completedAt
		r.Outcome = domain.Outcome(outcome)
		out = append(out, r)
	}
	return out, rows.Err()
}
