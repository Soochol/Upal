---
name: tool-http_request
description: Make HTTP requests to external APIs — GET/POST/PUT/PATCH/DELETE/HEAD
---

## Overview

Executes raw HTTP requests against any REST API or webhook endpoint. Use this when interacting with APIs that require authentication, custom headers, or structured POST payloads — anything that `get_webpage` or `fetch_rss` cannot handle.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `method` | string | Yes | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD` |
| `url` | string | Yes | Full request URL |
| `headers` | object | No | HTTP headers as key-value pairs |
| `body` | string | No | Request body as a string. For JSON, serialize manually. |

## Returns

| Field | Type | Description |
|-------|------|-------------|
| `status_code` | number | HTTP status code (200, 404, 500, etc.) |
| `status` | string | HTTP status line (e.g. `"200 OK"`, `"404 Not Found"`) |
| `headers` | object | Response headers as key-value pairs |
| `body` | string | Response body as string, max 100KB. JSON is preserved as-is. |

**Always check `status_code`** before using the response body. A 4xx or 5xx response means the request failed.

## Agent Prompt Patterns

**Authenticated GET:**
```
Call the API:
  method: GET
  url: https://api.example.com/v1/items
  headers: { "Authorization": "Bearer {{api_token}}", "Accept": "application/json" }

Parse the JSON response body and extract [specific fields].
```

**POST with JSON body:**
```
Submit the following data:
  method: POST
  url: https://api.example.com/v1/items
  headers: { "Content-Type": "application/json", "Authorization": "Bearer {{api_token}}" }
  body: {"name": "{{item_name}}", "value": "{{item_value}}"}

Report the created item's ID from the response.
```

**Paginated collection:**
```
Fetch all pages from the API until no more results:
1. GET https://api.example.com/items?page=1&limit=100
2. If the response JSON includes a non-null "next_cursor", fetch the next page using cursor={{next_cursor}}.
3. Repeat until "next_cursor" is null.
4. Return the combined item count and a summary.
```

**Webhook delivery:**
```
POST the analysis result to the webhook:
  method: POST
  url: {{webhook_url}}
  headers: { "Content-Type": "application/json" }
  body: {"event": "pipeline_completed", "result": "{{analysis_result}}"}

Verify the response status_code is 2xx. Report success or the error.
```

**API with query parameters:**
```
Call the search API:
  method: GET
  url: https://api.example.com/search?q={{encoded_query}}&limit=10&format=json
  headers: { "X-API-Key": "{{api_key}}" }

Extract the "results" array from the response and format as a list.
```

## Pitfalls & Limitations

- **Body must be a plain string** — JSON must be serialized as a string literal. Tell the LLM: `body: {"key": "value"}` — note the absence of outer quotes around the JSON itself when the LLM constructs the call.
- **Response body is always a string** — even for JSON APIs, `body` is returned as text. Instruct the LLM to mentally parse the JSON and extract specific fields rather than treating it as a native object.
- **100KB response cap** — large responses are truncated. Request paginated or filtered endpoints to stay under the limit.
- **30-second timeout** — slow APIs will fail. Not suitable for long-polling or streaming endpoints.
- **No automatic redirect on non-GET methods** — POST redirects (3xx) may not follow correctly.
- **No retry logic** — failed requests are returned immediately. Instruct the LLM to retry if needed.
- **URL encoding** — ensure query parameters with special characters are URL-encoded (spaces → `%20`, etc.).
