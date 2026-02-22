---
name: tool-tts
description: TTS is implemented as a model category, not a tool. Use a TTS agent node.
---

TTS (text-to-speech) in Upal is implemented as a model-type node, following the same pattern as image generation. Use an agent node with a TTS model:

- `model`: `"openai/tts-1-hd"`
- `system_prompt`: speaking style instructions
- `prompt`: `{{upstream_script_node}}`

Output: file path string stored in session state, accessible via `{{tts_node}}`.
