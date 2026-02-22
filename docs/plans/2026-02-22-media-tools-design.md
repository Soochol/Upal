# Media Tools Design: TTS, SRT, Remotion, Video Merge

**Date**: 2026-02-22
**Context**: YouTube automation pipeline — script → audio → video → final output

---

## Goals

Add media production capabilities to Upal workflows:
- **TTS**: text → audio file (speech synthesis)
- **SRT**: text script → subtitle file (timed caption format)
- **Remotion render**: React composition code → video file
- **Video merge**: multiple media files → single output file

---

## Architecture Decisions

### 1. TTS — Model, not Tool

TTS follows the existing image model pattern: a new `ModelOption` category `"tts"` with providers implementing `adkmodel.LLM`.

**Rationale**: TTS maps cleanly to the agent node's existing fields:
- `model` → TTS provider/model ID (e.g. `"openai/tts-1-hd"`)
- `system_prompt` → speaking instructions (tone, pace, emotion)
- `prompt` → text to speak (via `{{upstream_node}}` template)

This is consistent with how `"image"` models work: the agent node is a pure transformer of the prompt into a different modality output. No new node type needed.

**Execution flow**:
1. `llm_builder.go` calls `TTSModel.GenerateContent(ctx, req, false)`
2. `req.Config.SystemInstruction` → speaking instructions
3. `req.Contents` → text to speak
4. TTS model calls provider API → receives audio binary
5. Returns `adkmodel.LLMResponse` with `InlineData{Data: audioBytes, MIMEType: "audio/mp3"}`
6. `llm_builder.go` detects audio MIME type in InlineData → saves to output dir → stores **file path string** in session state

**Why file path, not data URI**: Audio files are multi-MB; data URIs are unsuitable for large binaries. Downstream consumers (Remotion, video merge) need file paths, not inline data. LLMs cannot "hear" audio so inlining provides no value.

**Session state result**: `state["tts_node"] = "/tmp/upal-outputs/tts-{uuid}.mp3"` (plain string)

**Downstream access**: `{{tts_node}}` resolves to the file path in any node's `input` or `prompt` field.

### 2. SRT — Agent output, no dedicated tool

SRT generation does not require a dedicated tool or node type. The script-writing agent already has all necessary context (the full script text) to produce SRT output directly.

**Rationale**: SRT from a text script is text formatting, not a transformation that benefits from Go infrastructure. The agent estimates timing based on character/word count and speaking pace. Frame-perfect accuracy is not required for YouTube automation.

**Pattern**: The script agent outputs a JSON object:
```json
{
  "text": "안녕하세요, 오늘은 AI에 대해 이야기합니다...",
  "srt": "1\n00:00:00,000 --> 00:00:03,500\n안녕하세요, 오늘은\n\n2\n...",
  "tts_instructions": "밝고 에너지 넘치게, 훅 구간은 빠르게"
}
```

Downstream nodes reference individual fields as strings — currently via `{{script_agent}}` (full JSON). If dot-notation (`{{script_agent.srt}}`) is not yet supported, the Remotion compose agent parses the JSON from `{{script_agent}}`.

### 3. Remotion — Two-stage: Agent → Tool

AI-generated Remotion compositions require two separate nodes:

**Stage 1 — `remotion_compose` (agent node, LLM model)**
Receives script JSON and TTS file path, generates a complete Remotion React composition as a string.

**Stage 2 — `remotion_render` (tool node)**
Receives the composition code string, writes it to a temp file, runs `@remotion/renderer`, returns the output `.mp4` file path string.

**Rationale for separation**: Render failures need to be inspectable (the generated code can be logged/reviewed before rendering). Combining both in one agent turn makes debugging opaque. The tool node's deterministic interface also gives clean error reporting.

**`remotion_render` tool input schema**:
```json
{
  "composition_code": "<React/Remotion source code string>",
  "audio_path": "{{tts_node}}",
  "duration_sec": 60,
  "fps": 30,
  "width": 1920,
  "height": 1080
}
```

**Returns**: file path string, e.g. `"/tmp/upal-outputs/render-{uuid}.mp4"`

**External dependency**: Node.js + `@remotion/renderer` package must be installed on the host.

### 4. Video Merge — Tool node

Pure deterministic file operation. FFmpeg concatenates or mixes audio/video tracks.

**`video_merge` tool input schema**:
```json
{
  "inputs": ["{{remotion_node}}", "{{tts_node}}"],
  "mode": "mux_audio",
  "output_format": "mp4"
}
```

**Modes**: `concat` (sequential), `mux_audio` (add audio track to video), `overlay`.

**Returns**: file path string.

**External dependency**: `ffmpeg` must be on PATH.

---

## File Path Convention

All media tools and TTS model write output files to a shared output directory (configured at startup, e.g. `./outputs/`). Files are named `{type}-{uuid}.{ext}` to avoid collisions. Cleanup of temporary files is out of scope for this iteration.

---

## Pipeline Structure

```
input(주제)
  → script_agent          [agent, model: LLM]
      system_prompt: 스크립트 작성 전문가 페르소나
      prompt: {{input_node}} 주제로 YouTube 스크립트 작성
      output: JSON { text, srt, tts_instructions }

  → tts_node              [agent, model: "openai/tts-1-hd"]
      system_prompt: {{script_agent}}.tts_instructions (또는 직접 지침)
      prompt: {{script_agent}}.text

  → remotion_compose      [agent, model: LLM]
      prompt: {{script_agent}} 와 오디오 {{tts_node}} 를 사용하는 Remotion 컴포지션 코드 작성

  → remotion_render       [tool node, tool: "remotion_render"]
      input: { composition_code: {{remotion_compose}}, audio_path: {{tts_node}} }

  → output
```

---

## Changes Required

### New files
| File | Description |
|------|-------------|
| `internal/model/tts.go` | TTS provider implementations (OpenAI, ElevenLabs) implementing `adkmodel.LLM` |
| `internal/tools/remotion_render.go` | Remotion render tool |
| `internal/tools/video_merge.go` | FFmpeg-based video merge tool |

### Modified files
| File | Change |
|------|--------|
| `internal/llmutil/response.go` | `ExtractContent`: detect audio MIME type → save file → return path |
| `internal/llmutil/response.go` | Needs `outputDir` parameter or global config access |
| `internal/generate/generate.go` | `buildModelPrompt`: add `"tts"` category section |
| `cmd/upal/main.go` | Register TTS models and new tools; pass output dir |
| `internal/skills/nodes/agent-node.md` | Add TTS model usage guidance |
| `internal/skills/nodes/tool-node.md` | Add `remotion_render`, `video_merge` to examples |

### New skill files
| File | Description |
|------|-------------|
| `internal/skills/tools/tool-remotion_render.md` | Parameters, prompt patterns, pitfalls |
| `internal/skills/tools/tool-video_merge.md` | Parameters, modes, examples |

---

## Open Questions

1. **`ExtractContent` signature change**: Adding `outputDir` requires threading it through `llm_builder.go` → `BuildDeps`. Or use a wrapper function passed at build time.
2. **Dot notation for JSON fields**: `{{script_agent.text}}` vs `{{script_agent}}` + LLM parses full JSON. Dot notation is cleaner but requires template engine change. Defer to implementation decision.
3. **TTS provider registration**: Follow existing provider registry pattern (`"provider/model"` format) or a separate TTS registry. Recommendation: reuse `"provider/model"` with category `"tts"` in `ModelOption`.
4. **Remotion project structure**: The `remotion_render` tool needs a base Remotion project (with `package.json`, Remotion config) to compile against. This must be pre-installed on the host — document as a deployment requirement.
