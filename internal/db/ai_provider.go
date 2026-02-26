package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

func (d *DB) CreateAIProvider(ctx context.Context, p *upal.AIProvider) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO ai_providers (id, name, category, type, api_key, is_default) VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Name, string(p.Category), p.Type, p.APIKey, p.IsDefault,
	)
	if err != nil {
		return fmt.Errorf("insert ai provider: %w", err)
	}
	return nil
}

func (d *DB) GetAIProvider(ctx context.Context, id string) (*upal.AIProvider, error) {
	p := &upal.AIProvider{}
	var category string
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, name, category, type, api_key, is_default FROM ai_providers WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &category, &p.Type, &p.APIKey, &p.IsDefault)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ai provider not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get ai provider: %w", err)
	}
	p.Category = upal.AIProviderCategory(category)
	return p, nil
}

func (d *DB) ListAIProviders(ctx context.Context) ([]*upal.AIProvider, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, name, category, type, api_key, is_default FROM ai_providers ORDER BY category, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai providers: %w", err)
	}
	defer rows.Close()
	var result []*upal.AIProvider
	for rows.Next() {
		p := &upal.AIProvider{}
		var category string
		if err := rows.Scan(&p.ID, &p.Name, &category, &p.Type, &p.APIKey, &p.IsDefault); err != nil {
			return nil, fmt.Errorf("scan ai provider: %w", err)
		}
		p.Category = upal.AIProviderCategory(category)
		result = append(result, p)
	}
	return result, nil
}

func (d *DB) UpdateAIProvider(ctx context.Context, p *upal.AIProvider) error {
	_, err := d.Pool.ExecContext(ctx,
		`UPDATE ai_providers SET name=$1, category=$2, type=$3, api_key=$4, is_default=$5 WHERE id=$6`,
		p.Name, string(p.Category), p.Type, p.APIKey, p.IsDefault, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update ai provider: %w", err)
	}
	return nil
}

func (d *DB) DeleteAIProvider(ctx context.Context, id string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM ai_providers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ai provider: %w", err)
	}
	return nil
}

func (d *DB) ClearAIProviderDefault(ctx context.Context, category string) error {
	_, err := d.Pool.ExecContext(ctx,
		`UPDATE ai_providers SET is_default = FALSE WHERE category = $1`, category,
	)
	if err != nil {
		return fmt.Errorf("clear ai provider default: %w", err)
	}
	return nil
}
