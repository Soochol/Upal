package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MCPServerRow represents an MCP server stored in the database.
type MCPServerRow struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CreateMCPServer inserts a new MCP server configuration.
func (d *DB) CreateMCPServer(ctx context.Context, userID, name string, config json.RawMessage) error {
	id := fmt.Sprintf("mcp_%d", time.Now().UnixNano())
	now := time.Now()
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO mcp_servers (id, user_id, name, config, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, name, config, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert mcp_server: %w", err)
	}
	return nil
}

// ListMCPServers returns all MCP servers for a user.
func (d *DB) ListMCPServers(ctx context.Context, userID string) ([]MCPServerRow, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, user_id, name, config, created_at, updated_at
		 FROM mcp_servers WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list mcp_servers: %w", err)
	}
	defer rows.Close()

	var result []MCPServerRow
	for rows.Next() {
		var row MCPServerRow
		if err := rows.Scan(&row.ID, &row.UserID, &row.Name, &row.Config, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan mcp_server: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp_servers: %w", err)
	}
	return result, nil
}

// DeleteMCPServer removes an MCP server by ID, scoped to a user.
func (d *DB) DeleteMCPServer(ctx context.Context, id, userID string) error {
	res, err := d.Pool.ExecContext(ctx,
		`DELETE FROM mcp_servers WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete mcp_server: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mcp server %q not found", id)
	}
	return nil
}
