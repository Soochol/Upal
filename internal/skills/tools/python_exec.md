---
name: tool-python_exec
description: Execute Python 3 code — returns stdout, stderr, exit_code
---

## Overview

Executes arbitrary Python 3 code in an isolated subprocess and captures its output. Use for calculations, data transformation, parsing, format conversion, and any computation that's more reliable in code than in the LLM's reasoning.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `code` | string | Yes | Complete Python 3 source code to execute |

## Returns

| Field | Type | Description |
|-------|------|-------------|
| `stdout` | string | Standard output, max 100KB. The primary output channel. |
| `stderr` | string | Standard error output, max 100KB. Contains tracebacks on failure. |
| `exit_code` | number | `0` = success, non-zero = error |

**Always check `exit_code`** before using `stdout`. If `exit_code != 0`, read `stderr` for the error message.

## Available Libraries

Standard library is always available:
- Data: `json`, `csv`, `re`, `struct`, `base64`
- Math: `math`, `statistics`, `decimal`, `fractions`
- Time: `datetime`, `calendar`, `time`
- Collections: `collections`, `itertools`, `functools`
- IO: `io`, `pathlib`, `textwrap`, `string`

**Do not assume** third-party libraries (pandas, numpy, requests, etc.) are installed unless confirmed by the user. Use only the standard library by default.

## Agent Prompt Patterns

**Data calculation:**
```
Write Python code that calculates [specific formula] using this input data:
{{data_input}}

Print the result as a JSON object: {"result": <value>, "unit": "<unit>"}
```

**Parse and transform JSON:**
```
Write Python code that:
1. Parses this JSON string: {{json_data}}
2. Extracts the "name" and "score" fields from each item in the "results" array
3. Sorts by score descending
4. Prints a CSV string with header: name,score

Print only the CSV to stdout, nothing else.
```

**Statistical analysis:**
```
Write Python code to analyze this data: {{numbers}}
(The data is a newline-separated list of numbers.)

Calculate and print JSON:
{"mean": <float>, "median": <float>, "std_dev": <float>, "min": <float>, "max": <float>, "count": <int>}
```

**Format conversion:**
```
Write Python code to convert the following CSV data to a JSON array of objects.
The first row is the header.

Input:
{{csv_data}}

Print the JSON array to stdout (compact, no indentation).
```

**Text processing:**
```
Write Python code using only the standard library to:
1. Read this text: {{raw_text}}
2. Count word frequency (case-insensitive, ignore punctuation)
3. Print the top 10 words as JSON: [{"word": "...", "count": N}, ...]
```

**Error-safe pattern (for critical paths):**
```
Write Python code that:
1. Attempts [operation]
2. On success, prints: {"status": "ok", "result": <value>}
3. On any exception, prints: {"status": "error", "message": "<error string>"}
   and exits with code 1 (sys.exit(1))

Always import sys at the top.
```

## Pitfalls & Limitations

- **Always print to stdout** — the tool captures `stdout`, not function return values. Every script must end with `print(...)`.
- **30-second timeout** — scripts exceeding this are killed with no output. Optimize for speed; avoid nested loops over large datasets.
- **No internet access** — `urllib`, `http.client`, `socket` calls to external hosts will fail or time out. Use `http_request` tool for external calls instead.
- **No persistent state** — each call is an isolated subprocess. Variables from previous calls are gone.
- **100KB output cap** — large outputs are truncated. For big datasets, summarize or paginate.
- **Avoid file I/O** — do not read or write files unless the path is explicitly provided in the workflow.
- **JSON numbers vs strings** — when printing numbers, do not quote them. Use `json.dumps({"value": 42})` not `{"value": "42"}`.
- **Newlines in multi-value output** — if printing multiple values, use `json.dumps()` for structured output rather than multiple `print()` calls.
