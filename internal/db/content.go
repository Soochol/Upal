package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// --- ContentSession ---

func (d *DB) CreateContentSession(ctx context.Context, s *upal.ContentSession) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO content_sessions (id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		s.ID, s.PipelineID, s.Name, string(s.Status), s.TriggerType, s.SourceCount, s.IsTemplate, s.ParentSessionID, s.CreatedAt, s.ReviewedAt, s.ArchivedAt,
	)
	if err != nil {
		return fmt.Errorf("insert content_session: %w", err)
	}
	return nil
}

func (d *DB) GetContentSession(ctx context.Context, id string) (*upal.ContentSession, error) {
	var s upal.ContentSession
	var status string
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.PipelineID, &s.Name, &status, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("content session %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get content_session: %w", err)
	}
	s.Status = upal.ContentSessionStatus(status)
	return &s, nil
}

func (d *DB) ListContentSessions(ctx context.Context) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE is_template = false AND archived_at IS NULL ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &status, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE pipeline_id = $1 AND is_template = false AND archived_at IS NULL ORDER BY created_at DESC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions by pipeline: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &status, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) UpdateContentSession(ctx context.Context, s *upal.ContentSession) error {
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE content_sessions SET name = $1, status = $2, source_count = $3, is_template = $4, parent_session_id = $5, reviewed_at = $6, archived_at = $7 WHERE id = $8`,
		s.Name, string(s.Status), s.SourceCount, s.IsTemplate, s.ParentSessionID, s.ReviewedAt, s.ArchivedAt, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update content_session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("content session %q not found", s.ID)
	}
	return nil
}

func (d *DB) ListContentSessionsByStatus(ctx context.Context, status string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE status = $1 AND is_template = false AND archived_at IS NULL ORDER BY created_at DESC`,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions by status: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var st string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &st, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(st)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListAllContentSessionsByStatus(ctx context.Context, status string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE status = $1 AND is_template = false ORDER BY created_at DESC`,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("list all content_sessions by status: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var st string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &st, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(st)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListContentSessionsByPipelineAndStatus(ctx context.Context, pipelineID, status string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE pipeline_id = $1 AND status = $2 AND is_template = false AND archived_at IS NULL ORDER BY created_at DESC`,
		pipelineID, status,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions by pipeline+status: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var st string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &st, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(st)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListArchivedContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE pipeline_id = $1 AND archived_at IS NOT NULL ORDER BY archived_at DESC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list archived content_sessions: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &status, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListTemplateContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE pipeline_id = $1 AND is_template = true AND archived_at IS NULL ORDER BY created_at DESC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list template content_sessions: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &status, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) DeleteContentSession(ctx context.Context, id string) error {
	res, err := d.Pool.ExecContext(ctx, `DELETE FROM content_sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete content_session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("content session %q not found", id)
	}
	return nil
}

func (d *DB) DeletePublishedContentBySession(ctx context.Context, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM published_content WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("delete published_content by session: %w", err)
	}
	return nil
}

// --- SourceFetch ---

func (d *DB) CreateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	itemsJSON, err := json.Marshal(sf.RawItems)
	if err != nil {
		return fmt.Errorf("marshal raw_items: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO source_fetches (id, session_id, tool_name, source_type, label, item_count, raw_items, error, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		sf.ID, sf.SessionID, sf.ToolName, sf.SourceType, sf.Label, sf.Count, itemsJSON, sf.Error, sf.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("insert source_fetch: %w", err)
	}
	return nil
}

func (d *DB) ListSourceFetchesBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, session_id, tool_name, source_type, COALESCE(label, ''), COALESCE(item_count, 0), raw_items, error, fetched_at
		 FROM source_fetches WHERE session_id = $1 ORDER BY fetched_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list source_fetches: %w", err)
	}
	defer rows.Close()
	var result []*upal.SourceFetch
	for rows.Next() {
		var sf upal.SourceFetch
		var itemsJSON []byte
		if err := rows.Scan(&sf.ID, &sf.SessionID, &sf.ToolName, &sf.SourceType, &sf.Label, &sf.Count, &itemsJSON, &sf.Error, &sf.FetchedAt); err != nil {
			return nil, fmt.Errorf("scan source_fetch: %w", err)
		}
		if err := json.Unmarshal(itemsJSON, &sf.RawItems); err != nil {
			return nil, fmt.Errorf("unmarshal raw_items: %w", err)
		}
		// Recover count from items if DB column was not populated.
		if sf.Count == 0 && len(sf.RawItems) > 0 {
			sf.Count = len(sf.RawItems)
		}
		result = append(result, &sf)
	}
	return result, rows.Err()
}

// --- LLMAnalysis ---

func (d *DB) CreateLLMAnalysis(ctx context.Context, a *upal.LLMAnalysis) error {
	insightsJSON, err := json.Marshal(a.Insights)
	if err != nil {
		return fmt.Errorf("marshal insights: %w", err)
	}
	anglesJSON, err := json.Marshal(a.SuggestedAngles)
	if err != nil {
		return fmt.Errorf("marshal suggested_angles: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO llm_analyses (id, session_id, raw_item_count, filtered_count, summary, insights, suggested_angles, overall_score, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.SessionID, a.RawItemCount, a.FilteredCount, a.Summary,
		insightsJSON, anglesJSON, a.OverallScore, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert llm_analysis: %w", err)
	}
	return nil
}

func (d *DB) UpdateLLMAnalysis(ctx context.Context, a *upal.LLMAnalysis) error {
	insightsJSON, err := json.Marshal(a.Insights)
	if err != nil {
		return fmt.Errorf("marshal insights: %w", err)
	}
	anglesJSON, err := json.Marshal(a.SuggestedAngles)
	if err != nil {
		return fmt.Errorf("marshal suggested_angles: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE llm_analyses SET summary = $1, insights = $2, suggested_angles = $3, overall_score = $4 WHERE id = $5`,
		a.Summary, insightsJSON, anglesJSON, a.OverallScore, a.ID,
	)
	if err != nil {
		return fmt.Errorf("update llm_analysis: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("llm analysis %q not found", a.ID)
	}
	return nil
}

func (d *DB) GetLLMAnalysisBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	var a upal.LLMAnalysis
	var insightsJSON, anglesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, session_id, raw_item_count, filtered_count, summary, insights, suggested_angles, overall_score, created_at
		 FROM llm_analyses WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1`,
		sessionID,
	).Scan(&a.ID, &a.SessionID, &a.RawItemCount, &a.FilteredCount, &a.Summary,
		&insightsJSON, &anglesJSON, &a.OverallScore, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("llm analysis for session %q not found", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("get llm_analysis: %w", err)
	}
	if err := json.Unmarshal(insightsJSON, &a.Insights); err != nil {
		return nil, fmt.Errorf("unmarshal insights: %w", err)
	}
	if err := json.Unmarshal(anglesJSON, &a.SuggestedAngles); err != nil {
		return nil, fmt.Errorf("unmarshal suggested_angles: %w", err)
	}
	return &a, nil
}

// --- PublishedContent ---

func (d *DB) CreatePublishedContent(ctx context.Context, pc *upal.PublishedContent) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO published_content (id, workflow_run_id, session_id, channel, external_url, title, published_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		pc.ID, pc.WorkflowRunID, pc.SessionID, pc.Channel, pc.ExternalURL, pc.Title, pc.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert published_content: %w", err)
	}
	return nil
}

func (d *DB) ListPublishedContent(ctx context.Context) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content ORDER BY published_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func (d *DB) ListPublishedContentBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content WHERE session_id = $1 ORDER BY published_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content by session: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func (d *DB) ListPublishedContentByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content WHERE channel = $1 ORDER BY published_at DESC`,
		channel,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content by channel: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func scanPublishedContent(rows *sql.Rows) ([]*upal.PublishedContent, error) {
	var result []*upal.PublishedContent
	for rows.Next() {
		var pc upal.PublishedContent
		if err := rows.Scan(&pc.ID, &pc.WorkflowRunID, &pc.SessionID, &pc.Channel, &pc.ExternalURL, &pc.Title, &pc.PublishedAt); err != nil {
			return nil, fmt.Errorf("scan published_content: %w", err)
		}
		result = append(result, &pc)
	}
	return result, rows.Err()
}

// --- SurgeEvent ---

func (d *DB) CreateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error {
	sourcesJSON, err := json.Marshal(se.Sources)
	if err != nil {
		return fmt.Errorf("marshal sources: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO surge_events (id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		se.ID, se.Keyword, se.PipelineID, se.Multiplier, sourcesJSON, se.Dismissed, se.SessionID, se.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert surge_event: %w", err)
	}
	return nil
}

func (d *DB) ListSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list surge_events: %w", err)
	}
	defer rows.Close()
	return scanSurgeEvents(rows)
}

