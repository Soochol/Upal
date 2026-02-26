package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func (d *DB) UpsertUser(ctx context.Context, user *upal.User) (*upal.User, error) {
	if user.ID == "" {
		user.ID = upal.GenerateID("usr")
	}
	now := time.Now()
	user.UpdatedAt = now

	err := d.Pool.QueryRowContext(ctx,
		`INSERT INTO users (id, email, name, avatar_url, oauth_provider, oauth_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (oauth_provider, oauth_id) DO UPDATE
		 SET email = EXCLUDED.email, name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url, updated_at = EXCLUDED.updated_at
		 RETURNING id, created_at, updated_at`,
		user.ID, user.Email, user.Name, user.AvatarURL, user.OAuthProvider, user.OAuthID, now, now,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}

func (d *DB) GetUserByOAuth(ctx context.Context, provider, oauthID string) (*upal.User, error) {
	u := &upal.User{}
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, email, name, avatar_url, oauth_provider, oauth_id, created_at, updated_at
		 FROM users WHERE oauth_provider = $1 AND oauth_id = $2`, provider, oauthID,
	).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.OAuthProvider, &u.OAuthID, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by oauth: %w", err)
	}
	return u, nil
}

func (d *DB) GetUser(ctx context.Context, id string) (*upal.User, error) {
	u := &upal.User{}
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, email, name, avatar_url, oauth_provider, oauth_id, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.OAuthProvider, &u.OAuthID, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}
