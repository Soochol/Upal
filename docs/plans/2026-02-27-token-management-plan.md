# Token Management Redesign — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** DB 기반 refresh token rotation + 프론트엔드 선제적 갱신으로 토큰 만료 에러를 근본적으로 해결하고 보안을 강화한다.

**Architecture:** refresh token을 DB에 저장하여 발급/검증/폐기를 서버에서 관리한다. JWT의 `jti` claim으로 DB 레코드를 연결한다. 프론트엔드는 만료 전 선제적으로 토큰을 갱신하여 401을 예방하고, 실패 시 자동 로그아웃한다.

**Tech Stack:** Go (jwt-go, crypto/sha256), PostgreSQL, React (Zustand), TypeScript

---

### Task 1: DB 마이그레이션 — `refresh_tokens` 테이블

**Files:**
- Modify: `internal/db/db.go:398` (migrationSQL 끝에 추가)

**Step 1: 마이그레이션 SQL 추가**

`internal/db/db.go`의 `migrationSQL` 상수 끝(닫는 백틱 직전)에 추가:

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    device_info TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    replaced_by TEXT
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

**Step 2: 테스트 실행**

Run: `go build ./...`
Expected: BUILD SUCCESS

**Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "feat: add refresh_tokens table migration"
```

---

### Task 2: DB refresh token CRUD

**Files:**
- Create: `internal/db/refresh_token.go`
- Create: `internal/db/refresh_token_test.go`

**Step 1: Write the failing test**

`internal/db/refresh_token_test.go` — DB 없이 동작하는 단위 테스트는 불가하므로, 빌드 확인 위주로 진행. 실제 통합 테스트는 Task 5에서 수행.

**Step 2: Implement refresh_token.go**

```go
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
```

**Step 3: 빌드 확인**

Run: `go build ./internal/db/...`
Expected: BUILD SUCCESS

**Step 4: Commit**

```bash
git add internal/db/refresh_token.go
git commit -m "feat: add refresh token DB CRUD operations"
```

---

### Task 3: AuthService — TTL 변경, jti claim, DB 연동, token rotation

**Files:**
- Modify: `internal/services/auth.go:21-24` (TTL 상수)
- Modify: `internal/services/auth.go:246-271` (GenerateTokens, signToken)
- Add new methods to `internal/services/auth.go`

**Step 1: TTL 상수 변경**

`internal/services/auth.go:21-24`:
```go
const (
	accessTokenTTL  = 1 * time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)
```

**Step 2: signToken에 jti claim 추가**

`internal/services/auth.go` — `signToken` 메서드에 `jti` 파라미터 추가:

```go
func (s *AuthService) signToken(userID, tokenType, jti string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID,
		"type": tokenType,
		"iat":  now.Unix(),
		"exp":  now.Add(ttl).Unix(),
	}
	if jti != "" {
		claims["jti"] = jti
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign %s token: %w", tokenType, err)
	}
	return signed, nil
}
```

**Step 3: GenerateTokens 변경 — refresh token에 jti 포함, DB 저장**

`GenerateTokens` 시그니처 변경 + DB 저장 로직:

```go
func (s *AuthService) GenerateTokens(ctx context.Context, user *upal.User, deviceInfo string) (access, refresh string, err error) {
	access, err = s.signToken(user.ID, "access", "", accessTokenTTL)
	if err != nil {
		return "", "", err
	}

	jti := upal.GenerateID("rt")
	refresh, err = s.signToken(user.ID, "refresh", jti, refreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	// Store refresh token in DB
	if err := s.database.CreateRefreshToken(ctx, &db.RefreshToken{
		ID:         jti,
		UserID:     user.ID,
		TokenHash:  db.HashToken(refresh),
		DeviceInfo: deviceInfo,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(refreshTokenTTL),
	}); err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}

	return access, refresh, nil
}
```

**Step 4: validateToken에서 jti 추출**

`validateToken` 반환값에 jti 추가:

```go
type tokenClaims struct {
	UserID string
	JTI    string
}

