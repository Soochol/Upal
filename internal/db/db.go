package db

import (
	"context"
	"database/sql"
	"fmt"
)

// DB wraps a database/sql connection pool for PostgreSQL.
type DB struct {
	Pool *sql.DB
}

// New creates a new database connection.
// The caller must import a PostgreSQL driver (e.g., _ "github.com/lib/pq").
func New(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(5)

	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the connection pool.
func (d *DB) Close() error {
	return d.Pool.Close()
}

// Migrate runs the database schema migrations.
func (d *DB) Migrate(ctx context.Context) error {
	_, err := d.Pool.ExecContext(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

const migrationSQL = `
CREATE TABLE IF NOT EXISTS workflows (
    id          TEXT PRIMARY KEY,
    name        TEXT UNIQUE NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1,
    definition  JSONB NOT NULL,
    visibility  TEXT NOT NULL DEFAULT 'private',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    state       JSONB NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'running',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    node_id     TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_session_id ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_sessions_workflow_id ON sessions(workflow_id);

CREATE TABLE IF NOT EXISTS assets (
    id           TEXT PRIMARY KEY,
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size         BIGINT NOT NULL DEFAULT 0,
    storage_path TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS runs (
    id             TEXT PRIMARY KEY,
    workflow_name  TEXT NOT NULL,
    trigger_type   TEXT NOT NULL DEFAULT 'manual',
    trigger_ref    TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'pending',
    inputs         JSONB NOT NULL DEFAULT '{}',
    outputs        JSONB NOT NULL DEFAULT '{}',
    error          TEXT,
    retry_of       TEXT,
    retry_count    INTEGER NOT NULL DEFAULT 0,
    node_runs      JSONB NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_runs_workflow ON runs(workflow_name);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at);

CREATE TABLE IF NOT EXISTS schedules (
    id             TEXT PRIMARY KEY,
    workflow_name  TEXT NOT NULL DEFAULT '',
    pipeline_id    TEXT NOT NULL DEFAULT '',
    cron_expr      TEXT NOT NULL,
    inputs         JSONB NOT NULL DEFAULT '{}',
    enabled        BOOLEAN NOT NULL DEFAULT true,
    timezone       TEXT NOT NULL DEFAULT 'UTC',
    retry_policy   JSONB,
    next_run_at    TIMESTAMPTZ NOT NULL,
    last_run_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS pipeline_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS triggers (
    id             TEXT PRIMARY KEY,
    workflow_name  TEXT NOT NULL DEFAULT '',
    pipeline_id    TEXT NOT NULL DEFAULT '',
    type           TEXT NOT NULL,
    config         JSONB NOT NULL DEFAULT '{}',
    enabled        BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE triggers ADD COLUMN IF NOT EXISTS pipeline_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS pipelines (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    stages      JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id            TEXT PRIMARY KEY,
    pipeline_id   TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'pending',
    current_stage TEXT NOT NULL DEFAULT '',
    stage_results JSONB NOT NULL DEFAULT '{}',
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_pipeline_id ON pipeline_runs(pipeline_id);

CREATE TABLE IF NOT EXISTS connections (
    id       TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    type     TEXT NOT NULL,
    host     TEXT NOT NULL DEFAULT '',
    port     INTEGER NOT NULL DEFAULT 0,
    login    TEXT NOT NULL DEFAULT '',
    password TEXT NOT NULL DEFAULT '',
    token    TEXT NOT NULL DEFAULT '',
    extras   JSONB NOT NULL DEFAULT '{}'
);
`
