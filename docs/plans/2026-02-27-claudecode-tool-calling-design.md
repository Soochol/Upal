# ClaudeCodeLLM Tool Calling 지원

**날짜**: 2026-02-27
**상태**: 승인됨

## 문제

ClaudeCodeLLM은 `claude -p` CLI를 호출하는 방식으로 API 키 없이 Claude 구독만으로 LLM을 사용할 수 있다. 하지만 CLI는 custom FunctionDeclaration을 지원하지 않아서, Upal의 tool calling이 필요한 기능들이 모두 실패한다:

- **Chat**: generate_workflow, configure_node 등 tool 호출 불가 → LLM이 `<tool_call>` 텍스트를 출력하지만 실행되지 않음
- **Workflow/Pipeline 생성**: `get_skill` tool로 스킬 문서를 참조할 수 없음 → 품질 저하
- **Agent 노드 실행**: web_search, python_exec 등 외부 tool 호출 불가
- **Research fetcher**: get_webpage tool 미지원

Anthropic/OpenAI/Gemini는 모두 tool calling이 완전 구현되어 있으며, ClaudeCodeLLM만 미지원.

## 설계

### 핵심 아이디어

ClaudeCodeLLM 내부에서 text-based tool calling을 구현한다:

1. custom FunctionDeclaration이 있으면 **시스템 프롬프트에 tool 스키마를 텍스트로 삽입**
2. LLM 응답에서 **`<tool_call>` 블록을 파싱**하여 `genai.FunctionCall` Part로 변환
3. Handler가 보내는 **`FunctionResponse` Part를 텍스트로 변환**하여 다음 CLI 호출에 포함

결과적으로 ClaudeCodeLLM이 다른 LLM과 동일한 `adkmodel.LLM` 인터페이스를 제공하므로, **handler/executor 코드 변경이 불필요**하다.

### 변경 파일

`internal/model/claudecode.go` — 이 파일만 수정

### 상세 설계

#### 1. Tool 정의 → 시스템 프롬프트 삽입

`generate()` 메서드에서 `req.Config.Tools`의 FunctionDeclaration을 순회하여 시스템 프롬프트 끝에 추가:

```
## Available Tools

You have access to the following tools. When you need to use a tool, output EXACTLY this format (no other text around it):

<tool_call>
{"name": "tool_name", "arguments": {"key": "value"}}
</tool_call>

You may output multiple tool calls. After tool calls, STOP and wait for results.

### generate_workflow
Generate a new workflow or edit an existing one from a natural-language description.
Parameters:
  - description (string, required): Natural-language description of the workflow

### configure_node
Configure a selected workflow node based on a user instruction.
Parameters:
  - node_id (string, required): ID of the node to configure
  - message (string, required): User instruction
```

#### 2. 응답 파싱 → FunctionCall 변환

CLI 텍스트 출력에서 `<tool_call>...</tool_call>` 블록을 정규식으로 추출:

```go
var toolCallRe = regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)
```

각 매치를 JSON 파싱하여 `genai.FunctionCall` Part로 변환. tool call 앞뒤의 텍스트는 별도 Text Part로 포함.

tool call이 하나라도 있으면 `FunctionCall` Part를 포함한 응답 반환 → handler가 tool 실행 루프 진입.

#### 3. FunctionResponse → 대화 텍스트 변환

현재 `generate()`의 메시지 빌드 로직에서 `FunctionCall`/`FunctionResponse` Part를 무시하고 있음. 이를 수정하여:

- **FunctionCall Part** (이전 턴의 LLM tool 호출):
  ```
  [Assistant called tool: generate_workflow]
  Arguments: {"description": "블로그 글 쓰는 워크플로우"}
  ```

- **FunctionResponse Part** (tool 실행 결과):
  ```
  [Tool result: generate_workflow]
  {"nodes": [...], "edges": [...], "name": "blog-writer"}
  ```

이렇게 텍스트로 변환하여 다음 CLI 호출의 대화 컨텍스트에 포함.

#### 4. Tool call ID 생성

Anthropic API와 달리 CLI 응답에는 tool call ID가 없으므로, `generate()` 내부에서 순차적으로 ID를 생성: `cc_tool_0`, `cc_tool_1`, ...

Handler가 FunctionResponse로 결과를 돌려보낼 때 이 ID를 사용하므로 매칭에 문제 없음.

#### 5. --effort 조정

현재 tool이 없고 thinking이 아닐 때 `--effort low`를 설정하는 로직이 있음. Custom tool이 있을 때는 effort를 설정하지 않아 기본값(medium)을 사용하도록 함 — tool 사용 판단에 충분한 추론이 필요하므로.

### 파싱 실패 시 폴백

`<tool_call>` 블록 파싱에 실패하면 (JSON 깨짐 등) 텍스트 그대로 반환. Handler는 tool call이 없으므로 텍스트 응답으로 처리 — 현재와 동일한 동작.

### 기존 native tool (WebSearch) 호환

`mapToolsToCLI()`가 이미 GoogleSearch → WebSearch 매핑을 처리하므로, FunctionDeclaration 중 이미 CLI에 매핑된 tool은 시스템 프롬프트에 추가하지 않음. 즉:

- GoogleSearch → CLI의 `--tools WebSearch`로 처리 (기존 방식 유지)
- Custom FunctionDeclaration → 시스템 프롬프트 + 텍스트 파싱으로 처리

### 테스트 계획

1. **tool call 파싱 단위 테스트**: 다양한 형식의 `<tool_call>` 블록 파싱
2. **FunctionResponse 텍스트 변환 테스트**: tool 결과가 올바른 텍스트로 변환되는지
3. **tool 정의 프롬프트 생성 테스트**: FunctionDeclaration → 시스템 프롬프트 텍스트
4. **전체 round-trip 테스트**: tool 정의 → 응답 파싱 → FunctionCall 변환 → FunctionResponse 수신 → 텍스트 재구성

### 성능 고려

- tool call마다 CLI 프로세스를 재실행하므로 각 턴당 10-30초 소요
- Chat에서 generate_workflow 호출 시: 턴1 (LLM이 tool call 결정) + 턴2 (tool 결과 보고 최종 응답) = 2번 CLI 실행
- 이는 Claude Code CLI의 본질적 한계이며, API 키 없이 사용하는 trade-off

### 리스크

| 리스크 | 완화 |
|--------|------|
| LLM이 형식을 안 따름 | 폴백으로 텍스트 반환. 시스템 프롬프트에서 형식을 강조 |
| JSON 파싱 실패 | 정규식 매치 후 JSON 파싱. 실패 시 해당 블록 무시 |
| CLI 업데이트 | 파싱 대상은 LLM 텍스트 출력이지 CLI 형식이 아님 — 영향 없음 |
| 긴 tool 결과 | CLI stdin에 전달. 크기 제한은 없지만 토큰 제한에 주의 |
