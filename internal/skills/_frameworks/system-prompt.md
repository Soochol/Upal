---
name: system-prompt
description: Shared guide for writing rich expert system_prompts for agent nodes
---

## SYSTEM PROMPT FRAMEWORK

When writing a `system_prompt` for an agent node, you MUST create a rich expert persona — not a generic assistant. The persona should feel like briefing a real human expert on exactly how to perform their role.

Structure the system_prompt with these elements woven naturally into a cohesive paragraph (do NOT use literal section headers):

1. **ROLE** — Define a specific expert identity with concrete domain expertise.
   - BAD: "You are a helpful assistant."
   - GOOD: "You are a senior tech blog editor with 10 years of experience in developer content strategy."

2. **EXPERTISE** — List 3-5 core competencies the agent excels at.
   - Example: "Your expertise includes SEO-optimized technical writing, audience engagement metrics, code snippet formatting, and narrative structure for developer audiences."

3. **STYLE** — Specify tone and communication approach appropriate for the task.
   - Example: "Write in a conversational yet authoritative tone. Use short paragraphs, clear subheadings, and concrete examples."

4. **CONSTRAINTS** — Set clear rules and boundaries for the agent's behavior.
   - Example: "Always include a strong opening hook. Keep paragraphs under 4 sentences. Never fabricate data or statistics. Cite sources when referencing specific claims."

5. **OUTPUT FORMAT** — Do NOT include output format instructions here. Use the separate `output` config field instead.

**Quality bar**: Generic or shallow system_prompts like "You are a helpful assistant" or "You are an AI that helps with writing" are NOT acceptable. Every system_prompt must demonstrate deep domain understanding and specific behavioral guidance.
