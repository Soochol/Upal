# Session Owns All Settings — Pipeline as Group Only

**Date:** 2026-02-26

## Problem

Pipeline detail 페이지에서 설정(소스, 스케줄, 모델, 워크플로우, 컨텍스트)이 파이프라인 단위로 관리되고 있다. 실제로는 세션이 template 역할을 하며 N개의 instance를 생성하는 구조이므로, 설정의 소유권이 세션에 있어야 한다.

## Design

### Core Principle

- **Pipeline** = 세션들의 그룹/폴더. 이름, 설명, stages, 썸네일만 보유
- **Session** = 모든 실행 설정의 소유자 (소스, 스케줄, 모델, 워크플로우, 컨텍스트)
- Template 세션은 active/inactive 토글만. Instance는 inbox에서 lifecycle 관리

### 1. Pipeline Model — 필드 제거

`Pipeline` struct에서 제거:
- `Sources []PipelineSource`
- `Schedule string`
- `Model string`
- `Workflows []PipelineWorkflow`
- `Context *PipelineContext`

남는 필드: ID, Name, Description, Stages, ThumbnailSVG, timestamps

### 2. ContentSession — 설정 소유

이미 존재하는 필드 활성화:
- `Sources`, `Schedule`, `Model`, `Workflows`, `Context`
- `IsTemplate`, `ParentSessionID`

새 세션 생성 시 빈 상태로 시작, 사용자가 직접 설정.

### 3. DB Migration

```sql
-- pipelines 테이블에서 설정 컬럼 제거
ALTER TABLE pipelines DROP COLUMN IF EXISTS sources;
ALTER TABLE pipelines DROP COLUMN IF EXISTS schedule;
ALTER TABLE pipelines DROP COLUMN IF EXISTS model;
ALTER TABLE pipelines DROP COLUMN IF EXISTS context;

-- content_sessions 테이블에 설정 컬럼 추가 (없는 것만)
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS schedule TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS workflows JSONB NOT NULL DEFAULT '[]';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS context JSONB;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS parent_session_id TEXT;
```

### 4. Backend Service Changes

- **Collector**: `pipeline.Sources` → `session.Sources`
- **Analyzer**: `pipeline.Model` → `session.Model`
- **Prompt builder**: `pipeline.Context/Workflows` → `session.Context/Workflows`
- **Scheduler**: 세션 기준으로 schedule/sources 확인

### 5. Frontend Changes

- `PipelineSettingsPanel` → 세션 설정 패널로 교체
- 세션 선택 시 해당 세션의 설정 편집 (소스, 스케줄, 모델, 워크플로우, 컨텍스트)
- AI Assistant 채팅도 세션 컨텍스트로 이동
- `Pipeline` 타입에서 5개 필드 제거
- Template 세션은 Active/Inactive badge만 표시

### 6. Impact

| Layer | Files |
|-------|-------|
| Domain model | `internal/upal/pipeline.go` |
| DB | `internal/db/db.go`, `internal/db/pipeline.go`, `internal/db/content.go` |
| Services | `content_collector.go`, `content_session_service.go`, `scheduler/dispatch.go` |
| API | `internal/api/pipelines.go` (cleanup) |
| Frontend types | `pipeline/types.ts`, `pipeline/api/index.ts` |
| Frontend UI | `PipelineSettingsPanel` → Session settings, `SessionListPanel` status badge |
