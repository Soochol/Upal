# User Authentication & Multi-Tenancy Design

## Overview

Upal에 사용자 인증(Google/GitHub OAuth)과 멀티테넌트 데이터 격리를 추가한다. 각 사용자는 자신만의 워크플로우, 파이프라인, AI 프로바이더, MCP 서버 등을 관리한다.

## 결정 사항

- **인증 방식**: Google + GitHub OAuth2 (이메일/비밀번호 없음)
- **구현 방식**: Go 자체 구현 (`golang.org/x/oauth2` + JWT)
- **세션**: JWT access token (30분) + refresh token (7일, httpOnly cookie)

## 1. config.yaml 설정 분리

### 시스템 설정 (config.yaml 유지)

| 항목 | 이유 |
|------|------|
| `server` (host, port) | 서버 기동 시점에 필요 |
| `database.url` | DB 연결 전제 조건 |

### 사용자별 설정 (DB 이동)

| 항목 | 이유 |
|------|------|
| `providers` (AI 프로바이더) | 각 사용자가 자신의 API 키/프로바이더 관리 |
| `mcp_servers` | 각 사용자가 자신의 외부 도구 연결 관리 |

config.yaml의 `providers`와 `mcp_servers`는 **시스템 기본값(fallback)**으로 남긴다. 사용자가 자신의 프로바이더를 등록하면 우선 사용, 없으면 시스템 기본값 사용.

### 변경 후 config.yaml 형태

```yaml
server:
  host: "0.0.0.0"
  port: 8081

database:
  url: ""  # Required for multi-tenant mode

# OAuth providers
auth:
  google:
    client_id: ""
    client_secret: ""
  github:
    client_id: ""
    client_secret: ""
  jwt_secret: ""  # HMAC signing key for JWT tokens

# System-default providers (fallback when user has no custom providers)
providers:
  ollama:
    type: openai
    url: "http://localhost:11434/v1"
    api_key: ""
  # ... etc

# System-default MCP servers (fallback)
mcp_servers: {}
```

## 2. DB 스키마

### 신규 테이블: `users`

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL DEFAULT '',
    avatar_url    TEXT NOT NULL DEFAULT '',
    oauth_provider TEXT NOT NULL,  -- 'google' | 'github'
    oauth_id      TEXT NOT NULL,   -- provider-specific unique ID
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(oauth_provider, oauth_id)
);
```

### 신규 테이블: `mcp_servers`

```sql
CREATE TABLE mcp_servers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 기존 테이블 변경: `user_id` 추가

다음 테이블에 `user_id UUID NOT NULL REFERENCES users(id)` 컬럼 추가:

- `workflows`
- `pipelines`
- `runs`
- `pipeline_runs`
- `schedules`
- `triggers`
- `ai_providers`
- `connections`
- `content_sessions`
- `published_content`
- `assets`
- `sessions` (workflow execution sessions)
- `events`
- `source_fetches`
- `llm_analyses`
- `surge_events`
- `workflow_results`

## 3. 인증 흐름

```
[로그인 페이지]
  → "Google로 로그인" / "GitHub로 로그인" 클릭
  → GET /api/auth/login/{provider}  (OAuth 리다이렉트 URL 생성)
  → 외부 OAuth 동의 화면
  → GET /api/auth/callback/{provider}  (콜백 처리)
    → 사용자 조회 또는 자동 생성
    → JWT access token + refresh token 발급
    → 프론트엔드로 리다이렉트 (access token 전달)
```

### API 엔드포인트

| Method | Path | 설명 |
|--------|------|------|
| GET | `/api/auth/login/{provider}` | OAuth 시작 (provider: google, github) |
| GET | `/api/auth/callback/{provider}` | OAuth 콜백 처리 |
| POST | `/api/auth/refresh` | refresh token으로 access token 갱신 |
| POST | `/api/auth/logout` | refresh token 무효화 |
| GET | `/api/auth/me` | 현재 로그인 사용자 정보 |

### JWT 구조

```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "name": "User Name",
  "exp": 1234567890,
  "iat": 1234567890
}
```

### 미들웨어

- Chi 미들웨어로 `Authorization: Bearer {token}` 검증
- 검증 성공 시 `context.WithUserID(ctx, userID)` 설정 (기존 스텁 활용)
- `/api/auth/*` 경로는 미들웨어에서 제외
- 미인증 요청 → 401 Unauthorized

## 4. 프론트엔드

### 로그인 페이지 (`/login`)

- Google / GitHub 로그인 버튼 2개
- 미인증 상태에서 모든 경로 → `/login` 리다이렉트
- 로그인 성공 후 원래 요청 경로로 복귀

### 인증 상태 관리

- access token을 메모리(zustand/context)에 보관
- refresh token은 httpOnly cookie (서버가 설정)
- API 요청 시 `Authorization: Bearer {token}` 헤더 첨부
- 401 응답 시 자동 refresh 시도 → 실패 시 로그인 페이지로

### UI 변경

- 사이드바 하단에 사용자 아바타 + 이름
- 아바타 클릭 → 드롭다운 메뉴 (로그아웃, 설정)

## 5. Repository / Service 레이어 변경

### 신규

- `UserRepository` (memory + persistent 패턴)
- `MCPServerRepository` (memory + persistent 패턴)
- `AuthService` (OAuth 처리, JWT 발급/검증, 사용자 관리)

### 기존 변경

- 모든 Repository 인터페이스의 `List`, `Get`, `Create` 등에 `userID` 파라미터 추가
- 기존 SQL 쿼리에 `WHERE user_id = $N` 조건 추가
- Service 레이어는 `context`에서 `userID` 추출하여 Repository에 전달
- Memory 구현도 `userID` 기반 필터링 추가

### 프로바이더 해석 우선순위

1. 사용자의 DB 등록 프로바이더
2. config.yaml 시스템 기본 프로바이더 (fallback)

## 6. 마이그레이션 전략

- 기존 데이터는 "default" 사용자에게 할당
- 첫 번째 OAuth 로그인 사용자가 기존 "default" 데이터를 인계받을 수 있는 옵션 제공
- DB 필수화 (멀티테넌트 모드에서는 in-memory 전용 불가)
