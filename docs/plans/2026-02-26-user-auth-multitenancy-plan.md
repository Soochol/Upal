# User Authentication & Multi-Tenancy Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Google/GitHub OAuth login and per-user data isolation so each user manages their own workflows, pipelines, providers, and MCP servers.

**Architecture:** Go-native OAuth2 + JWT auth with Chi middleware. All existing tables gain a `user_id` column. Repository and DB layers filter by user. Frontend gets a login page, auth context, and protected routes.

**Tech Stack:** `golang.org/x/oauth2`, `github.com/golang-jwt/jwt/v5`, Chi middleware, React context + React Query

**Design doc:** `docs/plans/2026-02-26-user-auth-multitenancy-design.md`

---

## Phase 1: Backend — User Domain & Auth Foundation

### Task 1: Add JWT and OAuth2 dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add Go dependencies**

Run:
```bash
cd /home/dev/code/Upal && go get golang.org/x/oauth2 github.com/golang-jwt/jwt/v5
```

**Step 2: Tidy modules**

Run:
```bash
cd /home/dev/code/Upal && go mod tidy
```

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add oauth2 and jwt dependencies"
```

---

### Task 2: Add auth config to Config struct

**Files:**
- Modify: `internal/config/config.go`
- Modify: `config.yaml`

**Step 1: Add AuthConfig struct and field**

In `internal/config/config.go`, add to the Config struct:

```go
type AuthConfig struct {
	Google    OAuthProviderConfig `yaml:"google"`
	GitHub    OAuthProviderConfig `yaml:"github"`
	JWTSecret string             `yaml:"jwt_secret"`
}

type OAuthProviderConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}
```

Add `Auth AuthConfig `yaml:"auth"`` field to the `Config` struct.

In `applyEnvOverrides`, add env overrides:
- `GOOGLE_CLIENT_ID` → `cfg.Auth.Google.ClientID`
- `GOOGLE_CLIENT_SECRET` → `cfg.Auth.Google.ClientSecret`
- `GITHUB_CLIENT_ID` → `cfg.Auth.GitHub.ClientID`
- `GITHUB_CLIENT_SECRET` → `cfg.Auth.GitHub.ClientSecret`
- `JWT_SECRET` → `cfg.Auth.JWTSecret`

**Step 2: Add auth section to config.yaml**

```yaml
auth:
  google:
    client_id: ""    # Set GOOGLE_CLIENT_ID in .env
    client_secret: "" # Set GOOGLE_CLIENT_SECRET in .env
  github:
    client_id: ""    # Set GITHUB_CLIENT_ID in .env
    client_secret: "" # Set GITHUB_CLIENT_SECRET in .env
  jwt_secret: ""     # Set JWT_SECRET in .env
```

**Step 3: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 4: Commit**

```bash
git add internal/config/config.go config.yaml
git commit -m "feat: add auth configuration for OAuth and JWT"
```

---

### Task 3: Create User domain type

**Files:**
- Create: `internal/upal/user.go`

**Step 1: Define User struct**

```go
package upal

import "time"

// User represents an authenticated platform user.
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	AvatarURL     string    `json:"avatar_url"`
	OAuthProvider string    `json:"oauth_provider"` // "google" | "github"
	OAuthID       string    `json:"oauth_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
```

**Step 2: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 3: Commit**

```bash
git add internal/upal/user.go
git commit -m "feat: add User domain type"
```

---

### Task 4: Create users table in DB schema

**Files:**
- Modify: `internal/db/db.go` (schema section)
- Create: `internal/db/user.go`

**Step 1: Add users table to schema in db.go**

Add to the `Migrate` method's schema SQL (BEFORE all other tables so foreign keys work):

```sql
CREATE TABLE IF NOT EXISTS users (
    id             TEXT PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL DEFAULT '',
    avatar_url     TEXT NOT NULL DEFAULT '',
    oauth_provider TEXT NOT NULL,
    oauth_id       TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(oauth_provider, oauth_id)
);
```

**Step 2: Create user DB operations in `internal/db/user.go`**

Implement CRUD following the existing pattern in `internal/db/workflow.go`:

