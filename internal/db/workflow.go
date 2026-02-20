package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// WorkflowRow represents a workflow stored in the database.
type WorkflowRow struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Version    int                      `json:"version"`
	Definition upal.WorkflowDefinition  `json:"definition"`
	Visibility string                   `json:"visibility"`
	CreatedAt  time.Time                `json:"created_at"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

// CreateWorkflow stores a new workflow.
func (d *DB) CreateWorkflow(ctx context.Context, wf *upal.WorkflowDefinition) (*WorkflowRow, error) {
	defJSON, err := json.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("marshal definition: %w", err)
	}

	id := upal.GenerateID("wf")
	now := time.Now()
	row := &WorkflowRow{
		ID:         id,
		Name:       wf.Name,
		Version:    wf.Version,
		Definition: *wf,
		Visibility: "private",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO workflows (id, name, version, definition, visibility, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (name) DO UPDATE SET definition = EXCLUDED.definition, version = EXCLUDED.version, updated_at = EXCLUDED.updated_at`,
		row.ID, row.Name, row.Version, defJSON, row.Visibility, row.CreatedAt, row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert workflow: %w", err)
	}
	return row, nil
}

// GetWorkflow retrieves a workflow by name.
func (d *DB) GetWorkflow(ctx context.Context, name string) (*WorkflowRow, error) {
	var row WorkflowRow
	var defJSON []byte

	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, name, version, definition, visibility, created_at, updated_at
		 FROM workflows WHERE name = $1`, name,
	).Scan(&row.ID, &row.Name, &row.Version, &defJSON, &row.Visibility, &row.CreatedAt, &row.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("get workflow: %w", err)
	}

	if err := json.Unmarshal(defJSON, &row.Definition); err != nil {
		return nil, fmt.Errorf("unmarshal definition: %w", err)
	}
	return &row, nil
}

// ListWorkflows returns all workflows.
func (d *DB) ListWorkflows(ctx context.Context) ([]WorkflowRow, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, name, version, definition, visibility, created_at, updated_at
		 FROM workflows ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()

	var result []WorkflowRow
	for rows.Next() {
		var row WorkflowRow
		var defJSON []byte
		if err := rows.Scan(&row.ID, &row.Name, &row.Version, &defJSON, &row.Visibility, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow: %w", err)
		}
		if err := json.Unmarshal(defJSON, &row.Definition); err != nil {
			return nil, fmt.Errorf("unmarshal definition: %w", err)
		}
		result = append(result, row)
	}
	return result, nil
}

// UpdateWorkflow updates a workflow definition.
func (d *DB) UpdateWorkflow(ctx context.Context, name string, wf *upal.WorkflowDefinition) error {
	defJSON, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshal definition: %w", err)
	}

	res, err := d.Pool.ExecContext(ctx,
		`UPDATE workflows SET name = $1, definition = $2, version = $3, updated_at = NOW() WHERE name = $4`,
		wf.Name, defJSON, wf.Version, name,
	)
	if err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workflow not found: %s", name)
	}
	return nil
}

// DeleteWorkflow removes a workflow by name.
func (d *DB) DeleteWorkflow(ctx context.Context, name string) error {
	res, err := d.Pool.ExecContext(ctx, `DELETE FROM workflows WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("delete workflow: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workflow not found: %s", name)
	}
	return nil
}