func (d *DB) ListActiveSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events WHERE dismissed = false ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active surge_events: %w", err)
	}
	defer rows.Close()
	return scanSurgeEvents(rows)
}

func (d *DB) UpdateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error {
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE surge_events SET dismissed = $1, session_id = $2 WHERE id = $3`,
		se.Dismissed, se.SessionID, se.ID,
	)
	if err != nil {
		return fmt.Errorf("update surge_event: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("surge event %q not found", se.ID)
	}
	return nil
}

func (d *DB) GetSurgeEvent(ctx context.Context, id string) (*upal.SurgeEvent, error) {
	var se upal.SurgeEvent
	var sourcesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events WHERE id = $1`, id,
	).Scan(&se.ID, &se.Keyword, &se.PipelineID, &se.Multiplier, &sourcesJSON, &se.Dismissed, &se.SessionID, &se.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("surge event %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get surge_event: %w", err)
	}
	if err := json.Unmarshal(sourcesJSON, &se.Sources); err != nil {
		return nil, fmt.Errorf("unmarshal sources: %w", err)
	}
	return &se, nil
}

// --- WorkflowResults ---

func (d *DB) SaveWorkflowResults(ctx context.Context, sessionID string, results []upal.WorkflowResult) error {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("marshal workflow_results: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO workflow_results (session_id, results, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (session_id) DO UPDATE SET results = $2, updated_at = NOW()`,
		sessionID, resultsJSON,
	)
	if err != nil {
		return fmt.Errorf("upsert workflow_results: %w", err)
	}
	return nil
}