```go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func (d *DB) UpsertUser(ctx context.Context, u *upal.User) (*upal.User, error) {
	now := time.Now()
	if u.ID == "" {
		u.ID = upal.GenerateID("usr")
	}
	u.UpdatedAt = now

	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO users (id, email, name, avatar_url, oauth_provider, oauth_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (oauth_provider, oauth_id) DO UPDATE SET
		   email = EXCLUDED.email, name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url, updated_at = EXCLUDED.updated_at
		 `,
		u.ID, u.Email, u.Name, u.AvatarURL, u.OAuthProvider, u.OAuthID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return d.GetUserByOAuth(ctx, u.OAuthProvider, u.OAuthID)
}

func (d *DB) GetUserByOAuth(ctx context.Context, provider, oauthID string) (*upal.User, error) {
	var u upal.User
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
	return &u, nil
}

func (d *DB) GetUser(ctx context.Context, id string) (*upal.User, error) {
	var u upal.User
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
	return &u, nil
}
```

**Step 3: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 4: Commit**

```bash
git add internal/db/db.go internal/db/user.go
git commit -m "feat: add users table and DB operations"
```

---

### Task 5: Create AuthService

**Files:**
- Create: `internal/services/auth.go`

**Step 1: Implement AuthService**

AuthService handles OAuth2 flow, JWT token generation/validation, and user lookup/creation.

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	googleOAuth "golang.org/x/oauth2/google"
)

type AuthService struct {
	db          *db.DB
	jwtSecret   []byte
	googleCfg   *oauth2.Config
	githubCfg   *oauth2.Config
}

func NewAuthService(database *db.DB, authCfg config.AuthConfig, baseURL string) *AuthService {
	s := &AuthService{
		db:        database,
		jwtSecret: []byte(authCfg.JWTSecret),
	}

	if authCfg.Google.ClientID != "" {
		s.googleCfg = &oauth2.Config{
			ClientID:     authCfg.Google.ClientID,
			ClientSecret: authCfg.Google.ClientSecret,
			Endpoint:     googleOAuth.Endpoint,
			RedirectURL:  baseURL + "/api/auth/callback/google",
			Scopes:       []string{"openid", "email", "profile"},
		}
	}

	if authCfg.GitHub.ClientID != "" {
		s.githubCfg = &oauth2.Config{
			ClientID:     authCfg.GitHub.ClientID,
			ClientSecret: authCfg.GitHub.ClientSecret,
			Endpoint:     github.Endpoint,
			RedirectURL:  baseURL + "/api/auth/callback/github",
			Scopes:       []string{"user:email"},
		}
	}

	return s
}

// OAuthConfig returns the oauth2 config for the given provider.
func (s *AuthService) OAuthConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "google":
		if s.googleCfg == nil {
			return nil, fmt.Errorf("google oauth not configured")
		}
		return s.googleCfg, nil
	case "github":
		if s.githubCfg == nil {
			return nil, fmt.Errorf("github oauth not configured")
		}
		return s.githubCfg, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// ExchangeAndUpsertUser exchanges the OAuth code, fetches user info, and upserts to DB.
func (s *AuthService) ExchangeAndUpsertUser(ctx context.Context, provider, code string) (*upal.User, error) {
	cfg, err := s.OAuthConfig(provider)
	if err != nil {
		return nil, err
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange: %w", err)
	}

	user, err := s.fetchUserInfo(ctx, provider, token)
	if err != nil {
		return nil, err
	}

	return s.db.UpsertUser(ctx, user)
}

func (s *AuthService) fetchUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*upal.User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	switch provider {
	case "google":
		return s.fetchGoogleUser(client)
	case "github":
		return s.fetchGitHubUser(client)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

func (s *AuthService) fetchGoogleUser(client *http.Client) (*upal.User, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode google user: %w", err)
	}

	return &upal.User{
		Email:         info.Email,
		Name:          info.Name,
		AvatarURL:     info.Picture,
		OAuthProvider: "google",
		OAuthID:       info.ID,
	}, nil
}

func (s *AuthService) fetchGitHubUser(client *http.Client) (*upal.User, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("github user: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode github user: %w", err)
	}

	name := info.Name
	if name == "" {
		name = info.Login
	}

	// GitHub email may be private; fetch from /user/emails if empty
	email := info.Email
	if email == "" {
		email, _ = s.fetchGitHubPrimaryEmail(client)
	}

	return &upal.User{
		Email:         email,
		Name:          name,
		AvatarURL:     info.AvatarURL,
		OAuthProvider: "github",
		OAuthID:       fmt.Sprintf("%d", info.ID),
	}, nil
}

func (s *AuthService) fetchGitHubPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", fmt.Errorf("no email found")
}

// GenerateTokens creates an access token and a refresh token for the user.
func (s *AuthService) GenerateTokens(user *upal.User) (accessToken string, refreshToken string, err error) {
	now := time.Now()

	access := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"iat":   now.Unix(),
		"exp":   now.Add(30 * time.Minute).Unix(),
	})
	accessToken, err = access.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(7 * 24 * time.Hour).Unix(),
	})
	refreshToken, err = refresh.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ValidateAccessToken parses and validates a JWT access token, returning the user ID.
func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Reject refresh tokens used as access tokens
	if claims["type"] == "refresh" {
		return "", fmt.Errorf("refresh token cannot be used as access token")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("missing sub claim")
	}

	return sub, nil
}

// ValidateRefreshToken parses a refresh token and returns the user ID.
func (s *AuthService) ValidateRefreshToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}

	if claims["type"] != "refresh" {
		return "", fmt.Errorf("not a refresh token")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("missing sub claim")
	}

	return sub, nil
}

// GetUser looks up a user by ID.
func (s *AuthService) GetUser(ctx context.Context, id string) (*upal.User, error) {
	return s.db.GetUser(ctx, id)
}

// Enabled returns true if at least one OAuth provider is configured.
func (s *AuthService) Enabled() bool {
	return s.googleCfg != nil || s.githubCfg != nil
}
```

