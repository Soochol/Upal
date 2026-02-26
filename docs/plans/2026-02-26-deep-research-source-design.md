# Deep Research Source — Design Document

## Problem

파이프라인 소스에 LLM 기반 리서치 기능이 빠져 있다. 백엔드에 `researchFetcher`와 `stage-research.md` 스킬이 이미 구현되어 있지만, 파이프라인 소스 타입으로 노출되지 않아 사용자가 소스 설정 UI에서 리서치를 추가할 수 없다. 또한 LLM 별 web_search 호환 여부를 확인할 방법이 없다.

## Solution

파이프라인 소스 시스템에 `research` 타입을 풀스택 통합한다.

### 변경 범위

| 레이어 | 현재 상태 | 추가할 것 |
|--------|----------|----------|
| `CollectSource` (Go) | Topic, Model, Depth 필드 있음 | 변경 없음 |
| `researchFetcher` (Go) | 완전 구현됨 | stage_collect.go에 등록 |
| `PipelineSource` (Go) | research 필드 없음 | Topic, Depth, Model 필드 추가 |
| `mapPipelineSources` | research case 없음 | research → CollectSource 매핑 |
| `PipelineSourceType` (TS) | 'research' 없음 | 타입 추가 + topic, depth, model 필드 |
| `AddSourceModal` (React) | 2개 카테고리 | 3번째 "AI Research" 카테고리 + 설정 UI |
| `stage-collect.md` 스킬 | research 문서 없음 | research 소스 타입 문서 추가 |
| `stage-research.md` 스킬 | light/deep 프롬프트 | 보강 (변경 최소) |

## Data Model

### PipelineSource (Go — 추가 필드)

```go
Topic string `json:"topic,omitempty"` // research: subject to investigate
Depth string `json:"depth,omitempty"` // research: "light" | "deep"
Model string `json:"model,omitempty"` // research: LLM model ID ("provider/model")
```

### PipelineSource (TypeScript — 추가 필드)

```typescript
export type PipelineSourceType =
  | 'rss' | 'hn' | 'reddit' | 'google_trends' | 'social' | 'http'
  | 'research'  // NEW

export type PipelineSource = {
  // ... existing fields ...
  topic?: string       // research
  depth?: 'light' | 'deep'  // research
  model?: string       // research: "provider/model"
}
```

### Source Mapping

```go
case "research":
    cs.Type = "research"
    cs.Topic = ps.Topic
    cs.Model = ps.Model
    cs.Depth = ps.Depth  // default "deep"
```

기본 depth를 `"deep"`으로 설정 (이전 설계에선 "light"였으나, 소스로 추가하는 의도 자체가 깊은 리서치).

## Frontend UI

### AddSourceModal — 3번째 카테고리

```
DATA SOURCES     RSS · Hacker News · HTTP
SIGNALS          Reddit · Google Trends · Social
AI RESEARCH      Deep Research
```

### 설정 화면

- **Topic** (text input, required): 리서치 주제
- **Depth** (radio: Light / Deep, default Deep)
- **Model** (ModelSelector, web_search 지원 모델만 필터링)
  - `ModelInfo.supportsTools === true` + Ollama 제외
  - 기존 `ModelSelector` 컴포넌트에 `models` prop으로 필터된 목록 전달

## LLM 호환성

프론트엔드에서 모델 목록을 가져올 때 `supportsTools` 필드(이미 API 응답에 포함됨)로 필터링.

| Provider | web_search 지원 | Research 사용 가능 |
|----------|----------------|-------------------|
| Anthropic | O | O |
| Gemini | O | O |
| OpenAI | O | O |
| Ollama | X | X |
| Claude Code | X | X |

추가로 `NativeToolProvider` 인터페이스 미구현 모델은 `researchFetcher`에서 런타임 에러 반환 (기존 로직).

## Skills

### stage-collect.md 업데이트

research 소스 타입 문서 추가:
- `research`: `topic` (required), `depth` ("light" | "deep"), `model` (provider/model). LLM 기반 웹 리서치. 모델이 web_search를 지원해야 함.

### stage-research.md

현재 프롬프트 유지. 필요 시 deep 모드 프롬프트 보강.

## Non-Goals

- max_searches UI 노출 (기본값 10 사용)
- 리서치 진행 상황 실시간 표시 (이미 ResearchProgress 지원, 별도 UI 작업 필요 시 후속)
