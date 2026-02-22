# Interface Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 하드코딩된 프로바이더/타입별 분기를 인터페이스/레지스트리 패턴으로 교체해 새 프로바이더·툴·소스 추가 시 단일 파일만 수정하도록 한다.

**Architecture:** 4개 독립 리팩터. 각각 기존 동작을 그대로 유지하며 구조만 개선. 공통 패턴은 "레지스트리에 등록 + 중앙 lookup" 이다.

**Tech Stack:** Go 1.23, `google.golang.org/adk`, `google.golang.org/genai`

---

## Task 1: Provider Factory Registry

기존 `main.go`의 `switch pc.Type { case "anthropic": ... }` 를 제거하고
각 모델 파일이 자신의 팩토리를 `init()` 으로 등록한다.

**Files:**
- Create: `internal/model/registry.go`
- Modify: `internal/model/anthropic.go` — init() 추가
- Modify: `internal/model/openai.go` — init() 추가
- Modify: `internal/model/gemini_text.go` — init() 추가
- Modify: `internal/model/gemini_image.go` — init() 추가
- Modify: `internal/model/claudecode.go` — init() 추가
- Modify: `internal/model/zimage.go` — init() 추가
- Modify: `cmd/upal/main.go` — switch 제거 → BuildLLM() 호출
- Test: `internal/model/registry_test.go`

### Step 1: registry.go 작성 (테스트 먼저)

`internal/model/registry_test.go`:
```go
package model_test

import (
	"testing"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/model"
)

func TestBuildLLM_KnownType(t *testing.T) {
	llm, ok := model.BuildLLM("myprovider", config.ProviderConfig{
		Type:   "anthropic",
		APIKey: "test-key",
	})
	if !ok {
		t.Fatal("expected ok=true for anthropic")
	}
	if llm == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestBuildLLM_UnknownTypeWithURL(t *testing.T) {
	llm, ok := model.BuildLLM("myprovider", config.ProviderConfig{
		Type:   "some-openai-compat",
		URL:    "http://localhost:1234/v1",
		APIKey: "key",
	})
	if !ok {
		t.Fatal("expected ok=true for fallback with URL")
	}
	if llm == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestBuildLLM_UnknownTypeNoURL(t *testing.T) {
	_, ok := model.BuildLLM("myprovider", config.ProviderConfig{
		Type: "totally-unknown",
	})
	if ok {
		t.Fatal("expected ok=false for unknown type with no URL")
	}
}
```

Run: `go test ./internal/model/ -run TestBuildLLM -v`
Expected: FAIL (BuildLLM not defined)

### Step 2: registry.go 구현

`internal/model/registry.go`:
```go
package model

import (
	"github.com/soochol/upal/internal/config"
	adkmodel "google.golang.org/adk/model"
)

// LLMFactory creates an adkmodel.LLM for a given provider name and config.
type LLMFactory func(providerName string, cfg config.ProviderConfig) adkmodel.LLM

var factories = map[string]LLMFactory{}

// RegisterProvider registers a factory for the given provider type string.
// Called from init() in each model implementation file.
func RegisterProvider(typeName string, factory LLMFactory) {
	factories[typeName] = factory
}

// BuildLLM looks up a registered factory for cfg.Type and calls it.
// If no factory is found but cfg.URL is set, falls back to OpenAI-compat.
// Returns (nil, false) if the type is unknown and no URL fallback is available.
func BuildLLM(providerName string, cfg config.ProviderConfig) (adkmodel.LLM, bool) {
	if factory, ok := factories[cfg.Type]; ok {
		return factory(providerName, cfg), true
	}
	// Fallback: any provider with a URL is treated as OpenAI-compatible.
	if cfg.URL != "" {
		return NewOpenAILLM(cfg.APIKey,
			WithOpenAIBaseURL(cfg.URL),
			WithOpenAIName(providerName)), true
	}
	return nil, false
}
```

Run: `go test ./internal/model/ -run TestBuildLLM -v`
Expected: FAIL (factories empty — init() not added yet)

### Step 3: 각 모델 파일에 init() 추가

`internal/model/anthropic.go` 끝에 추가:
```go
func init() {
	RegisterProvider("anthropic", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewAnthropicLLM(cfg.APIKey)
	})
}
```

`internal/model/gemini_text.go` 끝에 추가:
```go
func init() {
	RegisterProvider("gemini", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewGeminiLLM(name, cfg.APIKey)
	})
}
```

`internal/model/gemini_image.go` 끝에 추가:
```go
func init() {
	RegisterProvider("gemini-image", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewGeminiImageLLM(cfg.APIKey)
	})
}
```

