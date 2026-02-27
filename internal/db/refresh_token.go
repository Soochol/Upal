package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// RefreshToken represents a stored refresh token record.
type RefreshToken struct {
	ID         string
	UserID     string
	TokenHash  string
	DeviceInfo string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy string
}

// HashToken returns SHA-256 hex of the given token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CreateRefreshToken stores a new refresh token record.
func (d *DB) CreateRefreshToken(ctx context.Context, rt *RefreshToken) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, device_info, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		rt.ID, rt.UserID, rt.TokenHash, rt.DeviceInfo, rt.CreatedAt, rt.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken retrieves a refresh token by its ID (jti).
func (d *DB) GetRefreshToken(ctx context.Context, id string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	var revokedAt sql.NullTime
	var replacedBy sql.NullString
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, device_info, created_at, expires_at, revoked_at, replaced_by
		 FROM refresh_tokens WHERE id = $1`, id,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.DeviceInfo, &rt.CreatedAt, &rt.ExpiresAt, &revokedAt, &replacedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if revokedAt.Valid {
		rt.RevokedAt = &revokedAt.Time
	}
	if replacedBy.Valid {
		rt.ReplacedBy = replacedBy.String
	}
	return rt, nil
}

// RevokeRefreshToken marks a single token as revoked and optionally sets replaced_by.
func (d *DB) RevokeRefreshToken(ctx context.Context, id, replacedBy string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW(), replaced_by = $2 WHERE id = $1 AND revoked_at IS NULL`
	_, err := d.Pool.ExecContext(ctx, query, id, sql.NullString{String: replacedBy, Valid: replacedBy != ""})
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllRefreshTokens revokes all active tokens for a user (family revocation).
func (d *DB) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	_, err := d.Pool.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}
	return nil
}

// CleanupExpiredRefreshTokens deletes tokens that expired more than 1 day ago.
func (d *DB) CleanupExpiredRefreshTokens(ctx context.Context) (int64, error) {
	result, err := d.Pool.ExecContext(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '1 day'`,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired refresh tokens: %w", err)
	}
	return result.RowsAffected()
}
