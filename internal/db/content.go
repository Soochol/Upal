package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// --- ContentSession ---

const contentSessionColumns = `id, pipeline_id, name, status, trigger_type, source_count,
	sources, schedule, model, workflows, context,
	is_template, parent_session_id, created_at, reviewed_at`

// scanContentSession scans a single row into a ContentSession, unmarshaling JSONB fields.
func scanContentSession(scanner interface{ Scan(...any) error }) (*upal.ContentSession, error) {
	var s upal.ContentSession
	var status string
	var sourcesJSON, workflowsJSON, ctxJSON []byte
	if err := scanner.Scan(
		&s.ID, &s.PipelineID, &s.Name, &status, &s.TriggerType, &s.SourceCount,
		&sourcesJSON, &s.Schedule, &s.Model, &workflowsJSON, &ctxJSON,
		&s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt,
	); err != nil {
		return nil, err
	}
	s.Status = upal.ContentSessionStatus(status)
	if len(sourcesJSON) > 0 {
		if err := json.Unmarshal(sourcesJSON, &s.Sources); err != nil {
			return nil, fmt.Errorf("unmarshal sources: %w", err)
		}
	}
	if len(workflowsJSON) > 0 {
		if err := json.Unmarshal(workflowsJSON, &s.Workflows); err != nil {
			return nil, fmt.Errorf("unmarshal workflows: %w", err)
		}
	}
	if len(ctxJSON) > 0 {
		if err := json.Unmarshal(ctxJSON, &s.Context); err != nil {
			return nil, fmt.Errorf("unmarshal context: %w", err)
		}
	}
	return &s, nil
}

// scanContentSessions scans rows into a slice of ContentSessions.
func scanContentSessions(rows *sql.Rows) ([]*upal.ContentSession, error) {
	var result []*upal.ContentSession
	for rows.Next() {
		s, err := scanContentSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// marshalSessionJSON marshals sources, workflows, and context for insert/update.
func marshalSessionJSON(s *upal.ContentSession) (sourcesJSON, workflowsJSON, ctxJSON []byte, err error) {
	sourcesJSON, err = json.Marshal(s.Sources)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal sources: %w", err)
	}
	workflowsJSON, err = json.Marshal(s.Workflows)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal workflows: %w", err)
	}
	ctxJSON, err = json.Marshal(s.Context)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal context: %w", err)
	}
	return sourcesJSON, workflowsJSON, ctxJSON, nil
}

func (d *DB) CreateContentSession(ctx context.Context, userID string, s *upal.ContentSession) error {
	sourcesJSON, workflowsJSON, ctxJSON, err := marshalSessionJSON(s)
	if err != nil {
		return err
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO content_sessions (id, user_id, pipeline_id, name, status, trigger_type, source_count,
		 sources, schedule, model, workflows, context,
		 is_template, parent_session_id, created_at, reviewed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		s.ID, userID, s.PipelineID, s.Name, string(s.Status), s.TriggerType, s.SourceCount,
		sourcesJSON, s.Schedule, s.Model, workflowsJSON, ctxJSON,
		s.IsTemplate, s.ParentSessionID, s.CreatedAt, s.ReviewedAt,
	)
	if err != nil {
		return fmt.Errorf("insert content_session: %w", err)
	}
	return nil
}

func (d *DB) GetContentSession(ctx context.Context, userID string, id string) (*upal.ContentSession, error) {
	row := d.Pool.QueryRowContext(ctx,
		`SELECT `+contentSessionColumns+` FROM content_sessions WHERE id = $1 AND user_id = $2`, id, userID,
	)
	s, err := scanContentSession(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("content session %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get content_session: %w", err)
	}
	return s, nil
}

func (d *DB) ListContentSessions(ctx context.Context, userID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+contentSessionColumns+` FROM content_sessions WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions: %w", err)
	}
	defer rows.Close()
	return scanContentSessions(rows)
}

func (d *DB) ListContentSessionsByPipeline(ctx context.Context, userID string, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+contentSessionColumns+` FROM content_sessions WHERE pipeline_id = $1 AND user_id = $2 ORDER BY created_at DESC`,
		pipelineID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions by pipeline: %w", err)
	}
	defer rows.Close()
	return scanContentSessions(rows)
}

func (d *DB) UpdateContentSession(ctx context.Context, userID string, s *upal.ContentSession) error {
	sourcesJSON, workflowsJSON, ctxJSON, err := marshalSessionJSON(s)
	if err != nil {
		return err
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE content_sessions SET name = $1, status = $2, source_count = $3,
		 sources = $4, schedule = $5, model = $6, workflows = $7, context = $8,
		 is_template = $9, parent_session_id = $10, reviewed_at = $11
		 WHERE id = $12 AND user_id = $13`,
		s.Name, string(s.Status), s.SourceCount,
		sourcesJSON, s.Schedule, s.Model, workflowsJSON, ctxJSON,
		s.IsTemplate, s.ParentSessionID, s.ReviewedAt, s.ID, userID,
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

func (d *DB) ListContentSessionsByStatus(ctx context.Context, userID string, status string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+contentSessionColumns+` FROM content_sessions WHERE status = $1 AND user_id = $2 ORDER BY created_at DESC`,
		status, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions by status: %w", err)
	}
	defer rows.Close()
	return scanContentSessions(rows)
}

func (d *DB) ListAllContentSessionsByStatus(ctx context.Context, userID string, status string) ([]*upal.ContentSession, error) {
	return d.ListContentSessionsByStatus(ctx, userID, status)
}

func (d *DB) ListContentSessionsByPipelineAndStatus(ctx context.Context, userID string, pipelineID, status string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+contentSessionColumns+` FROM content_sessions WHERE pipeline_id = $1 AND status = $2 AND user_id = $3 ORDER BY created_at DESC`,
		pipelineID, status, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions by pipeline+status: %w", err)
	}
	defer rows.Close()
	return scanContentSessions(rows)
}