`internal/model/claudecode.go` 끝에 추가 (파일 확인 필요):
```go
func init() {
	RegisterProvider("claude-code", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewClaudeCodeLLM()
	})
}
```

`internal/model/zimage.go` 끝에 추가 (파일 확인 필요):
```go
func init() {
	RegisterProvider("zimage", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewZImageLLM(cfg.URL)
	})
}
```

`internal/model/openai.go` 끝에 추가 (openai-tts 포함):
```go
func init() {
	RegisterProvider("openai", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewOpenAILLM(cfg.APIKey,
			WithOpenAIBaseURL(cfg.URL),
			WithOpenAIName(name))
	})
	RegisterProvider("openai-tts", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewOpenAITTSModel(cfg.APIKey, cfg.URL)
	})
}
```

Run: `go test ./internal/model/ -run TestBuildLLM -v`
Expected: PASS

### Step 4: main.go switch 교체

`cmd/upal/main.go` 의 provider loop를:
```go
// 기존
for name, pc := range cfg.Providers {
    switch pc.Type {
    case "anthropic": ...
    case "gemini": ...
    ...
    }
    providerTypes[name] = pc.Type
}

// 변경
for name, pc := range cfg.Providers {
    llm, ok := upalmodel.BuildLLM(name, pc)
    if !ok {
        slog.Warn("unknown provider type, skipping", "name", name, "type", pc.Type)
        continue
    }
    llms[name] = llm
    providerTypes[name] = pc.Type
}
```

main.go imports에서 더 이상 필요없는 항목 정리 (있다면).

Run: `go build ./...`
Expected: 빌드 성공

### Step 5: 통합 확인 후 커밋

```bash
go test ./... -race
git add internal/model/registry.go internal/model/registry_test.go \
    internal/model/anthropic.go internal/model/openai.go \
    internal/model/gemini_text.go internal/model/gemini_image.go \
    internal/model/claudecode.go internal/model/zimage.go \
    cmd/upal/main.go
git commit -m "refactor(model): provider factory registry — remove switch in main.go"
```

---

## Task 2: Native Tool Registry 중앙화

현재 `AnthropicLLM`, `OpenAILLM`, `GeminiLLM` 세 파일이 동일한 switch 블록을 반복한다.
`native.go`의 중앙 맵으로 통합한다.

**Files:**
- Modify: `internal/model/native.go` — 중앙 레지스트리 추가
- Modify: `internal/model/anthropic.go` — NativeTool() 위임
- Modify: `internal/model/openai.go` — NativeTool() 위임
- Modify: `internal/model/gemini_text.go` — NativeTool() 위임
- Test: `internal/model/native_test.go`

### Step 1: 테스트 작성

`internal/model/native_test.go`:
```go
package model_test

import (
	"testing"
	"github.com/soochol/upal/internal/model"
)

func TestLookupNativeTool_Known(t *testing.T) {
	spec, ok := model.LookupNativeTool("web_search")
	if !ok {
		t.Fatal("web_search should be registered")
	}
	if spec == nil || spec.GoogleSearch == nil {
		t.Fatal("expected GoogleSearch spec")
	}
}

func TestLookupNativeTool_Unknown(t *testing.T) {
	_, ok := model.LookupNativeTool("unknown_tool")
	if ok {
		t.Fatal("unknown_tool should not be found")
	}
}

func TestRegisterNativeToolSpec_Custom(t *testing.T) {
	model.RegisterNativeToolSpec("test_tool_xyz", &genai.Tool{})
	spec, ok := model.LookupNativeTool("test_tool_xyz")
	if !ok || spec == nil {
		t.Fatal("custom tool should be findable after registration")
	}
}
```

Run: `go test ./internal/model/ -run TestLookupNativeTool -v`
Expected: FAIL

### Step 2: native.go 에 레지스트리 함수 추가

`internal/model/native.go` 수정:
```go
package model

import "google.golang.org/genai"

// NativeToolProvider is an optional interface ...
// (기존 주석 유지)
type NativeToolProvider interface {
	NativeTool(name string) (*genai.Tool, bool)
}

// nativeToolSpecs maps well-known Upal tool names to their genai.Tool spec.
// All LLM implementations that implement NativeToolProvider share this registry,
// so adding a new native tool requires editing only this file.
var nativeToolSpecs = map[string]*genai.Tool{}

// RegisterNativeToolSpec registers a genai.Tool spec for a native tool name.
// Call from init() to add new native tools without modifying model implementations.
func RegisterNativeToolSpec(name string, spec *genai.Tool) {
	nativeToolSpecs[name] = spec
}

// LookupNativeTool returns the genai.Tool spec for the given tool name.
func LookupNativeTool(name string) (*genai.Tool, bool) {
	spec, ok := nativeToolSpecs[name]
	return spec, ok
}

func init() {
	RegisterNativeToolSpec("web_search", &genai.Tool{GoogleSearch: &genai.GoogleSearch{}})
}
```