func (s *AuthService) validateToken(tokenStr, expectedType string) (*tokenClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, tokenType)
	}

	userID, _ := claims["sub"].(string)
	if userID == "" {
		return nil, fmt.Errorf("missing user ID in token")
	}

	jti, _ := claims["jti"].(string)

	return &tokenClaims{UserID: userID, JTI: jti}, nil
}
```

**Step 5: ValidateAccessToken / ValidateRefreshToken 반환값 업데이트**

```go
func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	claims, err := s.validateToken(tokenStr, "access")
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (s *AuthService) ValidateRefreshToken(tokenStr string) (*tokenClaims, error) {
	return s.validateToken(tokenStr, "refresh")
}
```

**Step 6: RotateRefreshToken — DB 기반 rotation 메서드**

```go
// RotateRefreshToken validates the refresh token against DB, revokes old, issues new pair.
// Returns (accessToken, refreshToken, error).
// If the token was already revoked → family revocation (all user tokens revoked).
func (s *AuthService) RotateRefreshToken(ctx context.Context, refreshTokenStr, deviceInfo string) (string, string, error) {
	claims, err := s.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.JTI == "" {
		return "", "", fmt.Errorf("refresh token missing jti")
	}

	// Check DB record
	stored, err := s.database.GetRefreshToken(ctx, claims.JTI)
	if err != nil {
		return "", "", fmt.Errorf("db lookup: %w", err)
	}
	if stored == nil {
		return "", "", fmt.Errorf("refresh token not found in DB")
	}

	// Token reuse detection — already revoked means stolen token
	if stored.RevokedAt != nil {
		// Family revocation: revoke ALL tokens for this user
		_ = s.database.RevokeAllRefreshTokens(ctx, stored.UserID)
		return "", "", fmt.Errorf("refresh token reuse detected, all sessions revoked")
	}

	// Verify hash matches
	if stored.TokenHash != db.HashToken(refreshTokenStr) {
		return "", "", fmt.Errorf("refresh token hash mismatch")
	}

	// Get user for new token generation
	user, err := s.GetUser(ctx, stored.UserID)
	if err != nil {
		return "", "", fmt.Errorf("get user: %w", err)
	}

	// Generate new token pair
	newAccess, newRefresh, err := s.GenerateTokens(ctx, user, deviceInfo)
	if err != nil {
		return "", "", fmt.Errorf("generate new tokens: %w", err)
	}

	// Extract new jti from the newly generated refresh token
	newClaims, _ := s.validateToken(newRefresh, "refresh")
	newJTI := ""
	if newClaims != nil {
		newJTI = newClaims.JTI
	}

	// Revoke old token, link to new
	if err := s.database.RevokeRefreshToken(ctx, claims.JTI, newJTI); err != nil {
		return "", "", fmt.Errorf("revoke old token: %w", err)
	}

	return newAccess, newRefresh, nil
}
```

**Step 7: RevokeRefreshToken — 로그아웃용**

```go
// RevokeUserRefreshToken revokes a specific refresh token by its JWT string.
func (s *AuthService) RevokeUserRefreshToken(refreshTokenStr string) error {
	claims, err := s.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		// Token is invalid/expired — nothing to revoke
		return nil
	}
	if claims.JTI == "" {
		return nil
	}
	return s.database.RevokeRefreshToken(context.Background(), claims.JTI, "")
}
```

**Step 8: Cleanup goroutine 추가**

```go
// StartCleanup starts a background goroutine that periodically removes expired refresh tokens.
func (s *AuthService) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := s.database.CleanupExpiredRefreshTokens(ctx); err != nil {
					slog.Error("refresh token cleanup failed", "err", err)
				} else if n > 0 {
					slog.Info("cleaned up expired refresh tokens", "count", n)
				}
			}
		}
	}()
}
```

(import `"log/slog"` 추가 필요)

**Step 9: 빌드 확인**

Run: `go build ./...`
Expected: 컴파일 에러 — `GenerateTokens` 호출부가 새 시그니처와 불일치. Task 4에서 수정.

**Step 10: Commit**

```bash
git add internal/services/auth.go
git commit -m "feat: add refresh token rotation, DB validation, cleanup in AuthService"
```

---

### Task 4: API 핸들러 업데이트 — refresh, callback, logout

**Files:**
- Modify: `internal/api/auth.go:15` (cookie MaxAge)
- Modify: `internal/api/auth.go:69-125` (authCallback)
- Modify: `internal/api/auth.go:127-158` (authRefresh)
- Modify: `internal/api/auth.go:160-163` (authLogout)

**Step 1: Cookie MaxAge 변경**

`internal/api/auth.go:15`:
```go
const refreshTokenMaxAge = 30 * 24 * 60 * 60 // 30 days
```

**Step 2: authCallback — GenerateTokens 호출 시그니처 업데이트**

`internal/api/auth.go:112` 변경:
```go
accessToken, refreshToken, err := s.authSvc.GenerateTokens(r.Context(), user, r.UserAgent())
```

**Step 3: authRefresh — DB 기반 rotation으로 교체**

`internal/api/auth.go:127-158` 전체 교체:
```go
func (s *Server) authRefresh(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w) {
		return
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, `{"error":"missing refresh token"}`, http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := s.authSvc.RotateRefreshToken(r.Context(), cookie.Value, r.UserAgent())
	if err != nil {
		slog.Warn("refresh token rotation failed", "err", err)
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	setRefreshTokenCookie(w, refreshToken)
	writeJSON(w, map[string]string{"token": accessToken})
}
```

**Step 4: authLogout — DB revoke 추가**

`internal/api/auth.go:160-163` 교체:
```go
func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		_ = s.authSvc.RevokeUserRefreshToken(cookie.Value)
	}
	setRefreshTokenCookie(w, "")
	w.WriteHeader(http.StatusNoContent)
}
```

**Step 5: 빌드 확인**

Run: `go build ./...`
Expected: BUILD SUCCESS

**Step 6: Commit**

```bash
git add internal/api/auth.go
git commit -m "feat: update auth handlers for DB-based token rotation"
```

---

### Task 5: 테스트 업데이트

**Files:**
- Modify: `internal/api/auth_test.go`

**Step 1: 기존 테스트가 새 시그니처에 맞게 컴파일되도록 수정**

`GenerateTokens` 호출부를 모두 업데이트. `newTestAuthService()`는 DB가 nil이므로 `GenerateTokens`에서 DB 저장 시 에러가 날 수 있다. DB 없는 테스트에서는 JWT 발급/검증 round-trip만 테스트하도록 조정.

- `TestAuthService_TokenRoundTrip`: DB 저장 없는 순수 JWT 테스트로 `signToken`을 직접 호출하는 방식으로 변경하거나, `GenerateTokens`가 DB nil일 때 graceful하게 동작하도록 guard 추가.

**접근법**: `GenerateTokens`에서 `s.database == nil`이면 DB 저장을 skip하도록 guard를 추가한다. 이렇게 하면 기존 테스트가 최소 변경으로 통과한다.

```go
// In GenerateTokens, after creating the refresh token:
if s.database != nil {
    if err := s.database.CreateRefreshToken(ctx, &db.RefreshToken{...}); err != nil {
        return "", "", fmt.Errorf("store refresh token: %w", err)
    }
}
```

동일하게 `RotateRefreshToken`, `RevokeUserRefreshToken`에도 DB nil guard.

**Step 2: GenerateTokens 호출부 업데이트**

`auth_test.go`의 `GenerateTokens` 호출부:
- `authSvc.GenerateTokens(testUser)` → `authSvc.GenerateTokens(context.Background(), testUser, "")`

**Step 3: 테스트 실행**

Run: `go test ./internal/api/... -v -race -run TestAuth`
Expected: ALL PASS

**Step 4: 전체 테스트 실행**

Run: `go test ./... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/services/auth.go internal/api/auth_test.go
git commit -m "test: update auth tests for new GenerateTokens signature"
```

---

### Task 6: JWTSecret 영속화

**Files:**
- Modify: `internal/services/auth.go:33-48` (NewAuthService)

**Step 1: JWTSecret 파일 읽기/쓰기 로직**

`NewAuthService`에서 secret이 비어있을 때 `data/jwt_secret` 파일을 먼저 확인하고, 없으면 생성:

```go
func NewAuthService(database *db.DB, authCfg config.AuthConfig, baseURL string) *AuthService {
	secret := authCfg.JWTSecret
	if secret == "" {
		secret = loadOrCreateJWTSecret("data/jwt_secret")
	}
	return &AuthService{
		database:  database,
		authCfg:   authCfg,
		baseURL:   baseURL,
		jwtSecret: []byte(secret),
	}
}

