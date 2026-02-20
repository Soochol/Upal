package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// CreateRun stores a new run record.
func (d *DB) CreateRun(ctx context.Context, r *upal.RunRecord) error {
	inputsJSON, _ := json.Marshal(r.Inputs)
	outputsJSON, _ := json.Marshal(r.Outputs)
	nodeRunsJSON, _ := json.Marshal(r.NodeRuns)

	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO runs (id, workflow_name, trigger_type, trigger_ref, status, inputs, outputs, error, retry_of, retry_count, node_runs, created_at, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		r.ID, r.WorkflowName, r.TriggerType, r.TriggerRef,
		string(r.Status), inputsJSON, outputsJSON, r.Error,
		r.RetryOf, r.RetryCount, nodeRunsJSON,
		r.CreatedAt, r.StartedAt, r.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	return nil
}

// GetRun retrieves a run record by ID.
func (d *DB) GetRun(ctx context.Context, id string) (*upal.RunRecord, error) {
	r := &upal.RunRecord{}
	var status string
	var inputsJSON, outputsJSON, nodeRunsJSON []byte

	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, workflow_name, trigger_type, trigger_ref, status, inputs, outputs, error, retry_of, retry_count, node_runs, created_at, started_at, completed_at
		 FROM runs WHERE id = $1`, id,
	).Scan(&r.ID, &r.WorkflowName, &r.TriggerType, &r.TriggerRef,
		&status, &inputsJSON, &outputsJSON, &r.Error,
		&r.RetryOf, &r.RetryCount, &nodeRunsJSON,
		&r.CreatedAt, &r.StartedAt, &r.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}

	r.Status = upal.RunStatus(status)
	json.Unmarshal(inputsJSON, &r.Inputs)
	json.Unmarshal(outputsJSON, &r.Outputs)
	json.Unmarshal(nodeRunsJSON, &r.NodeRuns)
	return r, nil
}

// UpdateRun updates an existing run record.
func (d *DB) UpdateRun(ctx context.Context, r *upal.RunRecord) error {
	outputsJSON, _ := json.Marshal(r.Outputs)
	nodeRunsJSON, _ := json.Marshal(r.NodeRuns)

	_, err := d.Pool.ExecContext(ctx,
		`UPDATE runs SET status = $1, outputs = $2, error = $3, retry_count = $4, node_runs = $5, started_at = $6, completed_at = $7
		 WHERE id = $8`,
		string(r.Status), outputsJSON, r.Error, r.RetryCount, nodeRunsJSON,
		r.StartedAt, r.CompletedAt, r.ID,
	)
	if err != nil {
		return fmt.Errorf("update run: %w", err)
	}
	return nil
}

// ListRunsByWorkflow returns runs for a specific workflow with pagination.
func (d *DB) ListRunsByWorkflow(ctx context.Context, workflowName string, limit, offset int) ([]*upal.RunRecord, int, error) {
	var total int
	err := d.Pool.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM runs WHERE workflow_name = $1`, workflowName,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count runs: %w", err)
	}

	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_name, trigger_type, trigger_ref, status, inputs, outputs, error, retry_of, retry_count, node_runs, created_at, started_at, completed_at
		 FROM runs WHERE workflow_name = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		workflowName, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	return scanRuns(rows, total)
}

// ListAllRuns returns all runs with pagination.
func (d *DB) ListAllRuns(ctx context.Context, limit, offset int) ([]*upal.RunRecord, int, error) {
	var total int
	err := d.Pool.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM runs`,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count runs: %w", err)
	}

	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_name, trigger_type, trigger_ref, status, inputs, outputs, error, retry_of, retry_count, node_runs, created_at, started_at, completed_at
		 FROM runs ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	return scanRuns(rows, total)
}

func scanRuns(rows *sql.Rows, total int) ([]*upal.RunRecord, int, error) {
	var result []*upal.RunRecord
	for rows.Next() {
		r := &upal.RunRecord{}
		var status string
		var inputsJSON, outputsJSON, nodeRunsJSON []byte

		if err := rows.Scan(&r.ID, &r.WorkflowName, &r.TriggerType, &r.TriggerRef,
			&status, &inputsJSON, &outputsJSON, &r.Error,
			&r.RetryOf, &r.RetryCount, &nodeRunsJSON,
			&r.CreatedAt, &r.StartedAt, &r.CompletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan run: %w", err)
		}

		r.Status = upal.RunStatus(status)
		json.Unmarshal(inputsJSON, &r.Inputs)
		json.Unmarshal(outputsJSON, &r.Outputs)
		json.Unmarshal(nodeRunsJSON, &r.NodeRuns)
		result = append(result, r)
	}
	return result, total, nil
}