Run: `go test ./internal/model/ -run TestLookupNativeTool -v`
Expected: PASS

### Step 3: 세 모델의 NativeTool() 를 LookupNativeTool() 로 위임

`internal/model/anthropic.go`:
```go
func (a *AnthropicLLM) NativeTool(name string) (*genai.Tool, bool) {
	return LookupNativeTool(name)
}
```

`internal/model/openai.go`:
```go
func (o *OpenAILLM) NativeTool(name string) (*genai.Tool, bool) {
	return LookupNativeTool(name)
}
```

`internal/model/gemini_text.go`:
```go
func (g *GeminiLLM) NativeTool(name string) (*genai.Tool, bool) {
	return LookupNativeTool(name)
}
```

Run: `go build ./... && go test ./... -race`
Expected: 빌드 및 테스트 통과

### Step 4: 커밋

```bash
git add internal/model/native.go internal/model/native_test.go \
    internal/model/anthropic.go internal/model/openai.go internal/model/gemini_text.go
git commit -m "refactor(model): centralize native tool specs in native.go registry"
```

---

## Task 3: CollectStage Source Fetcher 인터페이스

현재 `CollectStageExecutor.fetchSource()` 의 switch를 레지스트리로 교체한다.
`pipeline_runner.go`의 `StageExecutor` 패턴과 일관성을 맞춘다.

**Files:**
- Modify: `internal/services/stage_collect.go`
- Test: `internal/services/stage_collect_test.go` (신규 또는 기존 파일 확인)

### Step 1: 테스트 작성

`internal/services/stage_collect_test.go`:
```go
package services_test

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// stubFetcher implements SourceFetcher for testing.
type stubFetcher struct {
	typ  string
	text string
	data any
	err  error
}

func (s *stubFetcher) Type() string { return s.typ }
func (s *stubFetcher) Fetch(_ context.Context, _ upal.CollectSource) (string, any, error) {
	return s.text, s.data, s.err
}

func TestCollectStageExecutor_CustomFetcher(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	exec.RegisterFetcher(&stubFetcher{typ: "mock", text: "hello", data: "data"})

	stage := upal.Stage{
		ID:   "s1",
		Type: "collect",
		Config: upal.StageConfig{
			Sources: []upal.CollectSource{{ID: "src1", Type: "mock", URL: "http://x"}},
		},
	}
	result, err := exec.Execute(context.Background(), stage, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["text"] != "hello" {
		t.Errorf("expected text='hello', got %q", result.Output["text"])
	}
}

func TestCollectStageExecutor_UnknownSourceType(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	stage := upal.Stage{
		ID:   "s1",
		Type: "collect",
		Config: upal.StageConfig{
			Sources: []upal.CollectSource{{ID: "src1", Type: "nonexistent", URL: "http://x"}},
		},
	}
	// Should not panic; unknown type is handled as partial failure
	result, err := exec.Execute(context.Background(), stage, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Text should contain error info
	text, _ := result.Output["text"].(string)
	if text == "" {
		t.Error("expected error text for unknown source type")
	}
}
```

Run: `go test ./internal/services/ -run TestCollectStageExecutor -v`
Expected: FAIL (RegisterFetcher not defined, SourceFetcher not exported)

### Step 2: stage_collect.go 리팩터

`SourceFetcher` 인터페이스 추가 및 `CollectStageExecutor` 수정:

```go
// SourceFetcher fetches data for a single CollectSource.
// Implement and register via CollectStageExecutor.RegisterFetcher to support new source types.
type SourceFetcher interface {
	Type() string
	Fetch(ctx context.Context, src upal.CollectSource) (string, any, error)
}
```

`CollectStageExecutor` 구조체에 fetchers 맵 추가:
```go
type CollectStageExecutor struct {
	httpClient *http.Client
	fetchers   map[string]SourceFetcher
}
```

`NewCollectStageExecutor()` 에서 기본 fetcher 등록:
```go
func NewCollectStageExecutor() *CollectStageExecutor {
	e := &CollectStageExecutor{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		fetchers:   make(map[string]SourceFetcher),
	}
	e.RegisterFetcher(&rssFetcher{client: e.httpClient})
	e.RegisterFetcher(&httpFetcher{client: e.httpClient})
	e.RegisterFetcher(&scrapeFetcher{client: e.httpClient})
	return e
}

func (e *CollectStageExecutor) RegisterFetcher(f SourceFetcher) {
	e.fetchers[f.Type()] = f
}
```

