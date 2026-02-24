# Skill Prompt Extraction Design

## Goal

하드코딩된 LLM 시스템 프롬프트 4개를 `internal/skills/prompts/` md 파일로 분리하고, skills 접근 인터페이스를 통일한다.

## Current State

12개 LLM 호출 중 4개가 Go 코드에 시스템 프롬프트를 하드코딩:

| # | 파일 | 하드코딩 변수/위치 | 용도 |
|---|------|------------------|------|
| 1 | `services/content_collector.go:302-318` | 인라인 string | 콘텐츠 분석 |
| 2 | `api/configure.go:52-69` | `configureBasePrompt` var | 노드 설정 |
| 3 | `api/name.go:16-22` | `nameSystemPrompt` var | 워크플로우 이름 제안 |
| 4 | `output/formatter.go:68-81` | `BaseLayoutConstraints` var | HTML 레이아웃 |

### 인터페이스 불일치

- `skills.Provider`: `Get()` only — `api/Server`가 사용
- `generate.skillProvider` (로컬): `Get()` + `GetPrompt()` — `generate/Generator`가 사용
- `services/ContentCollector`: skills 접근 없음
- `output/formatter.go`: skills 접근 없음

## Changes

### 1. Interface Unification

`skills.Provider`에 `GetPrompt()` 추가, `generate.skillProvider` 제거:

```go
// skills/provider.go
type Provider interface {
    Get(name string) string
    GetPrompt(name string) string
}
```

`generate/generate.go`에서 로컬 `skillProvider` 인터페이스 삭제 → `skills.Provider` import.

### 2. Skill Files (4 new files)

프롬프트 텍스트는 기존 내용을 그대로 이동 (내용 변경 없음).

| File | Source |
|------|--------|
| `prompts/content-analyze.md` | `content_collector.go:302-318` |
| `prompts/node-configure.md` | `configure.go:52-69` |
| `prompts/workflow-name.md` | `name.go:16-22` |
| `prompts/html-layout.md` | `formatter.go:68-81` |

### 3. Go Code Changes

#### `skills/provider.go`
- `GetPrompt(name string) string` 메서드 추가

#### `generate/generate.go`
- 로컬 `skillProvider` interface 삭제
- `skills.Provider` import 사용
- Generator struct의 `skills` 필드 타입: `skills.Provider`

#### `services/content_collector.go`
- `ContentCollector` struct에 `skills skills.Provider` 필드 추가
- `NewContentCollector()` 파라미터에 `skills skills.Provider` 추가
- `buildAnalysisPrompt()` → 시스템 프롬프트를 `c.skills.GetPrompt("content-analyze")`로 교체
  - `buildAnalysisPrompt`의 시그니처: `systemPromptBase string` 파라미터 추가
  - 또는 `ContentCollector`의 메서드로 변환

#### `api/configure.go`
- `configureBasePrompt` var 삭제
- `s.skills.GetPrompt("node-configure")` 사용

#### `api/name.go`
- `nameSystemPrompt` var 삭제
- `s.skills.GetPrompt("workflow-name")` 사용

#### `output/formatter.go`
- `BaseLayoutConstraints` var 삭제
- `NewFormatter` 시그니처에 `basePrompt string` 파라미터 추가
- 호출자(`agents/output_builder.go`)가 전달

#### `agents/output_builder.go`
- `BuildDeps` struct에 `Skills skills.Provider` 필드 추가 (또는 `BaseLayoutPrompt string`)
- `output.NewFormatter` 호출 시 base prompt 전달

#### `agents/registry.go`
- `BuildDeps` struct에 `Skills skills.Provider` 추가

#### `cmd/upal/main.go`
- `NewContentCollector()`에 `skillReg` 전달
- `BuildDeps` 생성 시 `Skills: skillReg` 추가

#### `api/generate_test.go`
- `noopSkills`에 `GetPrompt()` 이미 존재 — 변경 불필요

## Decision: output 패키지의 skills 접근

`output.NewFormatter`에 `basePrompt string` 파라미터를 추가하는 방식 (Option A) 채택.
- output 패키지가 skills에 직접 의존하지 않음
- `BuildDeps`에 `Skills` 추가 → `output_builder.go`에서 `skills.GetPrompt("html-layout")` 호출 후 전달

## Not In Scope

- 프롬프트 텍스트 내용 변경 (rich persona 업그레이드 등)
- 기존 skill 파일로 관리되는 프롬프트 수정
- generate 패키지의 nil 체크 패턴 통일 (별도 작업)
