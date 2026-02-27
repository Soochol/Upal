# Token Management Redesign — DB 기반 Refresh Token + 선제적 갱신

## 문제

1. Access token 만료(30분) 후 첫 요청이 반드시 401 실패 → 사용자에게 에러 노출
2. Refresh token이 서버에 저장되지 않아 revoke 불가 — 로그아웃해도 탈취된 토큰 유효
3. JWTSecret 미설정 시 서버 재시작마다 모든 토큰 무효화
4. Refresh 실패 시 로그인 리다이렉트 없이 cryptic 에러만 노출

## 설계

### 1. DB: `refresh_tokens` 테이블

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          TEXT PRIMARY KEY,                        -- UUID
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,                           -- SHA-256 of JWT
    device_info TEXT NOT NULL DEFAULT '',                -- User-Agent 등
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,                            -- NULL = 유효
    replaced_by TEXT REFERENCES refresh_tokens(id)       -- rotation 추적
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

### 2. 백엔드: Token Lifecycle

#### 발급 (로그인/OAuth callback)
- Access token JWT 발급 (TTL: 1시간)
- Refresh token JWT 발급 (TTL: 30일)
- JWT claims에 `jti` (token ID) 추가
- `refresh_tokens` 테이블에 `(id=jti, user_id, token_hash, device_info, expires_at)` INSERT
- Refresh token을 HttpOnly/SameSite=Lax 쿠키로 설정

#### 갱신 (`POST /api/auth/refresh`)
1. 쿠키에서 refresh token JWT 추출
2. JWT 서명 검증 + 만료 확인
3. `jti`로 DB 조회 → `revoked_at IS NULL` 확인
4. **이미 revoke된 토큰이면** → token reuse 감지 → 해당 유저의 전체 refresh token revoke (family revocation) → 401
5. 현재 토큰 revoke (`revoked_at = NOW()`)
6. 새 access + refresh token 쌍 발급
7. 새 refresh token DB 저장 (이전 토큰의 `replaced_by` = 새 토큰 ID)
8. 새 refresh token 쿠키 설정

#### 로그아웃 (`POST /api/auth/logout`)
- 쿠키의 refresh token `jti`로 DB에서 해당 토큰 revoke
- 쿠키 삭제

#### 전체 세션 종료 (향후 확장)
- `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
- "다른 기기에서 로그아웃" 기능의 기반

#### 만료 토큰 정리
- `AuthService` 초기화 시 goroutine으로 24시간 주기 cleanup
- `DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '1 day'`

### 3. 백엔드: TTL 변경

| 항목 | 현재 | 변경 |
|------|------|------|
| Access token TTL | 30분 | 1시간 |
| Refresh token TTL | 7일 | 30일 |

### 4. 백엔드: JWTSecret 영속화

- `config.yaml`에 `jwt_secret`이 비어있으면 랜덤 생성 후 **`data/jwt_secret` 파일에 저장**
- 다음 시작 시 파일에서 읽음
- 이미 설정된 경우 파일 무시 (명시적 설정 우선)

### 5. 프론트엔드: 선제적 토큰 갱신

#### 타이머 기반 갱신
- Access token 발급 시 JWT에서 `exp` 파싱 (base64 decode, DB 불필요)
- 만료 5분 전에 자동 refresh 실행
- 갱신 성공 시 타이머 재설정

#### 탭 복귀 감지
- `document.addEventListener('visibilitychange', ...)` 등록
- `document.visibilityState === 'visible'`일 때:
  - 토큰 만료 여부 확인 (로컬 `exp` 비교)
  - 만료됐거나 5분 이내면 즉시 refresh

#### Refresh 실패 시 UX
- `tryRefresh()`가 `null` 반환하면:
  - `authStore`의 `token`/`user`를 `null`로 설정
  - 이로 인해 `AuthGuard`가 자동으로 `/login`으로 리다이렉트
  - toast로 "세션이 만료되었습니다. 다시 로그인해주세요." 표시

### 6. 프론트엔드: client.ts 개선

- `tryRefresh()` 실패 시 authStore 초기화 로직 추가
- 401 auto-retry 유지 (선제 갱신이 실패한 edge case 대비)
- 갱신 성공 시 `scheduleRefresh()` 호출하여 다음 타이머 설정

## 변경 파일 목록

### 백엔드
| 파일 | 변경 내용 |
|------|-----------|
| `internal/db/db.go` | `refresh_tokens` 테이블 마이그레이션 추가 |
| `internal/db/refresh_token.go` | 신규 — refresh token CRUD (Create, FindByHash, Revoke, RevokeAllForUser, Cleanup) |
| `internal/services/auth.go` | TTL 변경, `jti` claim 추가, refresh token DB 연동, token rotation 로직, cleanup goroutine |
| `internal/api/auth.go` | `authRefresh` — DB 기반 검증 + rotation. `authLogout` — DB revoke. JWTSecret 파일 영속화 |
| `internal/config/config.go` | JWTSecret 파일 fallback 로직 (또는 `cmd/upal/main.go`에서 처리) |

### 프론트엔드
| 파일 | 변경 내용 |
|------|-----------|
| `web/src/shared/api/client.ts` | `tryRefresh()` 실패 시 authStore 초기화, `scheduleRefresh()` 함수 추가 |
| `web/src/entities/auth/store.ts` | 선제적 갱신 타이머, `visibilitychange` 리스너, refresh 실패 시 로그아웃 처리 |

## 보안 모델

| 위협 | 방어 |
|------|------|
| XSS로 refresh token 탈취 | HttpOnly 쿠키 — JS 접근 불가 |
| CSRF로 refresh 요청 위조 | SameSite=Lax |
| Refresh token 탈취 후 사용 | Token rotation — 사용된 토큰은 즉시 revoke. 재사용 감지 시 전체 family revoke |
| 로그아웃 후 토큰 유효 | DB에서 즉시 revoke — JWT 서명 유효해도 DB에서 거부 |
| 서버 재시작 시 토큰 무효화 | JWTSecret 파일 영속화 |
| 계정 침해 | 전체 refresh token 일괄 revoke 가능 |