`fetchSource()` switch를 레지스트리 lookup으로 교체:
```go
func (e *CollectStageExecutor) fetchSource(ctx context.Context, src upal.CollectSource) (string, any, error) {
	f, ok := e.fetchers[src.Type]
	if !ok {
		return "", nil, fmt.Errorf("unknown source type %q", src.Type)
	}
	return f.Fetch(ctx, src)
}
```

기존 `fetchRSS`, `fetchHTTP`, `fetchScrape` 메서드를 각각 `rssFetcher`, `httpFetcher`, `scrapeFetcher` 타입으로 추출:

```go
type rssFetcher struct{ client *http.Client }
func (f *rssFetcher) Type() string { return "rss" }
func (f *rssFetcher) Fetch(ctx context.Context, src upal.CollectSource) (string, any, error) {
	// 기존 fetchRSS 로직 그대로 이전
}

type httpFetcher struct{ client *http.Client }
func (f *httpFetcher) Type() string { return "http" }
func (f *httpFetcher) Fetch(ctx context.Context, src upal.CollectSource) (string, any, error) {
	// 기존 fetchHTTP 로직 그대로 이전
}

type scrapeFetcher struct{ client *http.Client }
func (f *scrapeFetcher) Type() string { return "scrape" }
func (f *scrapeFetcher) Fetch(ctx context.Context, src upal.CollectSource) (string, any, error) {
	// 기존 fetchScrape 로직 그대로 이전
}
```

Run: `go test ./internal/services/ -run TestCollectStageExecutor -v`
Expected: PASS

Run: `go build ./...`
Expected: 빌드 성공

### Step 3: 커밋

```bash
git add internal/services/stage_collect.go internal/services/stage_collect_test.go
git commit -m "refactor(services): SourceFetcher interface for CollectStage — registry replaces switch"
```

---

## Task 4: Ollama 타입 감지 수정

현재 포트 번호("11434")로 Ollama를 감지하는 brittle한 방법을 수정한다.

**Files:**
- Modify: `internal/api/models.go`
- Modify: `internal/config/config.go` (선택적 — Backend 필드 추가)

**두 가지 옵션:**

### Option A (빠른 수정): isOllama를 type+url 조합으로 개선

```go
// isOllama detects if a provider config points to a local Ollama instance.
// Ollama uses OpenAI-compat type with a URL containing the default port 11434,
// or can be explicitly typed as "ollama".
func isOllama(pc config.ProviderConfig) bool {
	if pc.Type == "ollama" {
		return true
	}
	return pc.Type == "openai" && strings.Contains(pc.URL, "11434")
}
```

`config.yaml` 에 `type: ollama` 지원 추가 (문서화):
```yaml
ollama:
  type: ollama        # 명시적 타입 (기존 openai도 여전히 동작)
  url: "http://localhost:11434/v1"
```

`main.go` 의 provider factory에서 `ollama` 타입도 OpenAI-compat으로 처리:
`BuildLLM` fallback 로직이 URL이 있으면 자동으로 OpenAI-compat을 사용하므로 자동 처리됨.

### Step 1: 테스트 작성

```go
// internal/api/models_test.go 에 추가
func TestIsOllama(t *testing.T) {
	cases := []struct {
		pc   config.ProviderConfig
		want bool
	}{
		{config.ProviderConfig{Type: "ollama", URL: "http://localhost:11434/v1"}, true},
		{config.ProviderConfig{Type: "openai", URL: "http://localhost:11434/v1"}, true},
		{config.ProviderConfig{Type: "openai", URL: "http://localhost:8080/v1"}, false},
		{config.ProviderConfig{Type: "anthropic", URL: "http://localhost:11434/v1"}, false},
	}
	for _, c := range cases {
		got := isOllama(c.pc)  // package-level, test in same package
		if got != c.want {
			t.Errorf("isOllama(%+v) = %v, want %v", c.pc, got, c.want)
		}
	}
}
```

Run: `go test ./internal/api/ -run TestIsOllama -v`

### Step 2: isOllama 수정

```go
func isOllama(pc config.ProviderConfig) bool {
	if pc.Type == "ollama" {
		return true
	}
	return pc.Type == "openai" && strings.Contains(pc.URL, "11434")
}
```

Run: `go test ./internal/api/ -run TestIsOllama -v`
Expected: PASS

### Step 3: 커밋

```bash
git add internal/api/models.go
git commit -m "fix(api): isOllama supports explicit 'ollama' type, not just port heuristic"
```

---

## 완료 후 전체 검증

```bash
go test ./... -race -v 2>&1 | tail -20
go build ./...
```

모든 테스트 통과, 빌드 성공 확인.
