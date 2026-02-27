---
name: run_input-node
description: Guide for configuring run_input nodes — pipeline brief entry points
---

## Objective

Configure a run_input node that receives data from pipeline runs automatically.

Unlike the `input` node (which collects data from the user at execution time), `run_input` is populated by the pipeline runner with a structured brief containing the analysis context, selected angle, source highlights, and reference URLs.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | No | Brief explanation of what this run_input receives |

## Rules

1. A workflow should have at most **one** `run_input` node.
2. The `run_input` node is the entry point for pipeline-triggered executions. When a pipeline selects an angle and triggers a workflow, the brief is injected into this node.
3. The brief content includes: assignment (selected angle + rationale), context summary, per-source highlights, cross-source insights, and reference URLs.
4. Downstream agent nodes should reference this node via `{{node_id}}` to access the brief.
5. If a workflow has no `run_input` node, the pipeline runner falls back to populating `input` nodes instead (backward compatibility).
