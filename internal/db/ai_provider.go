package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

func (d *DB) CreateAIProvider(ctx context.Context, userID string, p *upal.AIProvider) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO ai_providers (id, user_id, name, category, type, model, api_key, is_default) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, userID, p.Name, string(p.Category), p.Type, p.Model, p.APIKey, p.IsDefault,
	)
	if err != nil {
		return fmt.Errorf("insert ai provider: %w", err)
	}
	return nil
}

func (d *DB) GetAIProvider(ctx context.Context, userID string, id string) (*upal.AIProvider, error) {
	p := &upal.AIProvider{}
	var category string
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, name, category, type, model, api_key, is_default FROM ai_providers WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&p.ID, &p.Name, &category, &p.Type, &p.Model, &p.APIKey, &p.IsDefault)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ai provider not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get ai provider: %w", err)
	}
	p.Category = upal.AIProviderCategory(category)
	return p, nil
}

func (d *DB) ListAIProviders(ctx context.Context, userID string) ([]*upal.AIProvider, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, name, category, type, model, api_key, is_default FROM ai_providers WHERE user_id = $1 ORDER BY category, name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai providers: %w", err)
	}
	defer rows.Close()
	var result []*upal.AIProvider
	for rows.Next() {
		p := &upal.AIProvider{}
		var category string
		if err := rows.Scan(&p.ID, &p.Name, &category, &p.Type, &p.Model, &p.APIKey, &p.IsDefault); err != nil {
			return nil, fmt.Errorf("scan ai provider: %w", err)
		}
		p.Category = upal.AIProviderCategory(category)
		result = append(result, p)
	}
	return result, nil
}

func (d *DB) UpdateAIProvider(ctx context.Context, userID string, p *upal.AIProvider) error {
	_, err := d.Pool.ExecContext(ctx,
		`UPDATE ai_providers SET name=$1, category=$2, type=$3, model=$4, api_key=$5, is_default=$6 WHERE id=$7 AND user_id=$8`,
		p.Name, string(p.Category), p.Type, p.Model, p.APIKey, p.IsDefault, p.ID, userID,
	)
	if err != nil {
		return fmt.Errorf("update ai provider: %w", err)
	}
	return nil
}

func (d *DB) DeleteAIProvider(ctx context.Context, userID string, id string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM ai_providers WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete ai provider: %w", err)
	}
	return nil
}

func (d *DB) ClearAIProviderDefault(ctx context.Context, userID string, category string) error {
	_, err := d.Pool.ExecContext(ctx,
		`UPDATE ai_providers SET is_default = FALSE WHERE category = $1 AND user_id = $2`, category, userID,
	)
	if err != nil {
		return fmt.Errorf("clear ai provider default: %w", err)
	}
	return nil
}