**Step 2: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 3: Commit**

```bash
git add internal/services/auth.go
git commit -m "feat: add AuthService with OAuth2 and JWT support"
```

---

### Task 6: Create auth middleware

**Files:**
- Create: `internal/api/auth_middleware.go`

**Step 1: Implement JWT auth middleware**

```go
package api

import (
	"net/http"
	"strings"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// AuthMiddleware extracts and validates JWT from Authorization header,
// then injects userID into request context.
func AuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for auth endpoints and static files
			if strings.HasPrefix(r.URL.Path, "/api/auth/") || !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// If auth is not configured, use default user
			if !authSvc.Enabled() {
				ctx := upal.WithUserID(r.Context(), "default")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			userID, err := authSvc.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := upal.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

**Step 2: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 3: Commit**

```bash
git add internal/api/auth_middleware.go
git commit -m "feat: add JWT auth middleware"
```

---

### Task 7: Create auth API handlers

**Files:**
- Create: `internal/api/auth.go`
- Modify: `internal/api/server.go` (add authSvc field + routes)

**Step 1: Implement auth handlers**

```go
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/services"
)

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	cfg, err := s.authSvc.OAuthConfig(provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	url := cfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) authCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	user, err := s.authSvc.ExchangeAndUpsertUser(r.Context(), provider, code)
	if err != nil {
		slog.Error("oauth callback failed", "provider", provider, "err", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	accessToken, refreshToken, err := s.authSvc.GenerateTokens(user)
	if err != nil {
		http.Error(w, "token generation failed", http.StatusInternalServerError)
		return
	}

	// Set refresh token as httpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	// Redirect to frontend with access token as fragment
	http.Redirect(w, r, "/?token="+accessToken, http.StatusTemporaryRedirect)
}

func (s *Server) authRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "no refresh token", http.StatusUnauthorized)
		return
	}

	userID, err := s.authSvc.ValidateRefreshToken(cookie.Value)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	user, err := s.authSvc.GetUser(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := s.authSvc.GenerateTokens(user)
	if err != nil {
		http.Error(w, "token generation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": accessToken})
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	userID := upalUserIDFromContext(r.Context())
	if userID == "" || userID == "default" {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	user, err := s.authSvc.GetUser(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

**Step 2: Add authSvc to Server struct and register routes**

In `internal/api/server.go`:
- Add `authSvc *services.AuthService` field to `Server` struct
- Add `SetAuthService(svc *services.AuthService)` setter method
- In `Handler()`, add `r.Use(AuthMiddleware(s.authSvc))` after CORS middleware
- In the `/api` route group, add auth routes:

```go
r.Route("/auth", func(r chi.Router) {
    r.Get("/login/{provider}", s.authLogin)
    r.Get("/callback/{provider}", s.authCallback)
    r.Post("/refresh", s.authRefresh)
    r.Post("/logout", s.authLogout)
    r.Get("/me", s.authMe)
})
```

Also add a helper to extract userID from context:

```go
func upalUserIDFromContext(ctx context.Context) string {
    return upal.UserIDFromContext(ctx)
}
```

**Step 3: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 4: Commit**

```bash
git add internal/api/auth.go internal/api/server.go
git commit -m "feat: add auth API handlers and register routes"
```

---

### Task 8: Wire AuthService in main.go

**Files:**
- Modify: `cmd/upal/main.go`

**Step 1: Create and wire AuthService**

After the database setup block, add:

```go
// Auth service (requires database for user storage)
var authSvc *services.AuthService
if database != nil {
    baseURL := fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
    authSvc = services.NewAuthService(database, cfg.Auth, baseURL)
}
```

After Server creation, add:
```go
if authSvc != nil {
    srv.SetAuthService(authSvc)
}
```

**Step 2: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 3: Run tests**

Run: `cd /home/dev/code/Upal && go test ./... -v -race -count=1 2>&1 | tail -30`

**Step 4: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: wire AuthService into application startup"
```

---

## Phase 2: Backend — Multi-Tenancy (user_id on all tables)

### Task 9: Add user_id column to all existing tables

**Files:**
- Modify: `internal/db/db.go` (schema SQL)

**Step 1: Add user_id column to all tables**

Add `user_id TEXT NOT NULL DEFAULT 'default'` to every existing table in the schema creation SQL:
- `workflows`
- `sessions`
- `events`
- `runs`
- `assets`
- `schedules`
- `triggers`
- `pipelines`
- `pipeline_runs`
- `content_sessions`
- `source_fetches`
- `llm_analyses`
- `published_content`
- `surge_events`
- `connections`
- `ai_providers`
- `workflow_results`
- `publish_channels`

Also add migration ALTERs after the CREATE TABLE statements (for existing databases):

```sql
-- Migration: add user_id to existing tables
DO $$ BEGIN
    ALTER TABLE workflows ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
    ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT 'default';
    -- ... repeat for all tables
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
```

Add indexes: `CREATE INDEX IF NOT EXISTS idx_{table}_user_id ON {table}(user_id);`

**Step 2: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "feat: add user_id column to all existing tables"
```

---

### Task 10: Update DB query functions with user_id filtering

**Files:**
- Modify: `internal/db/workflow.go`
- Modify: `internal/db/run.go`
- Modify: `internal/db/pipeline.go`
- Modify: `internal/db/content_session.go`
- Modify: `internal/db/ai_provider.go`
- Modify: `internal/db/connection.go`
- Modify: all other `internal/db/*.go` files with queries

**Step 1: Update each DB file**

For each query file, add `userID string` parameter to relevant functions and add `WHERE user_id = $N` / `AND user_id = $N` to queries. Also add `user_id` to INSERT statements.

Example for workflow.go:
- `CreateWorkflow(ctx, wf, userID)` → INSERT includes `user_id`
- `GetWorkflow(ctx, name, userID)` → `WHERE name = $1 AND user_id = $2`
- `ListWorkflows(ctx, userID)` → `WHERE user_id = $1`
- `DeleteWorkflow(ctx, name, userID)` → `WHERE name = $1 AND user_id = $2`

Repeat the same pattern for all DB operation files.

**Step 2: Fix compilation errors in repository layer**

Update all persistent repository implementations to pass `userID` from context to DB calls. Use `upal.UserIDFromContext(ctx)` to extract the user ID.

**Step 3: Verify build**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 4: Run tests**

Run: `cd /home/dev/code/Upal && go test ./... -v -race -count=1 2>&1 | tail -30`

**Step 5: Commit**

```bash
git add internal/db/ internal/repository/
git commit -m "feat: add user_id filtering to all DB queries and repositories"
```

---

### Task 11: Create mcp_servers table

**Files:**
- Modify: `internal/db/db.go` (add table to schema)
- Create: `internal/db/mcp_server.go`
- Create: `internal/repository/mcp_server.go`

**Step 1: Add mcp_servers table to schema**

```sql
CREATE TABLE IF NOT EXISTS mcp_servers (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_user_id ON mcp_servers(user_id);
```

**Step 2: Create MCP server DB operations**

In `internal/db/mcp_server.go`, implement CRUD following the existing pattern.

**Step 3: Create MCP server repository**

In `internal/repository/mcp_server.go`, implement with dual-layer pattern.

**Step 4: Verify build and commit**

```bash
go build ./...
git add internal/db/db.go internal/db/mcp_server.go internal/repository/mcp_server.go
git commit -m "feat: add mcp_servers table and repository"
```

---

## Phase 3: Frontend — Login & Auth

### Task 12: Add auth API client functions

**Files:**
- Create: `web/src/shared/api/auth.ts`
- Modify: `web/src/shared/api/client.ts` (add Authorization header)

**Step 1: Create auth API module**

```typescript
// web/src/shared/api/auth.ts
import { apiFetch, API_BASE } from './client'

export interface User {
  id: string
  email: string
  name: string
  avatar_url: string
  oauth_provider: string
}

export function fetchMe(): Promise<User> {
  return apiFetch<User>(`${API_BASE}/auth/me`)
}

export function refreshToken(): Promise<{ token: string }> {
  return apiFetch<{ token: string }>(`${API_BASE}/auth/refresh`, { method: 'POST' })
}

export function logout(): Promise<void> {
  return apiFetch<void>(`${API_BASE}/auth/logout`, { method: 'POST' })
}
```

**Step 2: Update API client to attach Authorization header**

In `web/src/shared/api/client.ts`, modify `apiFetch` to read token from a module-level getter and attach it. Also add 401 → auto-refresh logic.

```typescript
let getToken: (() => string | null) = () => null

export function setTokenGetter(fn: () => string | null) {
  getToken = fn
}

export async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const res = await fetch(url, { ...init, headers })
  // ... existing error handling ...
}
```

**Step 3: Commit**

```bash
git add web/src/shared/api/auth.ts web/src/shared/api/client.ts
git commit -m "feat: add auth API client and token injection"
```

---

### Task 13: Create auth store (Zustand)

**Files:**
- Create: `web/src/entities/auth/store.ts`
- Create: `web/src/entities/auth/index.ts`

**Step 1: Create auth Zustand store**

```typescript
// web/src/entities/auth/store.ts
import { create } from 'zustand'
import { fetchMe, refreshToken, logout as apiLogout, type User } from '@/shared/api/auth'
import { setTokenGetter } from '@/shared/api/client'

interface AuthState {
  user: User | null
  token: string | null
  loading: boolean
  initialized: boolean
  setToken: (token: string) => void
  init: () => Promise<void>
  refresh: () => Promise<boolean>
  logout: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => {
  // Wire token getter for API client
  setTokenGetter(() => get().token)

  return {
    user: null,
    token: null,
    loading: true,
    initialized: false,

    setToken: (token) => set({ token }),

    init: async () => {
      // Check URL for token from OAuth redirect
      const params = new URLSearchParams(window.location.search)
      const urlToken = params.get('token')
      if (urlToken) {
        set({ token: urlToken })
        window.history.replaceState({}, '', window.location.pathname)
      }

      // Try refresh if no token
      if (!get().token) {
        const ok = await get().refresh()
        if (!ok) {
          set({ loading: false, initialized: true })
          return
        }
      }

      // Fetch user info
      try {
        const user = await fetchMe()
        set({ user, loading: false, initialized: true })
      } catch {
        set({ token: null, user: null, loading: false, initialized: true })
      }
    },

    refresh: async () => {
      try {
        const { token } = await refreshToken()
        set({ token })
        return true
      } catch {
        return false
      }
    },

    logout: async () => {
      try { await apiLogout() } catch { /* ignore */ }
      set({ token: null, user: null })
    },
  }
})
```

**Step 2: Create barrel export**

```typescript
// web/src/entities/auth/index.ts
export { useAuthStore } from './store'
```

**Step 3: Commit**

```bash
git add web/src/entities/auth/
git commit -m "feat: add auth Zustand store with token management"
```

---

### Task 14: Create Login page

**Files:**
- Create: `web/src/pages/login/LoginPage.tsx`
- Create: `web/src/pages/login/index.ts`

**Step 1: Create the login page component**

A clean login page with Google and GitHub OAuth buttons. Use existing Shadcn Button component. Upal branding.

```typescript
// web/src/pages/login/LoginPage.tsx
import { Button } from '@/shared/ui/button'
import { API_BASE } from '@/shared/api/client'

export function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 rounded-xl border bg-card p-8 shadow-sm">
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold">Upal</h1>
          <p className="text-sm text-muted-foreground">Sign in to continue</p>
        </div>
        <div className="space-y-3">
          <Button
            variant="outline"
            className="w-full justify-center gap-2"
            onClick={() => window.location.href = `${API_BASE}/auth/login/google`}
          >
            <GoogleIcon />
            Continue with Google
          </Button>
          <Button
            variant="outline"
            className="w-full justify-center gap-2"
            onClick={() => window.location.href = `${API_BASE}/auth/login/github`}
          >
            <GitHubIcon />
            Continue with GitHub
          </Button>
        </div>
      </div>
    </div>
  )
}

function GoogleIcon() {
  return (
    <svg className="h-4 w-4" viewBox="0 0 24 24">
      <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" />
      <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
      <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
      <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
    </svg>
  )
}

function GitHubIcon() {
  return (
    <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z" />
    </svg>
  )
}
```

**Step 2: Barrel export**

```typescript
// web/src/pages/login/index.ts
export { LoginPage } from './LoginPage'
```

**Step 3: Commit**

```bash
git add web/src/pages/login/
git commit -m "feat: add login page with Google and GitHub OAuth buttons"
```

---

### Task 15: Add auth guard and integrate into router

**Files:**
- Modify: `web/src/app/router.tsx`
- Modify: `web/src/app/providers.tsx`

**Step 1: Create AuthGuard wrapper**

In `router.tsx`, add a guard component that redirects to `/login` if not authenticated:

```typescript
function AuthGuard({ children }: { children: ReactNode }) {
  const { user, loading, initialized } = useAuthStore()

  if (!initialized || loading) {
    return <div className="flex h-screen items-center justify-center">
      <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
    </div>
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
```

**Step 2: Update routes**

Wrap all authenticated routes in `AuthGuard`, add `/login` route:

```typescript
<Routes>
  <Route path="/login" element={<LoginPage />} />
  <Route path="/*" element={
    <AuthGuard>
      {/* existing routes */}
    </AuthGuard>
  } />
</Routes>
```

**Step 3: Init auth on app start**

In `providers.tsx`, call `useAuthStore.getState().init()` on mount.

**Step 4: Verify frontend builds**

Run: `cd /home/dev/code/Upal/web && npx tsc -b && npm run build`

**Step 5: Commit**

```bash
git add web/src/app/router.tsx web/src/app/providers.tsx
git commit -m "feat: add auth guard and protected routes"
```

---

### Task 16: Add user avatar and logout to sidebar

**Files:**
- Modify: `web/src/app/layout.tsx`

**Step 1: Add user section to sidebar**

At the bottom of the sidebar navigation, add user avatar and logout button:

```typescript
// In MainLayout, at the bottom of sidebar
const { user, logout } = useAuthStore()

// Render at bottom of sidebar nav:
{user && (
  <div className="mt-auto border-t p-2">
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="flex w-full items-center gap-2 rounded-md p-2 hover:bg-accent">
          <img src={user.avatar_url} alt="" className="h-6 w-6 rounded-full" />
          <span className="truncate text-sm">{user.name}</span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuItem onClick={logout}>Log out</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>
)}
```

**Step 2: Verify frontend builds**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`

**Step 3: Commit**

```bash
git add web/src/app/layout.tsx
git commit -m "feat: add user avatar and logout to sidebar"
```

---

## Phase 4: Integration & Verification

### Task 17: Update CORS to include credentials

**Files:**
- Modify: `internal/api/server.go`

**Step 1: Update CORS config**

Change `AllowedOrigins` from `["*"]` to the actual frontend origins (wildcard doesn't work with credentials), and ensure `AllowCredentials: true`:

```go
cors.Handler(cors.Options{
    AllowOriginFunc: func(r *http.Request, origin string) bool { return true },
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
})
```

**Step 2: Commit**

```bash
git add internal/api/server.go
git commit -m "fix: update CORS to support auth credentials"
```

---

### Task 18: End-to-end verification

**Step 1: Build backend**

Run: `cd /home/dev/code/Upal && go build ./...`

**Step 2: Build frontend**

Run: `cd /home/dev/code/Upal/web && npm run build`

**Step 3: Run all tests**

Run: `cd /home/dev/code/Upal && go test ./... -v -race -count=1`

**Step 4: Run frontend type check**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`

**Step 5: Fix any issues found**

**Step 6: Final commit**

```bash
git add -A
git commit -m "feat: complete user auth and multi-tenancy implementation"
```
