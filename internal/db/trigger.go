package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// CreateTrigger stores a new trigger.
func (d *DB) CreateTrigger(ctx context.Context, t *upal.Trigger) error {
	configJSON, _ := json.Marshal(t.Config)

	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO triggers (id, workflow_name, pipeline_id, type, config, enabled, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.ID, t.WorkflowName, t.PipelineID, string(t.Type), configJSON, t.Enabled, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert trigger: %w", err)
	}
	return nil
}

// GetTrigger retrieves a trigger by ID.
func (d *DB) GetTrigger(ctx context.Context, id string) (*upal.Trigger, error) {
	t := &upal.Trigger{}
	var triggerType string
	var configJSON []byte

	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, workflow_name, pipeline_id, type, config, enabled, created_at
		 FROM triggers WHERE id = $1`, id,
	).Scan(&t.ID, &t.WorkflowName, &t.PipelineID, &triggerType, &configJSON, &t.Enabled, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trigger not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get trigger: %w", err)
	}

	t.Type = upal.TriggerType(triggerType)
	json.Unmarshal(configJSON, &t.Config)
	return t, nil
}

// DeleteTrigger removes a trigger by ID.
func (d *DB) DeleteTrigger(ctx context.Context, id string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM triggers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete trigger: %w", err)
	}
	return nil
}

// ListTriggersByWorkflow returns triggers for a specific workflow.
func (d *DB) ListTriggersByWorkflow(ctx context.Context, workflowName string) ([]*upal.Trigger, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_name, pipeline_id, type, config, enabled, created_at
		 FROM triggers WHERE workflow_name = $1 ORDER BY created_at DESC`, workflowName,
	)
	if err != nil {
		return nil, fmt.Errorf("list triggers: %w", err)
	}
	defer rows.Close()

	return scanTriggers(rows)
}

// ListTriggersByPipeline returns triggers for a specific pipeline.
func (d *DB) ListTriggersByPipeline(ctx context.Context, pipelineID string) ([]*upal.Trigger, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_name, pipeline_id, type, config, enabled, created_at
		 FROM triggers WHERE pipeline_id = $1 ORDER BY created_at DESC`, pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipeline triggers: %w", err)
	}
	defer rows.Close()

	return scanTriggers(rows)
}

func scanTriggers(rows *sql.Rows) ([]*upal.Trigger, error) {
	var result []*upal.Trigger
	for rows.Next() {
		t := &upal.Trigger{}
		var triggerType string
		var configJSON []byte

		if err := rows.Scan(&t.ID, &t.WorkflowName, &t.PipelineID, &triggerType, &configJSON, &t.Enabled, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan trigger: %w", err)
		}

		t.Type = upal.TriggerType(triggerType)
		json.Unmarshal(configJSON, &t.Config)
		result = append(result, t)
	}
	return result, nil
}