func loadOrCreateJWTSecret(path string) string {
	data, err := os.ReadFile(path)
	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		slog.Info("loaded JWT secret from file", "path", path)
		return strings.TrimSpace(string(data))
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	secret := hex.EncodeToString(b)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		slog.Warn("cannot create directory for JWT secret", "path", path, "err", err)
		return secret
	}
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		slog.Warn("cannot persist JWT secret", "path", path, "err", err)
	} else {
		slog.Info("generated and persisted new JWT secret", "path", path)
	}
	return secret
}
```

(import `"os"`, `"path/filepath"`, `"strings"`, `"log/slog"` 추가)

**Step 2: 빌드 + 테스트**

Run: `go build ./... && go test ./internal/services/... -v -race`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/services/auth.go
git commit -m "feat: persist JWT secret to data/jwt_secret file"
```

---

### Task 7: Cleanup goroutine wiring

**Files:**
- Modify: `cmd/upal/main.go:117` (authSvc 생성 후)

**Step 1: StartCleanup 호출 추가**

`cmd/upal/main.go`에서 authSvc 생성 직후에:

```go
authSvc = services.NewAuthService(database, cfg.Auth, baseURL)
authSvc.StartCleanup(context.Background())
```

**Step 2: 빌드 확인**

Run: `go build ./...`
Expected: BUILD SUCCESS

