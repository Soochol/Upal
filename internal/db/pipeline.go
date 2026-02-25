package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// CreatePipeline inserts a new pipeline. stages is stored as JSONB.
func (d *DB) CreatePipeline(ctx context.Context, p *upal.Pipeline) error {
	stagesJSON, err := json.Marshal(p.Stages)
	if err != nil {
		return fmt.Errorf("marshal stages: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO pipelines (id, name, description, stages, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Name, p.Description, stagesJSON, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline: %w", err)
	}
	return nil
}

// GetPipeline retrieves a pipeline by ID.
func (d *DB) GetPipeline(ctx context.Context, id string) (*upal.Pipeline, error) {
	var p upal.Pipeline
	var stagesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, name, description, stages, created_at, updated_at
		 FROM pipelines WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}
	if err := json.Unmarshal(stagesJSON, &p.Stages); err != nil {
		return nil, fmt.Errorf("unmarshal stages: %w", err)
	}
	return &p, nil
}

// ListPipelines returns all pipelines ordered by updated_at descending.
func (d *DB) ListPipelines(ctx context.Context) ([]*upal.Pipeline, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, name, description, stages, created_at, updated_at
		 FROM pipelines ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	defer rows.Close()

	var result []*upal.Pipeline
	for rows.Next() {
		var p upal.Pipeline
		var stagesJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan pipeline: %w", err)
		}
		if err := json.Unmarshal(stagesJSON, &p.Stages); err != nil {
			return nil, fmt.Errorf("unmarshal stages: %w", err)
		}
		result = append(result, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pipelines: %w", err)
	}
	return result, nil
}

// UpdatePipeline updates an existing pipeline's name, description, stages, and updated_at.
func (d *DB) UpdatePipeline(ctx context.Context, p *upal.Pipeline) error {
	stagesJSON, err := json.Marshal(p.Stages)
	if err != nil {
		return fmt.Errorf("marshal stages: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE pipelines SET name = $1, description = $2, stages = $3, updated_at = $4
		 WHERE id = $5`,
		p.Name, p.Description, stagesJSON, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update pipeline: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pipeline %q not found", p.ID)
	}
	return nil
}

// DeletePipeline removes a pipeline by ID. Cascade deletes pipeline_runs.
func (d *DB) DeletePipeline(ctx context.Context, id string) error {
	res, err := d.Pool.ExecContext(ctx, `DELETE FROM pipelines WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pipeline %q not found", id)
	}
	return nil
}

// CreatePipelineRun inserts a new pipeline run.
func (d *DB) CreatePipelineRun(ctx context.Context, run *upal.PipelineRun) error {
	stageResultsJSON, err := json.Marshal(run.StageResults)
	if err != nil {
		return fmt.Errorf("marshal stage_results: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO pipeline_runs (id, pipeline_id, status, current_stage, stage_results, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		run.ID, run.PipelineID, run.Status, run.CurrentStage, stageResultsJSON, run.StartedAt, run.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline_run: %w", err)
	}
	return nil
}

// GetPipelineRun retrieves a pipeline run by ID.
func (d *DB) GetPipelineRun(ctx context.Context, id string) (*upal.PipelineRun, error) {
	var run upal.PipelineRun
	var stageResultsJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, pipeline_id, status, current_stage, stage_results, started_at, completed_at
		 FROM pipeline_runs WHERE id = $1`, id,
	).Scan(&run.ID, &run.PipelineID, &run.Status, &run.CurrentStage, &stageResultsJSON, &run.StartedAt, &run.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline run %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get pipeline_run: %w", err)
	}
	if err := json.Unmarshal(stageResultsJSON, &run.StageResults); err != nil {
		return nil, fmt.Errorf("unmarshal stage_results: %w", err)
	}
	return &run, nil
}

// ListPipelineRunsByPipeline returns all runs for a pipeline ordered by started_at descending.
func (d *DB) ListPipelineRunsByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, status, current_stage, stage_results, started_at, completed_at
		 FROM pipeline_runs WHERE pipeline_id = $1 ORDER BY started_at DESC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipeline_runs: %w", err)
	}
	defer rows.Close()

	var result []*upal.PipelineRun
	for rows.Next() {
		var run upal.PipelineRun
		var stageResultsJSON []byte
		if err := rows.Scan(&run.ID, &run.PipelineID, &run.Status, &run.CurrentStage, &stageResultsJSON, &run.StartedAt, &run.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan pipeline_run: %w", err)
		}
		if err := json.Unmarshal(stageResultsJSON, &run.StageResults); err != nil {
			return nil, fmt.Errorf("unmarshal stage_results: %w", err)
		}
		result = append(result, &run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pipeline_runs: %w", err)
	}
	return result, nil
}

// UpdatePipelineRun updates an existing pipeline run's mutable fields.
func (d *DB) UpdatePipelineRun(ctx context.Context, run *upal.PipelineRun) error {
	stageResultsJSON, err := json.Marshal(run.StageResults)
	if err != nil {
		return fmt.Errorf("marshal stage_results: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE pipeline_runs
		 SET status = $1, current_stage = $2, stage_results = $3, completed_at = $4
		 WHERE id = $5`,
		run.Status, run.CurrentStage, stageResultsJSON, run.CompletedAt, run.ID,
	)
	if err != nil {
		return fmt.Errorf("update pipeline_run: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pipeline run %q not found", run.ID)
	}
	return nil
}
