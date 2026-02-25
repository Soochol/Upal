package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// CreateSchedule stores a new schedule.
func (d *DB) CreateSchedule(ctx context.Context, s *upal.Schedule) error {
	inputsJSON, _ := json.Marshal(s.Inputs)
	var retryParam any // SQL NULL when nil; lib/pq rejects nil []byte for JSONB
	if s.RetryPolicy != nil {
		retryParam, _ = json.Marshal(s.RetryPolicy)
	}

	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO schedules (id, workflow_name, pipeline_id, cron_expr, inputs, enabled, timezone, retry_policy, next_run_at, last_run_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		s.ID, s.WorkflowName, s.PipelineID, s.CronExpr, inputsJSON,
		s.Enabled, s.Timezone, retryParam,
		s.NextRunAt, s.LastRunAt, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}
	return nil
}

// GetSchedule retrieves a schedule by ID.
func (d *DB) GetSchedule(ctx context.Context, id string) (*upal.Schedule, error) {
	s := &upal.Schedule{}
	var inputsJSON, retryJSON []byte

	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, workflow_name, pipeline_id, cron_expr, inputs, enabled, timezone, retry_policy, next_run_at, last_run_at, created_at, updated_at
		 FROM schedules WHERE id = $1`, id,
	).Scan(&s.ID, &s.WorkflowName, &s.PipelineID, &s.CronExpr, &inputsJSON,
		&s.Enabled, &s.Timezone, &retryJSON,
		&s.NextRunAt, &s.LastRunAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	json.Unmarshal(inputsJSON, &s.Inputs)
	if len(retryJSON) > 0 {
		s.RetryPolicy = &upal.RetryPolicy{}
		json.Unmarshal(retryJSON, s.RetryPolicy)
	}
	return s, nil
}

// UpdateSchedule updates an existing schedule.
func (d *DB) UpdateSchedule(ctx context.Context, s *upal.Schedule) error {
	inputsJSON, _ := json.Marshal(s.Inputs)
	var retryParam any
	if s.RetryPolicy != nil {
		retryParam, _ = json.Marshal(s.RetryPolicy)
	}

	_, err := d.Pool.ExecContext(ctx,
		`UPDATE schedules SET workflow_name = $1, pipeline_id = $2, cron_expr = $3, inputs = $4, enabled = $5, timezone = $6, retry_policy = $7, next_run_at = $8, last_run_at = $9, updated_at = $10
		 WHERE id = $11`,
		s.WorkflowName, s.PipelineID, s.CronExpr, inputsJSON,
		s.Enabled, s.Timezone, retryParam,
		s.NextRunAt, s.LastRunAt, s.UpdatedAt, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	return nil
}

// DeleteSchedule removes a schedule by ID.
func (d *DB) DeleteSchedule(ctx context.Context, id string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

// ListSchedules returns all schedules.
func (d *DB) ListSchedules(ctx context.Context) ([]*upal.Schedule, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_name, pipeline_id, cron_expr, inputs, enabled, timezone, retry_policy, next_run_at, last_run_at, created_at, updated_at
		 FROM schedules ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	return scanSchedules(rows)
}

// ListDueSchedules returns enabled schedules whose next_run_at is at or before now.
func (d *DB) ListDueSchedules(ctx context.Context, now time.Time) ([]*upal.Schedule, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_name, pipeline_id, cron_expr, inputs, enabled, timezone, retry_policy, next_run_at, last_run_at, created_at, updated_at
		 FROM schedules WHERE enabled = true AND next_run_at <= $1`, now,
	)
	if err != nil {
		return nil, fmt.Errorf("list due schedules: %w", err)
	}
	defer rows.Close()

	return scanSchedules(rows)
}

// ListSchedulesByPipeline returns all schedules associated with a pipeline.
func (d *DB) ListSchedulesByPipeline(ctx context.Context, pipelineID string) ([]*upal.Schedule, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_name, pipeline_id, cron_expr, inputs, enabled, timezone, retry_policy, next_run_at, last_run_at, created_at, updated_at
		 FROM schedules WHERE pipeline_id = $1 ORDER BY created_at DESC`, pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list schedules by pipeline: %w", err)
	}
	defer rows.Close()

	return scanSchedules(rows)
}

func scanSchedules(rows *sql.Rows) ([]*upal.Schedule, error) {
	var result []*upal.Schedule
	for rows.Next() {
		s := &upal.Schedule{}
		var inputsJSON, retryJSON []byte

		if err := rows.Scan(&s.ID, &s.WorkflowName, &s.PipelineID, &s.CronExpr, &inputsJSON,
			&s.Enabled, &s.Timezone, &retryJSON,
			&s.NextRunAt, &s.LastRunAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}

		json.Unmarshal(inputsJSON, &s.Inputs)
		if len(retryJSON) > 0 {
			s.RetryPolicy = &upal.RetryPolicy{}
			json.Unmarshal(retryJSON, s.RetryPolicy)
		}
		result = append(result, s)
	}
	return result, nil
}
