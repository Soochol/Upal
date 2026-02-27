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
CREATE TABLE IF NOT EXISTS users (
    id             TEXT PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL DEFAULT '',
    avatar_url     TEXT NOT NULL DEFAULT '',
    oauth_provider TEXT NOT NULL,
    oauth_id       TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(oauth_provider, oauth_id)
);

CREATE TABLE IF NOT EXISTS workflows (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
    name        TEXT NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1,
    definition  JSONB NOT NULL,
    visibility  TEXT NOT NULL DEFAULT 'private',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Ensure user_id exists before creating composite index
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
-- Drop old single-column unique constraint and replace with per-user unique
ALTER TABLE workflows DROP CONSTRAINT IF EXISTS workflows_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflows_user_name ON workflows(user_id, name);

CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    state       JSONB NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'running',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
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
    user_id      TEXT NOT NULL DEFAULT 'default',
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size         BIGINT NOT NULL DEFAULT 0,
    storage_path TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS runs (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL DEFAULT 'default',
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
    token_usage    JSONB DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ
);
ALTER TABLE runs ADD COLUMN IF NOT EXISTS token_usage JSONB DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_runs_workflow ON runs(workflow_name);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at);

CREATE TABLE IF NOT EXISTS schedules (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL DEFAULT 'default',
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
    user_id        TEXT NOT NULL DEFAULT 'default',
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
    user_id     TEXT NOT NULL DEFAULT 'default',
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    stages      JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL DEFAULT 'default',
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
    user_id  TEXT NOT NULL DEFAULT 'default',
    name     TEXT NOT NULL,
    type     TEXT NOT NULL,
    host     TEXT NOT NULL DEFAULT '',
    port     INTEGER NOT NULL DEFAULT 0,
    login    TEXT NOT NULL DEFAULT '',
    password TEXT NOT NULL DEFAULT '',
    token    TEXT NOT NULL DEFAULT '',
    extras   JSONB NOT NULL DEFAULT '{}'
);

ALTER TABLE runs ADD COLUMN IF NOT EXISTS session_id TEXT;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS workflow_definition JSONB;

