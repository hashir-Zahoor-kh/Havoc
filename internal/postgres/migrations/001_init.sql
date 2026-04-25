-- 001_init.sql
-- Base schema for Havoc. Three tables:
--   experiments        — every experiment ever requested
--   experiment_results — what each agent actually did
--   blackout_windows   — configured chaos-forbidden time ranges

CREATE TABLE IF NOT EXISTS experiments (
    id                UUID        PRIMARY KEY,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_for     TIMESTAMPTZ,
    action_type       TEXT        NOT NULL,
    target_selector   JSONB       NOT NULL,
    target_namespace  TEXT        NOT NULL,
    target_pods       TEXT[]      NOT NULL DEFAULT '{}',
    duration_seconds  INT         NOT NULL,
    parameters        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    status            TEXT        NOT NULL,
    rejection_reason  TEXT
);

CREATE INDEX IF NOT EXISTS experiments_created_at_idx  ON experiments (created_at DESC);
CREATE INDEX IF NOT EXISTS experiments_status_idx      ON experiments (status);
CREATE INDEX IF NOT EXISTS experiments_namespace_idx   ON experiments (target_namespace);

CREATE TABLE IF NOT EXISTS experiment_results (
    id             UUID        PRIMARY KEY,
    experiment_id  UUID        NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    agent_node     TEXT        NOT NULL,
    affected_pods  TEXT[]      NOT NULL DEFAULT '{}',
    started_at     TIMESTAMPTZ NOT NULL,
    completed_at   TIMESTAMPTZ NOT NULL,
    outcome        TEXT        NOT NULL,
    error_message  TEXT
);

CREATE INDEX IF NOT EXISTS experiment_results_experiment_idx ON experiment_results (experiment_id);
CREATE INDEX IF NOT EXISTS experiment_results_outcome_idx    ON experiment_results (outcome);

CREATE TABLE IF NOT EXISTS blackout_windows (
    id                UUID PRIMARY KEY,
    name              TEXT NOT NULL UNIQUE,
    cron_expression   TEXT NOT NULL,
    duration_minutes  INT  NOT NULL CHECK (duration_minutes > 0)
);
