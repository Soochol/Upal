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
		`INSERT INTO content_sessions (id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.PipelineID, string(s.Status), s.TriggerType, s.SourceCount, s.CreatedAt, s.ReviewedAt,
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
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.PipelineID, &status, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt)
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
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &status, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions WHERE pipeline_id = $1 ORDER BY created_at DESC`,
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
		if err := rows.Scan(&s.ID, &s.PipelineID, &status, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) UpdateContentSession(ctx context.Context, s *upal.ContentSession) error {
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE content_sessions SET status = $1, source_count = $2, reviewed_at = $3 WHERE id = $4`,
		string(s.Status), s.SourceCount, s.ReviewedAt, s.ID,
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
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions WHERE status = $1 ORDER BY created_at DESC`,
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
		if err := rows.Scan(&s.ID, &s.PipelineID, &st, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(st)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListContentSessionsByPipelineAndStatus(ctx context.Context, pipelineID, status string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions WHERE pipeline_id = $1 AND status = $2 ORDER BY created_at DESC`,
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
		if err := rows.Scan(&s.ID, &s.PipelineID, &st, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(st)
		result = append(result, &s)
	}
	return result, rows.Err()
}

// --- SourceFetch ---

func (d *DB) CreateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	itemsJSON, err := json.Marshal(sf.RawItems)
	if err != nil {
		return fmt.Errorf("marshal raw_items: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO source_fetches (id, session_id, tool_name, source_type, raw_items, error, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sf.ID, sf.SessionID, sf.ToolName, sf.SourceType, itemsJSON, sf.Error, sf.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("insert source_fetch: %w", err)
	}
	return nil
}

func (d *DB) ListSourceFetchesBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, session_id, tool_name, source_type, raw_items, error, fetched_at
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
		if err := rows.Scan(&sf.ID, &sf.SessionID, &sf.ToolName, &sf.SourceType, &itemsJSON, &sf.Error, &sf.FetchedAt); err != nil {
			return nil, fmt.Errorf("scan source_fetch: %w", err)
		}
		if err := json.Unmarshal(itemsJSON, &sf.RawItems); err != nil {
			return nil, fmt.Errorf("unmarshal raw_items: %w", err)
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
