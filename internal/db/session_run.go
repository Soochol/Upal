package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// --- SessionRun (upal_runs table) ---

// marshalRunConfig serializes the JSONB fields of a Run for INSERT/UPDATE.
func marshalRunConfig(r *upal.Run) ([]byte, []byte, *string) {
	sourcesJSON, _ := json.Marshal(r.Sources)
	workflowsJSON, _ := json.Marshal(r.Workflows)
	var contextJSON *string
	if r.Context != nil {
		b, _ := json.Marshal(r.Context)
		s := string(b)
		contextJSON = &s
	}
	return sourcesJSON, workflowsJSON, contextJSON
}

// scanSessionRun scans a single row into a Run.
func scanSessionRun(scanner interface{ Scan(...any) error }) (*upal.Run, error) {
	var r upal.Run
	var status string
	var sourcesJSON, workflowsJSON []byte
	var contextJSON sql.NullString
	if err := scanner.Scan(
		&r.ID, &r.SessionID, &r.Name, &status, &r.TriggerType, &r.SourceCount,
		&r.ScheduleID, &sourcesJSON, &workflowsJSON, &contextJSON,
		&r.Schedule, &r.ScheduleActive, &r.CreatedAt, &r.ReviewedAt,
	); err != nil {
		return nil, err
	}
	r.Status = upal.SessionRunStatus(status)
	if len(sourcesJSON) > 0 {
		if err := json.Unmarshal(sourcesJSON, &r.Sources); err != nil {
			return nil, fmt.Errorf("unmarshal run sources: %w", err)
		}
	}
	if len(workflowsJSON) > 0 {
		if err := json.Unmarshal(workflowsJSON, &r.Workflows); err != nil {
			return nil, fmt.Errorf("unmarshal run workflows: %w", err)
		}
	}
	if contextJSON.Valid && contextJSON.String != "" {
		var ctx upal.SessionContext
		if err := json.Unmarshal([]byte(contextJSON.String), &ctx); err != nil {
			return nil, fmt.Errorf("unmarshal run context: %w", err)
		}
		r.Context = &ctx
	}
	return &r, nil
}

// scanSessionRuns scans rows into a slice of Runs.
func scanSessionRuns(rows *sql.Rows) ([]*upal.Run, error) {
	var result []*upal.Run
	for rows.Next() {
		r, err := scanSessionRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan upal_run: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

const upalRunColumns = `id, session_id, name, status, trigger_type, source_count, schedule_id, sources, workflows, context, schedule, schedule_active, created_at, reviewed_at`

func (d *DB) CreateSessionRun(ctx context.Context, userID string, r *upal.Run) error {
	sourcesJSON, workflowsJSON, contextJSON := marshalRunConfig(r)
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO upal_runs (id, user_id, session_id, name, status, trigger_type, source_count, schedule_id, sources, workflows, context, schedule, schedule_active, created_at, reviewed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		r.ID, userID, r.SessionID, r.Name, string(r.Status), r.TriggerType, r.SourceCount,
		r.ScheduleID, sourcesJSON, workflowsJSON, contextJSON, r.Schedule, r.ScheduleActive,
		r.CreatedAt, r.ReviewedAt,
	)
	if err != nil {
		return fmt.Errorf("insert upal_run: %w", err)
	}
	return nil
}

func (d *DB) GetSessionRun(ctx context.Context, userID string, id string) (*upal.Run, error) {
	row := d.Pool.QueryRowContext(ctx,
		`SELECT `+upalRunColumns+` FROM upal_runs WHERE id = $1 AND user_id = $2`, id, userID,
	)
	r, err := scanSessionRun(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get upal_run: %w", err)
	}
	return r, nil
}

func (d *DB) ListSessionRuns(ctx context.Context, userID string) ([]*upal.Run, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+upalRunColumns+` FROM upal_runs WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list upal_runs: %w", err)
	}
	defer rows.Close()
	return scanSessionRuns(rows)
}

func (d *DB) ListSessionRunsBySession(ctx context.Context, userID string, sessionID string) ([]*upal.Run, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+upalRunColumns+` FROM upal_runs WHERE session_id = $1 AND user_id = $2 ORDER BY created_at DESC`,
		sessionID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list upal_runs by session: %w", err)
	}
	defer rows.Close()
	return scanSessionRuns(rows)
}

func (d *DB) ListSessionRunsByStatus(ctx context.Context, userID string, status string) ([]*upal.Run, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+upalRunColumns+` FROM upal_runs WHERE status = $1 AND user_id = $2 ORDER BY created_at DESC`,
		status, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list upal_runs by status: %w", err)
	}
	defer rows.Close()
	return scanSessionRuns(rows)
}

func (d *DB) UpdateSessionRun(ctx context.Context, userID string, r *upal.Run) error {
	sourcesJSON, workflowsJSON, contextJSON := marshalRunConfig(r)
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE upal_runs SET name = $1, status = $2, source_count = $3, reviewed_at = $4,
		 sources = $5, workflows = $6, context = $7, schedule = $8, schedule_active = $9
		 WHERE id = $10 AND user_id = $11`,
		r.Name, string(r.Status), r.SourceCount, r.ReviewedAt,
		sourcesJSON, workflowsJSON, contextJSON, r.Schedule, r.ScheduleActive,
		r.ID, userID,
	)
	if err != nil {
		return fmt.Errorf("update upal_run: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("run %q not found", r.ID)
	}
	return nil
}

func (d *DB) DeleteSessionRun(ctx context.Context, userID string, id string) error {
	res, err := d.Pool.ExecContext(ctx, `DELETE FROM upal_runs WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete upal_run: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("run %q not found", id)
	}
	return nil
}

func (d *DB) DeleteSessionRunsBySession(ctx context.Context, userID string, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM upal_runs WHERE session_id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("delete upal_runs by session: %w", err)
	}
	return nil
}

// --- WorkflowRun (upal_workflow_runs table) ---

func (d *DB) SaveWorkflowRuns(ctx context.Context, userID string, runID string, results []upal.WorkflowRun) error {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("marshal workflow_runs: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO upal_workflow_runs (run_id, user_id, results, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (run_id) DO UPDATE SET results = $3, updated_at = NOW()`,
		runID, userID, resultsJSON,
	)
	if err != nil {
		return fmt.Errorf("upsert upal_workflow_runs: %w", err)
	}
	return nil
}

func (d *DB) GetWorkflowRunsByRun(ctx context.Context, userID string, runID string) ([]upal.WorkflowRun, error) {
	var resultsJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT results FROM upal_workflow_runs WHERE run_id = $1 AND user_id = $2`, runID, userID,
	).Scan(&resultsJSON)
	if err == sql.ErrNoRows {
		return []upal.WorkflowRun{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get upal_workflow_runs: %w", err)
	}
	var results []upal.WorkflowRun
	if err := json.Unmarshal(resultsJSON, &results); err != nil {
		return nil, fmt.Errorf("unmarshal workflow_runs: %w", err)
	}
	return results, nil
}

func (d *DB) DeleteWorkflowRunsByRun(ctx context.Context, userID string, runID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM upal_workflow_runs WHERE run_id = $1 AND user_id = $2`, runID, userID)
	if err != nil {
		return fmt.Errorf("delete upal_workflow_runs: %w", err)
	}
	return nil
}
