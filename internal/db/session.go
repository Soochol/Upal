package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// --- Session (upal_sessions table) ---

const upalSessionColumns = `id, name, description, sources, schedule, model, workflows, context, stages, status, thumbnail_svg, last_collected_at, created_at, updated_at`

// scanUpalSession scans a single row into a Session, unmarshaling JSONB fields.
func scanUpalSession(scanner interface{ Scan(...any) error }) (*upal.Session, error) {
	var s upal.Session
	var status string
	var sourcesJSON, workflowsJSON, ctxJSON, stagesJSON []byte
	if err := scanner.Scan(
		&s.ID, &s.Name, &s.Description,
		&sourcesJSON, &s.Schedule, &s.Model, &workflowsJSON, &ctxJSON, &stagesJSON,
		&status, &s.ThumbnailSVG, &s.LastCollectedAt, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	s.Status = upal.SessionStatus(status)
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
	if len(stagesJSON) > 0 {
		if err := json.Unmarshal(stagesJSON, &s.Stages); err != nil {
			return nil, fmt.Errorf("unmarshal stages: %w", err)
		}
	}
	return &s, nil
}

// scanUpalSessions scans rows into a slice of Sessions.
func scanUpalSessions(rows *sql.Rows) ([]*upal.Session, error) {
	var result []*upal.Session
	for rows.Next() {
		s, err := scanUpalSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan upal_session: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// marshalUpalSessionJSON marshals sources, workflows, context, and stages for insert/update.
func marshalUpalSessionJSON(s *upal.Session) (sourcesJSON, workflowsJSON, ctxJSON, stagesJSON []byte, err error) {
	sourcesJSON, err = json.Marshal(s.Sources)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("marshal sources: %w", err)
	}
	workflowsJSON, err = json.Marshal(s.Workflows)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("marshal workflows: %w", err)
	}
	ctxJSON, err = json.Marshal(s.Context)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("marshal context: %w", err)
	}
	stagesJSON, err = json.Marshal(s.Stages)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("marshal stages: %w", err)
	}
	return sourcesJSON, workflowsJSON, ctxJSON, stagesJSON, nil
}

func (d *DB) CreateSession(ctx context.Context, userID string, s *upal.Session) error {
	sourcesJSON, workflowsJSON, ctxJSON, stagesJSON, err := marshalUpalSessionJSON(s)
	if err != nil {
		return err
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO upal_sessions (id, user_id, name, description, sources, schedule, model, workflows, context, stages, status, thumbnail_svg, last_collected_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		s.ID, userID, s.Name, s.Description,
		sourcesJSON, s.Schedule, s.Model, workflowsJSON, ctxJSON, stagesJSON,
		string(s.Status), s.ThumbnailSVG, s.LastCollectedAt, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert upal_session: %w", err)
	}
	return nil
}

func (d *DB) GetSession(ctx context.Context, userID string, id string) (*upal.Session, error) {
	row := d.Pool.QueryRowContext(ctx,
		`SELECT `+upalSessionColumns+` FROM upal_sessions WHERE id = $1 AND user_id = $2`, id, userID,
	)
	s, err := scanUpalSession(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get upal_session: %w", err)
	}
	return s, nil
}

func (d *DB) ListSessions(ctx context.Context, userID string) ([]*upal.Session, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT `+upalSessionColumns+` FROM upal_sessions WHERE user_id = $1 ORDER BY updated_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list upal_sessions: %w", err)
	}
	defer rows.Close()
	return scanUpalSessions(rows)
}

func (d *DB) UpdateSession(ctx context.Context, userID string, s *upal.Session) error {
	sourcesJSON, workflowsJSON, ctxJSON, stagesJSON, err := marshalUpalSessionJSON(s)
	if err != nil {
		return err
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE upal_sessions SET name = $1, description = $2, sources = $3, schedule = $4, model = $5,
		 workflows = $6, context = $7, stages = $8, status = $9, thumbnail_svg = $10,
		 last_collected_at = $11, updated_at = $12
		 WHERE id = $13 AND user_id = $14`,
		s.Name, s.Description, sourcesJSON, s.Schedule, s.Model,
		workflowsJSON, ctxJSON, stagesJSON, string(s.Status), s.ThumbnailSVG,
		s.LastCollectedAt, s.UpdatedAt, s.ID, userID,
	)
	if err != nil {
		return fmt.Errorf("update upal_session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session %q not found", s.ID)
	}
	return nil
}

func (d *DB) DeleteSession(ctx context.Context, userID string, id string) error {
	res, err := d.Pool.ExecContext(ctx, `DELETE FROM upal_sessions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete upal_session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session %q not found", id)
	}
	return nil
}