**Step 3: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: start refresh token cleanup goroutine on boot"
```

---

### Task 8: 프론트엔드 — 선제적 토큰 갱신 + graceful 로그아웃

**Files:**
- Modify: `web/src/shared/api/client.ts`
- Modify: `web/src/entities/auth/store.ts`

**Step 1: client.ts — tryRefresh 실패 시 authStore 초기화 + scheduleRefresh**

`web/src/shared/api/client.ts` 전체 교체:

```typescript
export const API_BASE = '/api'

let getToken: (() => string | null) = () => null
let onTokenRefreshed: ((token: string) => void) | null = null
let onAuthExpired: (() => void) | null = null

export function setTokenGetter(fn: () => string | null) {
  getToken = fn
}

export function setTokenRefreshCallback(fn: (token: string) => void) {
  onTokenRefreshed = fn
}

export function setAuthExpiredCallback(fn: () => void) {
  onAuthExpired = fn
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

let refreshPromise: Promise<string | null> | null = null

async function tryRefresh(): Promise<string | null> {
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE}/auth/refresh`, { method: 'POST' })
      if (!res.ok) {
        if (onAuthExpired) onAuthExpired()
        return null
      }
      const { token } = await res.json()
      if (token && onTokenRefreshed) onTokenRefreshed(token)
      return token as string
    } catch {
      if (onAuthExpired) onAuthExpired()
      return null
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

// Export for proactive refresh use
export { tryRefresh }

export async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const res = await fetch(url, { ...init, headers })

  if (res.status === 401 && !url.includes('/api/auth/')) {
    const newToken = await tryRefresh()
    if (newToken) {
      const retryHeaders = new Headers(init?.headers)
      retryHeaders.set('Authorization', `Bearer ${newToken}`)
      const retry = await fetch(url, { ...init, headers: retryHeaders })
      if (!retry.ok) {
        const text = await retry.text().catch(() => retry.statusText)
        throw new ApiError(retry.status, text || retry.statusText)
      }
      if (retry.status === 204 || retry.headers.get('content-length') === '0') {
        return undefined as T
      }
      return retry.json()
    }
  }

  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new ApiError(res.status, text || res.statusText)
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T
  }
  return res.json()
}
```

**Step 2: auth store — 선제적 갱신 타이머 + visibilitychange + graceful 로그아웃**

`web/src/entities/auth/store.ts` 전체 교체:

```typescript
import { create } from 'zustand'
import { fetchMe, refreshToken, logout as apiLogout, type User } from '@/shared/api/auth'
import {
  setTokenGetter,
  setTokenRefreshCallback,
  setAuthExpiredCallback,
  tryRefresh,
} from '@/shared/api/client'

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

let refreshTimer: ReturnType<typeof setTimeout> | null = null

/** Decode JWT payload without verification (for reading exp). */
function decodeTokenExp(token: string): number | null {
  try {
    const payload = token.split('.')[1]
    const decoded = JSON.parse(atob(payload))
    return decoded.exp ?? null
  } catch {
    return null
  }
}

function scheduleProactiveRefresh(token: string) {
  if (refreshTimer) clearTimeout(refreshTimer)

  const exp = decodeTokenExp(token)
  if (!exp) return

  // Refresh 5 minutes before expiry
  const msUntilRefresh = (exp * 1000) - Date.now() - (5 * 60 * 1000)
  if (msUntilRefresh <= 0) {
    // Already near expiry, refresh now
    tryRefresh()
    return
  }

  refreshTimer = setTimeout(() => {
    tryRefresh()
  }, msUntilRefresh)
}

function isTokenExpiredOrNearExpiry(token: string): boolean {
  const exp = decodeTokenExp(token)
  if (!exp) return true
  // Consider expired if within 5 minutes of expiry
  return (exp * 1000) - Date.now() < 5 * 60 * 1000
}

export const useAuthStore = create<AuthState>((set, get) => {
  setTokenGetter(() => get().token)
  setTokenRefreshCallback((token) => {
    set({ token })
    scheduleProactiveRefresh(token)
  })
  setAuthExpiredCallback(() => {
    set({ token: null, user: null })
  })

  // Visibility change handler — refresh on tab return
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState !== 'visible') return
      const { token, user } = get()
      if (!token || !user) return
      if (isTokenExpiredOrNearExpiry(token)) {
        tryRefresh()
      }
    })
  }

  return {
    user: null,
    token: null,
    loading: true,
    initialized: false,

    setToken: (token) => {
      set({ token })
      scheduleProactiveRefresh(token)
    },

    init: async () => {
      const params = new URLSearchParams(window.location.search)
      const urlToken = params.get('token')
      if (urlToken) {
        set({ token: urlToken })
        scheduleProactiveRefresh(urlToken)
        window.history.replaceState({}, '', window.location.pathname)
      }

      if (!get().token) {
        const ok = await get().refresh()
        if (!ok) {
          set({ loading: false, initialized: true })
          return
        }
      }

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
        scheduleProactiveRefresh(token)
        return true
      } catch {
        return false
      }
    },

    logout: async () => {
      if (refreshTimer) clearTimeout(refreshTimer)
      try { await apiLogout() } catch { /* ignore */ }
      set({ token: null, user: null })
    },
  }
})
```

**Step 3: 프론트엔드 빌드 확인**

Run: `cd web && npx tsc -b`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/shared/api/client.ts web/src/entities/auth/store.ts
git commit -m "feat: proactive token refresh, tab visibility detection, graceful auth expiry"
```

---

### Task 9: 통합 테스트 + 전체 빌드

**Step 1: Go 전체 테스트**

Run: `go test ./... -v -race`
Expected: ALL PASS

**Step 2: 프론트엔드 타입 체크**

Run: `cd web && npx tsc -b`
Expected: No errors

**Step 3: 프론트엔드 lint**

Run: `cd web && npm run lint`
Expected: No errors

**Step 4: 전체 빌드**

Run: `make build`
Expected: BUILD SUCCESS

**Step 5: Commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: resolve integration issues from token management redesign"
```

---

### Task 10: data/jwt_secret을 .gitignore에 추가

**Files:**
- Modify: `.gitignore`

**Step 1: .gitignore에 추가**

```
data/jwt_secret
```

**Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: gitignore jwt_secret file"
```
