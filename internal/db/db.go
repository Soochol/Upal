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
`
