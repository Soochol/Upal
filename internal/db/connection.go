package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

func (d *DB) CreateConnection(ctx context.Context, c *upal.Connection) error {
	extrasJSON, _ := json.Marshal(c.Extras)
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO connections (id, name, type, host, port, login, password, token, extras)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		c.ID, c.Name, string(c.Type), c.Host, c.Port, c.Login, c.Password, c.Token, extrasJSON,
	)
	if err != nil {
		return fmt.Errorf("insert connection: %w", err)
	}
	return nil
}

func (d *DB) GetConnection(ctx context.Context, id string) (*upal.Connection, error) {
	c := &upal.Connection{}
	var extrasJSON []byte
	var connType string

	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, name, type, host, port, login, password, token, extras FROM connections WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &connType, &c.Host, &c.Port, &c.Login, &c.Password, &c.Token, &extrasJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	c.Type = upal.ConnectionType(connType)
	json.Unmarshal(extrasJSON, &c.Extras)
	return c, nil
}

func (d *DB) ListConnections(ctx context.Context) ([]*upal.Connection, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, name, type, host, port, login, password, token, extras FROM connections ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()

	var result []*upal.Connection
	for rows.Next() {
		c := &upal.Connection{}
		var extrasJSON []byte
		var connType string

		if err := rows.Scan(&c.ID, &c.Name, &connType, &c.Host, &c.Port, &c.Login, &c.Password, &c.Token, &extrasJSON); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		c.Type = upal.ConnectionType(connType)
		json.Unmarshal(extrasJSON, &c.Extras)
		result = append(result, c)
	}
	return result, nil
}

func (d *DB) UpdateConnection(ctx context.Context, c *upal.Connection) error {
	extrasJSON, _ := json.Marshal(c.Extras)
	_, err := d.Pool.ExecContext(ctx,
		`UPDATE connections SET name=$1, type=$2, host=$3, port=$4, login=$5, password=$6, token=$7, extras=$8 WHERE id=$9`,
		c.Name, string(c.Type), c.Host, c.Port, c.Login, c.Password, c.Token, extrasJSON, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update connection: %w", err)
	}
	return nil
}

func (d *DB) DeleteConnection(ctx context.Context, id string) error {
	_, err := d.Pool.ExecContext(ctx, `DELETE FROM connections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	return nil
}