CREATE TABLE IF NOT EXISTS content_sessions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL DEFAULT 'default',
    pipeline_id  TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'collecting',
    trigger_type TEXT NOT NULL DEFAULT 'manual',
    source_count INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_content_sessions_pipeline_id ON content_sessions(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_content_sessions_status ON content_sessions(status);

CREATE TABLE IF NOT EXISTS source_fetches (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
    session_id  TEXT NOT NULL REFERENCES content_sessions(id) ON DELETE CASCADE,
    tool_name   TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'static',
    label       TEXT NOT NULL DEFAULT '',
    item_count  INTEGER NOT NULL DEFAULT 0,
    raw_items   JSONB NOT NULL DEFAULT '[]',
    error       TEXT,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_source_fetches_session_id ON source_fetches(session_id);
ALTER TABLE source_fetches ADD COLUMN IF NOT EXISTS label TEXT NOT NULL DEFAULT '';
ALTER TABLE source_fetches ADD COLUMN IF NOT EXISTS item_count INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS llm_analyses (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL DEFAULT 'default',
    session_id       TEXT NOT NULL REFERENCES content_sessions(id) ON DELETE CASCADE,
    raw_item_count   INTEGER NOT NULL DEFAULT 0,
    filtered_count   INTEGER NOT NULL DEFAULT 0,
    summary          TEXT NOT NULL DEFAULT '',
    insights         JSONB NOT NULL DEFAULT '[]',
    suggested_angles JSONB NOT NULL DEFAULT '[]',
    overall_score    INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_llm_analyses_session_id ON llm_analyses(session_id);

CREATE TABLE IF NOT EXISTS published_content (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL DEFAULT 'default',
    workflow_run_id  TEXT NOT NULL,
    session_id       TEXT NOT NULL,
    channel          TEXT NOT NULL,
    external_url     TEXT NOT NULL DEFAULT '',
    title            TEXT NOT NULL DEFAULT '',
    published_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_published_content_session_id ON published_content(session_id);
CREATE INDEX IF NOT EXISTS idx_published_content_channel ON published_content(channel);

CREATE TABLE IF NOT EXISTS surge_events (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
    keyword     TEXT NOT NULL,
    pipeline_id TEXT NOT NULL DEFAULT '',
    multiplier  DOUBLE PRECISION NOT NULL DEFAULT 0,
    sources     JSONB NOT NULL DEFAULT '[]',
    dismissed   BOOLEAN NOT NULL DEFAULT false,
    session_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_surge_events_dismissed ON surge_events(dismissed);

ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS schedule TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS workflows JSONB NOT NULL DEFAULT '[]';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS context JSONB;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS parent_session_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS workflow_results (
    session_id  TEXT PRIMARY KEY REFERENCES content_sessions(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL DEFAULT 'default',
    results     JSONB NOT NULL DEFAULT '[]',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_providers (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL DEFAULT 'default',
    name       TEXT NOT NULL,
    category   TEXT NOT NULL,
    type       TEXT NOT NULL,
    model      TEXT NOT NULL DEFAULT '',
    api_key    TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT FALSE
);
-- Ensure user_id exists before creating composite indexes
ALTER TABLE ai_providers ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
-- Drop old single-column unique indexes and replace with per-user unique
DROP INDEX IF EXISTS idx_ai_provider_name;
DROP INDEX IF EXISTS ai_providers_name_key;
DROP INDEX IF EXISTS idx_ai_provider_default_per_category;
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_provider_user_name
    ON ai_providers(user_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_provider_user_default_per_category
    ON ai_providers(user_id, category) WHERE is_default = TRUE;
ALTER TABLE ai_providers ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS mcp_servers (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_user_id ON mcp_servers(user_id);

-- Add user_id to existing tables (idempotent)
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE events ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE runs ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE triggers ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE pipeline_runs ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE connections ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE source_fetches ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE llm_analyses ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE published_content ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE surge_events ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE workflow_results ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE ai_providers ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';

-- Add user_id indexes
CREATE INDEX IF NOT EXISTS idx_workflows_user_id ON workflows(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
CREATE INDEX IF NOT EXISTS idx_assets_user_id ON assets(user_id);
CREATE INDEX IF NOT EXISTS idx_runs_user_id ON runs(user_id);
CREATE INDEX IF NOT EXISTS idx_schedules_user_id ON schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_triggers_user_id ON triggers(user_id);
CREATE INDEX IF NOT EXISTS idx_pipelines_user_id ON pipelines(user_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_user_id ON pipeline_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_connections_user_id ON connections(user_id);
CREATE INDEX IF NOT EXISTS idx_content_sessions_user_id ON content_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_source_fetches_user_id ON source_fetches(user_id);
CREATE INDEX IF NOT EXISTS idx_llm_analyses_user_id ON llm_analyses(user_id);
CREATE INDEX IF NOT EXISTS idx_published_content_user_id ON published_content(user_id);
CREATE INDEX IF NOT EXISTS idx_surge_events_user_id ON surge_events(user_id);
CREATE INDEX IF NOT EXISTS idx_workflow_results_user_id ON workflow_results(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_providers_user_id ON ai_providers(user_id);

CREATE TABLE IF NOT EXISTS upal_sessions (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL DEFAULT 'default',
    name            TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    sources         JSONB NOT NULL DEFAULT '[]',
    schedule        TEXT NOT NULL DEFAULT '',
    model           TEXT NOT NULL DEFAULT '',
    workflows       JSONB NOT NULL DEFAULT '[]',
    context         JSONB,
    stages          JSONB NOT NULL DEFAULT '[]',
    status          TEXT NOT NULL DEFAULT 'draft',
    thumbnail_svg   TEXT NOT NULL DEFAULT '',
    last_collected_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_upal_sessions_user_id ON upal_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_upal_sessions_status ON upal_sessions(status);

CREATE TABLE IF NOT EXISTS upal_runs (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL DEFAULT 'default',
    session_id   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'collecting',
    trigger_type TEXT NOT NULL DEFAULT 'manual',
    source_count INTEGER NOT NULL DEFAULT 0,
    schedule_id  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_upal_runs_user_id ON upal_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_upal_runs_session_id ON upal_runs(session_id);
CREATE INDEX IF NOT EXISTS idx_upal_runs_status ON upal_runs(status);

ALTER TABLE upal_runs ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE upal_runs ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]';
ALTER TABLE upal_runs ADD COLUMN IF NOT EXISTS workflows JSONB NOT NULL DEFAULT '[]';
ALTER TABLE upal_runs ADD COLUMN IF NOT EXISTS context JSONB;
ALTER TABLE upal_runs ADD COLUMN IF NOT EXISTS schedule TEXT NOT NULL DEFAULT '';
ALTER TABLE upal_runs ADD COLUMN IF NOT EXISTS schedule_active BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS upal_workflow_runs (
    run_id      TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
    results     JSONB NOT NULL DEFAULT '[]',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_upal_workflow_runs_user_id ON upal_workflow_runs(user_id);

CREATE TABLE IF NOT EXISTS upal_source_fetches (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
    run_id      TEXT NOT NULL,
    tool_name   TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'static',
    label       TEXT NOT NULL DEFAULT '',
    item_count  INTEGER NOT NULL DEFAULT 0,
    raw_items   JSONB NOT NULL DEFAULT '[]',
    error       TEXT,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_upal_source_fetches_user_id ON upal_source_fetches(user_id);
CREATE INDEX IF NOT EXISTS idx_upal_source_fetches_run_id ON upal_source_fetches(run_id);

CREATE TABLE IF NOT EXISTS upal_llm_analyses (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL DEFAULT 'default',
    run_id           TEXT NOT NULL,
    raw_item_count   INTEGER NOT NULL DEFAULT 0,
    filtered_count   INTEGER NOT NULL DEFAULT 0,
    summary          TEXT NOT NULL DEFAULT '',
    insights         JSONB NOT NULL DEFAULT '[]',
    suggested_angles JSONB NOT NULL DEFAULT '[]',
    overall_score    INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_upal_llm_analyses_user_id ON upal_llm_analyses(user_id);
CREATE INDEX IF NOT EXISTS idx_upal_llm_analyses_run_id ON upal_llm_analyses(run_id);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    device_info TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    replaced_by TEXT
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
`