func (d *DB) GetWorkflowResultsBySession(ctx context.Context, sessionID string) ([]upal.WorkflowResult, error) {
	var resultsJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT results FROM workflow_results WHERE session_id = $1`, sessionID,
	).Scan(&resultsJSON)
	if err == sql.ErrNoRows {
		return []upal.WorkflowResult{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get workflow_results: %w", err)
	}
	var results []upal.WorkflowResult
	if err := json.Unmarshal(resultsJSON, &results); err != nil {
		return nil, fmt.Errorf("unmarshal workflow_results: %w", err)
	}
	return results, nil
}

func (d *DB) DeleteWorkflowResultsBySession(ctx context.Context, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM workflow_results WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("delete workflow_results: %w", err)
	}
	return nil
}

func scanSurgeEvents(rows *sql.Rows) ([]*upal.SurgeEvent, error) {
	var result []*upal.SurgeEvent
	for rows.Next() {
		var se upal.SurgeEvent
		var sourcesJSON []byte
		if err := rows.Scan(&se.ID, &se.Keyword, &se.PipelineID, &se.Multiplier, &sourcesJSON, &se.Dismissed, &se.SessionID, &se.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan surge_event: %w", err)
		}
		if err := json.Unmarshal(sourcesJSON, &se.Sources); err != nil {
			return nil, fmt.Errorf("unmarshal sources: %w", err)
		}
		result = append(result, &se)
	}
	return result, rows.Err()
}
