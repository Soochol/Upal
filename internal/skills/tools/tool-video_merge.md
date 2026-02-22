---
name: tool-video_merge
description: Guide for tool node using video_merge — merges video and audio files with FFmpeg
---

## Objective

Combine media files using FFmpeg. Use `mux_audio` to add a TTS audio track to a video, or `concat` to join clips sequentially.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `inputs` | array of strings | Yes | File paths to merge. For `mux_audio`: `[video_path, audio_path]` |
| `mode` | string | No | `"mux_audio"` (default) or `"concat"` |
| `output_format` | string | No | Container format (default: `"mp4"`) |

## Output

File path string to the merged file, e.g. `/tmp/upal-outputs/abc123.mp4`.

## Examples

### Add TTS audio to Remotion video
```json
{
  "tool": "video_merge",
  "input": {
    "inputs": ["{{remotion_render_node}}", "{{tts_node}}"],
    "mode": "mux_audio"
  }
}
```

### Concatenate clips
```json
{
  "tool": "video_merge",
  "input": {
    "inputs": ["{{intro_node}}", "{{main_node}}", "{{outro_node}}"],
    "mode": "concat"
  }
}
```

## Rules

1. Host must have `ffmpeg` on PATH.
2. For `mux_audio`, inputs order is always `[video, audio]`.
3. Output path is returned as a plain string — reference as `{{video_merge_node}}`.
