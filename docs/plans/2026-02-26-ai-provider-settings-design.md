# AI Provider Settings Design

## Problem

현재 AI provider 설정이 `config.yaml` 파일에만 존재하여 UI에서 동적 관리가 불가능. 사용자가 Settings에서 API를 등록/관리하고, 카테고리별 기본 모델을 선택할 수 있어야 함.

## Design

### Domain Model

```go
// internal/upal/ai_provider.go

type AIProviderCategory string
const (
    CategoryLLM   AIProviderCategory = "llm"
    CategoryTTS   AIProviderCategory = "tts"
    CategoryImage AIProviderCategory = "image"
    CategoryVideo AIProviderCategory = "video"
)

type AIProvider struct {
    ID        string             // "aip_xxx" auto-generated
    Name      string             // user label, e.g. "My Anthropic"
    Category  AIProviderCategory // llm, tts, image, video
    Type      string             // provider type: "anthropic", "openai", "gemini", "openai-tts", etc.
    APIKey    string             // encrypted at rest
    IsDefault bool               // default for this category (one per category)
}

type AIProviderSafe struct {
    ID        string             // same as AIProvider but no APIKey
    Name      string
    Category  AIProviderCategory
    Type      string
    IsDefault bool
}
```

### Category → Provider Type Mapping

| Category | Available Provider Types |
|----------|------------------------|
| llm      | anthropic, openai, gemini, ollama |
| tts      | openai-tts |
| image    | gemini-image, zimage |
| video    | (future) |

이 매핑은 기존 `internal/model/catalog.go`의 `modelCategoryByType`을 확장하여 활용.

### DB Schema

```sql
CREATE TABLE IF NOT EXISTS ai_providers (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    category   TEXT NOT NULL,  -- 'llm', 'tts', 'image', 'video'
    type       TEXT NOT NULL,  -- provider type
    api_key    TEXT NOT NULL DEFAULT '',  -- encrypted
    is_default BOOLEAN NOT NULL DEFAULT FALSE
);
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST   | /api/ai-providers | 새 provider 등록 |
| GET    | /api/ai-providers | 전체 목록 (safe view) |
| PUT    | /api/ai-providers/{id} | 수정 |
| DELETE | /api/ai-providers/{id} | 삭제 |
| PUT    | /api/ai-providers/{id}/default | 해당 카테고리 기본으로 설정 |

### Backend Layer Structure

기존 Connection 패턴과 동일한 레이어 구조:

1. **Domain** (`internal/upal/ai_provider.go`): 타입 + Safe 변환
2. **Port** (`internal/upal/ports/ai_provider.go`): Repository 인터페이스
3. **Repository** (`internal/repository/ai_provider_*.go`): Memory + Persistent
4. **DB** (`internal/db/ai_provider.go`): SQL queries
5. **Service** (`internal/services/ai_provider.go`): 암호화 + 비즈니스 로직
6. **API** (`internal/api/ai_providers.go`): HTTP handlers

### 기본 모델 선택 로직

- 카테고리당 is_default=true인 provider 최대 1개
- 새 provider를 default로 설정하면 같은 카테고리의 기존 default 해제
- default provider의 첫 번째 모델이 해당 카테고리의 기본 모델

### GET /api/models 변경

현재: `config.yaml`의 providerConfigs 기반으로 모델 반환
변경: DB의 등록된 AI provider 기반으로 모델 반환

```
등록된 providers → ProviderConfig 변환 → 기존 AllStaticModels() 호출
```

config.yaml provider 설정은 fallback으로 유지 (DB에 아무것도 없을 때).

### 기본 모델 적용 위치

1. **워크플로우 생성/설정 LLM**: `Server.defaultGenerateModel`을 DB 기본 LLM으로 교체
2. **새 Agent 노드**: 프론트엔드에서 기본 LLM 모델 ID를 받아 초기값 설정
3. **워크플로우 노드 모델 선택**: 모든 카테고리의 등록된 모델 표시 (현재와 동일)

### Frontend

#### Settings 페이지 확장

```
Settings
├── Appearance (기존 테마)
└── AI Providers (신규)
    ├── LLM
    │   ├── Anthropic - claude-sonnet... ⭐ [삭제]
    │   ├── OpenAI - gpt-4o             [삭제]
    │   └── [+ Add LLM Provider]
    ├── TTS
    │   └── [+ Add TTS Provider]
    ├── Image
    │   └── [+ Add Image Provider]
    └── Video
        └── [+ Add Video Provider]
```

#### 추가 모달

- Category는 섹션에서 자동 결정
- Provider type 선택 (카테고리별 가용 타입 필터)
- API key 입력
- 이름 입력 (선택, 없으면 provider type으로 자동)

#### 기본 모델 API 반영

- `GET /api/ai-providers` 응답에 각 카테고리의 default 정보 포함
- `GET /api/models`에 `isDefault` 필드 추가하여 프론트엔드에서 기본 모델 인식
- `useModels()` 캐시 무효화: provider 추가/삭제 시 모델 목록 재조회 필요

### Migration Path

1. config.yaml에 provider가 있고 DB에 없으면 → 첫 기동 시 자동 마이그레이션
2. DB에 provider가 있으면 → DB 우선 사용
3. config.yaml provider 섹션은 deprecated하지 않고 fallback으로 유지
