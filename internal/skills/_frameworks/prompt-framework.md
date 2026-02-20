---
name: prompt-framework
description: Shared guide for writing effective user prompts (the prompt field) for agent nodes
---

## USER PROMPT FRAMEWORK

When writing the `prompt` field for an agent node, you MUST create a clear, actionable instruction that the agent will execute at runtime. The prompt receives data from upstream nodes via `{{node_id}}` template references.

Structure the prompt with these elements:

1. **CONTEXT** — Inject upstream data using `{{node_id}}` template references.
   - CRITICAL: When upstream nodes exist, you MUST use `{{node_id}}` to receive their output. The `{{node_id}}` placeholder gets replaced with the actual upstream node's output at runtime.
   - BAD: "Analyze the following text: [paste text here]"
   - GOOD: "Analyze the following text:\n\n{{user_input}}"
   - If multiple upstream nodes exist, reference each one explicitly: "Topic: {{topic_input}}\n\nResearch data: {{researcher}}"

2. **TASK** — Give the agent a specific, unambiguous instruction about what to produce.
   - BAD: "Do something with this."
   - GOOD: "Write a comprehensive blog post about the topic above, targeting intermediate developers."

3. **FORMAT** — When relevant, specify the desired output format.
   - Example: "Format your response as a JSON object with keys: summary, key_points, recommendations."
   - Example: "Use Markdown with H2 headings for each section."

**Rules**:
- NEVER write hardcoded placeholder text like "[insert here]" or "[여기에 입력]" — always use `{{node_id}}` references.
- Every upstream node's output should be consumed — don't ignore available data.
- Keep prompts focused on ONE clear task. If the task is complex, break it into steps within the prompt.