func (d *DB) ListTemplateContentSessionsByPipeline(ctx context.Context, userID string, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+contentSessionColumns+` FROM content_sessions WHERE pipeline_id = $1 AND is_template = true AND user_id = $2 ORDER BY created_at DESC`,
		pipelineID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list template content_sessions: %w", err)
	}
	defer rows.Close()
	return scanContentSessions(rows)
}

func (d *DB) DeleteContentSession(ctx context.Context, userID string, id string) error {
	res, err := d.Pool.ExecContext(ctx, `DELETE FROM content_sessions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete content_session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("content session %q not found", id)
	}
	return nil
}

func (d *DB) DeletePublishedContentBySession(ctx context.Context, userID string, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM published_content WHERE session_id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("delete published_content by session: %w", err)
	}
	return nil
}

// --- SourceFetch ---

func (d *DB) CreateSourceFetch(ctx context.Context, userID string, sf *upal.SourceFetch) error {
	itemsJSON, err := json.Marshal(sf.RawItems)
	if err != nil {
		return fmt.Errorf("marshal raw_items: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO source_fetches (id, user_id, session_id, tool_name, source_type, label, item_count, raw_items, error, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		sf.ID, userID, sf.SessionID, sf.ToolName, sf.SourceType, sf.Label, sf.Count, itemsJSON, sf.Error, sf.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("insert source_fetch: %w", err)
	}
	return nil
}

func (d *DB) ListSourceFetchesBySession(ctx context.Context, userID string, sessionID string) ([]*upal.SourceFetch, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, session_id, tool_name, source_type, COALESCE(label, ''), COALESCE(item_count, 0), raw_items, error, fetched_at
		 FROM source_fetches WHERE session_id = $1 AND user_id = $2 ORDER BY fetched_at ASC`,
		sessionID, userID,
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

func (d *DB) DeleteSourceFetchesBySession(ctx context.Context, userID string, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM source_fetches WHERE session_id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("delete source_fetches by session: %w", err)
	}
	return nil
}

// --- LLMAnalysis ---

func (d *DB) CreateLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error {
	insightsJSON, err := json.Marshal(a.Insights)
	if err != nil {
		return fmt.Errorf("marshal insights: %w", err)
	}
	anglesJSON, err := json.Marshal(a.SuggestedAngles)
	if err != nil {
		return fmt.Errorf("marshal suggested_angles: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO llm_analyses (id, user_id, session_id, raw_item_count, filtered_count, summary, insights, suggested_angles, overall_score, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		a.ID, userID, a.SessionID, a.RawItemCount, a.FilteredCount, a.Summary,
		insightsJSON, anglesJSON, a.OverallScore, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert llm_analysis: %w", err)
	}
	return nil
}

func (d *DB) UpdateLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error {
	insightsJSON, err := json.Marshal(a.Insights)
	if err != nil {
		return fmt.Errorf("marshal insights: %w", err)
	}
	anglesJSON, err := json.Marshal(a.SuggestedAngles)
	if err != nil {
		return fmt.Errorf("marshal suggested_angles: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE llm_analyses SET summary = $1, insights = $2, suggested_angles = $3, overall_score = $4 WHERE id = $5 AND user_id = $6`,
		a.Summary, insightsJSON, anglesJSON, a.OverallScore, a.ID, userID,
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

func (d *DB) GetLLMAnalysisBySession(ctx context.Context, userID string, sessionID string) (*upal.LLMAnalysis, error) {
	var a upal.LLMAnalysis
	var insightsJSON, anglesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, session_id, raw_item_count, filtered_count, summary, insights, suggested_angles, overall_score, created_at
		 FROM llm_analyses WHERE session_id = $1 AND user_id = $2 ORDER BY created_at DESC LIMIT 1`,
		sessionID, userID,
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

func (d *DB) DeleteLLMAnalysesBySession(ctx context.Context, userID string, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM llm_analyses WHERE session_id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("delete llm_analyses by session: %w", err)
	}
	return nil
}

// --- PublishedContent ---

func (d *DB) CreatePublishedContent(ctx context.Context, userID string, pc *upal.PublishedContent) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO published_content (id, user_id, workflow_run_id, session_id, channel, external_url, title, published_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		pc.ID, userID, pc.WorkflowRunID, pc.SessionID, pc.Channel, pc.ExternalURL, pc.Title, pc.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert published_content: %w", err)
	}
	return nil
}

func (d *DB) ListPublishedContent(ctx context.Context, userID string) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content WHERE user_id = $1 ORDER BY published_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func (d *DB) ListPublishedContentBySession(ctx context.Context, userID string, sessionID string) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content WHERE session_id = $1 AND user_id = $2 ORDER BY published_at DESC`,
		sessionID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content by session: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func (d *DB) ListPublishedContentByChannel(ctx context.Context, userID string, channel string) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content WHERE channel = $1 AND user_id = $2 ORDER BY published_at DESC`,
		channel, userID,
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

func (d *DB) CreateSurgeEvent(ctx context.Context, userID string, se *upal.SurgeEvent) error {
	sourcesJSON, err := json.Marshal(se.Sources)
	if err != nil {
		return fmt.Errorf("marshal sources: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO surge_events (id, user_id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		se.ID, userID, se.Keyword, se.PipelineID, se.Multiplier, sourcesJSON, se.Dismissed, se.SessionID, se.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert surge_event: %w", err)
	}
	return nil
}

func (d *DB) ListSurgeEvents(ctx context.Context, userID string) ([]*upal.SurgeEvent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list surge_events: %w", err)
	}
	defer rows.Close()
	return scanSurgeEvents(rows)
}

func (d *DB) ListActiveSurgeEvents(ctx context.Context, userID string) ([]*upal.SurgeEvent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events WHERE dismissed = false AND user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active surge_events: %w", err)
	}
	defer rows.Close()
	return scanSurgeEvents(rows)
}

func (d *DB) UpdateSurgeEvent(ctx context.Context, userID string, se *upal.SurgeEvent) error {
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE surge_events SET dismissed = $1, session_id = $2 WHERE id = $3 AND user_id = $4`,
		se.Dismissed, se.SessionID, se.ID, userID,
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

func (d *DB) GetSurgeEvent(ctx context.Context, userID string, id string) (*upal.SurgeEvent, error) {
	var se upal.SurgeEvent
	var sourcesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events WHERE id = $1 AND user_id = $2`, id, userID,
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

func (d *DB) SaveWorkflowResults(ctx context.Context, userID string, sessionID string, results []upal.WorkflowResult) error {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("marshal workflow_results: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO workflow_results (session_id, user_id, results, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (session_id) DO UPDATE SET results = $3, updated_at = NOW()`,
		sessionID, userID, resultsJSON,
	)
	if err != nil {
		return fmt.Errorf("upsert workflow_results: %w", err)
	}
	return nil
}

func (d *DB) GetWorkflowResultsBySession(ctx context.Context, userID string, sessionID string) ([]upal.WorkflowResult, error) {
	var resultsJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT results FROM workflow_results WHERE session_id = $1 AND user_id = $2`, sessionID, userID,
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

func (d *DB) DeleteWorkflowResultsBySession(ctx context.Context, userID string, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM workflow_results WHERE session_id = $1 AND user_id = $2`, sessionID, userID)
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

// --- V2 Source Fetches (upal_source_fetches, keyed by run_id) ---

func (d *DB) CreateRunSourceFetch(ctx context.Context, userID string, sf *upal.SourceFetch) error {
	itemsJSON, err := json.Marshal(sf.RawItems)
	if err != nil {
		return fmt.Errorf("marshal raw_items: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO upal_source_fetches (id, user_id, run_id, tool_name, source_type, label, item_count, raw_items, error, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		sf.ID, userID, sf.SessionID, sf.ToolName, sf.SourceType, sf.Label, sf.Count, itemsJSON, sf.Error, sf.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("insert upal_source_fetch: %w", err)
	}
	return nil
}

func (d *DB) ListRunSourceFetches(ctx context.Context, userID string, runID string) ([]*upal.SourceFetch, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, run_id, tool_name, source_type, COALESCE(label, ''), COALESCE(item_count, 0), raw_items, error, fetched_at
		 FROM upal_source_fetches WHERE run_id = $1 AND user_id = $2 ORDER BY fetched_at ASC`,
		runID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list upal_source_fetches: %w", err)
	}
	defer rows.Close()
	var result []*upal.SourceFetch
	for rows.Next() {
		var sf upal.SourceFetch
		var itemsJSON []byte
		if err := rows.Scan(&sf.ID, &sf.SessionID, &sf.ToolName, &sf.SourceType, &sf.Label, &sf.Count, &itemsJSON, &sf.Error, &sf.FetchedAt); err != nil {
			return nil, fmt.Errorf("scan upal_source_fetch: %w", err)
		}
		if err := json.Unmarshal(itemsJSON, &sf.RawItems); err != nil {
			return nil, fmt.Errorf("unmarshal raw_items: %w", err)
		}
		if sf.Count == 0 && len(sf.RawItems) > 0 {
			sf.Count = len(sf.RawItems)
		}
		result = append(result, &sf)
	}
	return result, rows.Err()
}

func (d *DB) DeleteRunSourceFetches(ctx context.Context, userID string, runID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM upal_source_fetches WHERE run_id = $1 AND user_id = $2`, runID, userID)
	if err != nil {
		return fmt.Errorf("delete upal_source_fetches: %w", err)
	}
	return nil
}

// --- V2 LLM Analyses (upal_llm_analyses, keyed by run_id) ---

func (d *DB) CreateRunLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error {
	highlightsJSON, err := json.Marshal(a.SourceHighlights)
	if err != nil {
		return fmt.Errorf("marshal source_highlights: %w", err)
	}
	insightsJSON, err := json.Marshal(a.Insights)
	if err != nil {
		return fmt.Errorf("marshal insights: %w", err)
	}
	anglesJSON, err := json.Marshal(a.SuggestedAngles)
	if err != nil {
		return fmt.Errorf("marshal suggested_angles: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO upal_llm_analyses (id, user_id, run_id, raw_item_count, filtered_count, summary, source_highlights, insights, suggested_angles, overall_score, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		a.ID, userID, a.SessionID, a.RawItemCount, a.FilteredCount, a.Summary,
		highlightsJSON, insightsJSON, anglesJSON, a.OverallScore, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert upal_llm_analysis: %w", err)
	}
	return nil
}

func (d *DB) GetRunLLMAnalysis(ctx context.Context, userID string, runID string) (*upal.LLMAnalysis, error) {
	var a upal.LLMAnalysis
	var highlightsJSON, insightsJSON, anglesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, run_id, raw_item_count, filtered_count, summary, source_highlights, insights, suggested_angles, overall_score, created_at
		 FROM upal_llm_analyses WHERE run_id = $1 AND user_id = $2`,
		runID, userID,
	).Scan(&a.ID, &a.SessionID, &a.RawItemCount, &a.FilteredCount, &a.Summary,
		&highlightsJSON, &insightsJSON, &anglesJSON, &a.OverallScore, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("upal_llm_analysis for run %q not found", runID)
	}
	if err != nil {
		return nil, fmt.Errorf("get upal_llm_analysis: %w", err)
	}
	if err := json.Unmarshal(highlightsJSON, &a.SourceHighlights); err != nil {
		return nil, fmt.Errorf("unmarshal source_highlights: %w", err)
	}
	if err := json.Unmarshal(insightsJSON, &a.Insights); err != nil {
		return nil, fmt.Errorf("unmarshal insights: %w", err)
	}
	if err := json.Unmarshal(anglesJSON, &a.SuggestedAngles); err != nil {
		return nil, fmt.Errorf("unmarshal suggested_angles: %w", err)
	}
	return &a, nil
}

func (d *DB) UpdateRunLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error {
	highlightsJSON, err := json.Marshal(a.SourceHighlights)
	if err != nil {
		return fmt.Errorf("marshal source_highlights: %w", err)
	}
	insightsJSON, err := json.Marshal(a.Insights)
	if err != nil {
		return fmt.Errorf("marshal insights: %w", err)
	}
	anglesJSON, err := json.Marshal(a.SuggestedAngles)
	if err != nil {
		return fmt.Errorf("marshal suggested_angles: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE upal_llm_analyses SET raw_item_count=$1, filtered_count=$2, summary=$3, source_highlights=$4, insights=$5, suggested_angles=$6, overall_score=$7
		 WHERE run_id = $8 AND user_id = $9`,
		a.RawItemCount, a.FilteredCount, a.Summary, highlightsJSON, insightsJSON, anglesJSON, a.OverallScore,
		a.SessionID, userID,
	)
	if err != nil {
		return fmt.Errorf("update upal_llm_analysis: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("upal_llm_analysis for run %q not found", a.SessionID)
	}
	return nil
}

func (d *DB) DeleteRunLLMAnalyses(ctx context.Context, userID string, runID string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM upal_llm_analyses WHERE run_id = $1 AND user_id = $2`, runID, userID)
	if err != nil {
		return fmt.Errorf("delete upal_llm_analyses: %w", err)
	}
	return nil
}
