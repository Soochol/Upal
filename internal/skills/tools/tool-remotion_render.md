---
name: tool-remotion_render
description: Guide for tool node using remotion_render — renders Remotion React compositions to MP4
---

## Objective

Execute a Remotion render from a React composition code string. Requires an upstream agent that generates the composition code, plus a TTS audio file path.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `composition_code` | string | Yes | Full Remotion React/TSX composition source |
| `audio_path` | string | No | TTS output file path, e.g. `{{tts_node}}` |
| `duration_sec` | number | No | Video duration in seconds (default: 60) |
| `fps` | number | No | Frames per second (default: 30) |
| `width` | number | No | Width in pixels (default: 1920) |
| `height` | number | No | Height in pixels (default: 1080) |

## Output

File path string to the rendered `.mp4`, e.g. `/tmp/upal-outputs/abc123.mp4`.

## Typical pipeline position

```
script_agent → tts_node → remotion_compose_agent → remotion_render (tool) → output
```

## Rules

1. `composition_code` must be complete, renderable Remotion source — not a snippet.
2. `audio_path` should reference the TTS node output: `{{tts_node}}`.
3. The upstream remotion_compose agent MUST be instructed to output ONLY the code, no markdown fences.
4. Host must have Node.js and `@remotion/renderer` installed.
