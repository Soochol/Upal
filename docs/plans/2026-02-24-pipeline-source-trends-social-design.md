# Pipeline Source: Google Trends + Social Trends Design

**Date**: 2026-02-24
**Status**: Approved

## Problem

파이프라인 소스 중 Google Trends와 Twitter/X는 프론트엔드 UI만 존재하고 백엔드에서 `skip` 처리됨. 사용자가 소스를 추가해도 데이터가 수집되지 않고, 에러 표시도 없이 조용히 무시됨.

## Solution Overview

- **Google Trends**: 공식 RSS 피드 활용 → 기존 `rssFetcher` 재사용
- **Twitter/X → Social Trends**: Bluesky + Mastodon 공개 API → 새 `socialFetcher` 구현

둘 다 무료, 인증 불필요, 법적/기술적 리스크 제로.

## Design

### 1. Google Trends Source

**데이터 흐름**:
```
사용자 설정: keywords=["AI", "React"], geo="US"
  ↓
mapPipelineSources: → CollectSource{ Type: "rss", URL: "https://trends.google.com/trending/rss?geo=US" }
  ↓
기존 rssFetcher로 수집 → 일간 트렌딩 키워드 + 검색량 + 관련 뉴스
  ↓
LLM 분석 단계에서 사용자 keywords로 필터링/관련성 평가
```

**사용자 입력 필드**:
- `keywords[]` — LLM 분석 시 관심 키워드 필터로 활용
- `limit` — 수집 항목 수
- `geo` — 국가 코드 (기본값 "US", 드롭다운으로 주요 국가 제공)

**구현**: `mapPipelineSources`에서 `case "google_trends"` → RSS URL 변환. 새 fetcher 불필요.

### 2. Social Trends Source (replaces Twitter/X)

**데이터 흐름**:
```
사용자 설정: keywords=["AI", "startup"], limit=20
  ↓
mapPipelineSources: → CollectSource{ Type: "social", Keywords: ["AI", "startup"], Limit: 20 }
  ↓
socialFetcher가 두 API 병렬 호출:
  1) Bluesky: getTrendingTopics + searchPosts?q=keyword
  2) Mastodon: /api/v1/trends/tags + /api/v1/trends/links
  ↓
결과 병합 → 통합 SourceItem 리스트 → LLM 분석
```

**socialFetcher 동작**:
- 트렌딩: Bluesky `getTrendingTopics` + Mastodon `trends/tags` → 현재 트렌딩 토픽
- 키워드 검색: Bluesky `searchPosts?q={keyword}` → 키워드별 최신 포스트
- 계정 피드: Bluesky `getAuthorFeed` + Mastodon `/accounts/:id/statuses` → 팔로우 계정의 최신 글
- 세 종류의 결과를 하나의 포맷된 텍스트로 병합

**Bluesky 엔드포인트** (공개, 인증 불필요):
- `GET https://public.api.bsky.app/xrpc/app.bsky.unspecced.getTrendingTopics`
- `GET https://public.api.bsky.app/xrpc/app.bsky.feed.searchPosts?q=keyword`
- `GET https://public.api.bsky.app/xrpc/app.bsky.feed.getAuthorFeed?actor=handle&limit=N`

**Mastodon 엔드포인트** (공개, 인증 불필요):
- `GET https://mastodon.social/api/v1/trends/tags`
- `GET https://mastodon.social/api/v1/trends/links`
- `GET https://mastodon.social/api/v1/accounts/lookup?acct=username` → account ID 조회
- `GET https://mastodon.social/api/v1/accounts/:id/statuses?limit=N` → 계정 포스트

**계정 핸들 형식**:
- Bluesky: `alice.bsky.social` (`.bsky.social` 또는 커스텀 도메인)
- Mastodon: `user@mastodon.social` (`@` 포함 시 Mastodon, 아니면 Bluesky로 판별)

**프론트엔드 변경**:
- 소스 라벨: "X / Twitter" → "Social Trends"
- 설명: "Bluesky & Mastodon trends + feeds"
- 아이콘: Twitter → Globe 또는 TrendingUp
- 입력 필드: keywords + accounts + limit

### 3. Type System Changes

**Frontend `PipelineSourceType`**:
```
기존: 'rss' | 'hn' | 'reddit' | 'google_trends' | 'twitter' | 'http'
변경: 'rss' | 'hn' | 'reddit' | 'google_trends' | 'social' | 'http'
```

`'twitter'` → `'social'`로 rename. 기존 `"twitter"` 데이터 호환을 위해 `mapPipelineSources`에서 fallback 처리.

**Backend `CollectSource` 추가 필드**:
```go
Keywords []string  // social fetcher: 검색 키워드
Accounts []string  // social fetcher: 팔로우 계정 핸들
Geo      string    // google_trends RSS: 국가 코드
```

**Backend `PipelineSource` 추가 필드**:
```go
Accounts []string `json:"accounts,omitempty"` // social: 팔로우 계정 핸들
Geo      string   `json:"geo,omitempty"`      // google_trends: 국가 코드
```

### 4. No Changes Required

- `SourceTypeBadge` — `social`은 기존 `source_type: "signal"` 유지, 배지 동일
- DB 스키마 — JSONB 저장이므로 마이그레이션 불필요
- API 엔드포인트 — 기존 CRUD 그대로 사용

## Alternatives Considered

- **Twitter 스크래퍼 (imperatrona/twitter-scraper)**: Go 네이티브이나 계정 쿠키 필요, 수주 단위 장애, 계정 정지 리스크, ToS 위반
- **하이브리드 (Bluesky/Mastodon 기본 + Twitter Connection 시 직접 접근)**: 구현 범위 넓고 두 경로 유지보수 필요
- **유료 API (TwitterAPI.io, SerpApi)**: 월 $45-75 비용 발생

접근법 A (RSS + Bluesky/Mastodon)가 안정성, 비용, 유지보수 면에서 최적.
