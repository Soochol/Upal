package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxResponseBody caps how much of an HTTP response body we return to the LLM.
const maxResponseBody = 100 * 1024 // 100 KB

// allowedMethods is the set of HTTP methods this tool supports.
var allowedMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true,
}

// HTTPRequestTool makes HTTP requests to external APIs and URLs.
type HTTPRequestTool struct{}

func (h *HTTPRequestTool) Name() string { return "http_request" }

func (h *HTTPRequestTool) Description() string {
	return "Make HTTP requests to external APIs and URLs. Returns the response status, headers, and body."
}

func (h *HTTPRequestTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method: GET, POST, PUT, PATCH, DELETE, or HEAD",
			},
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to send the request to",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Optional HTTP headers as key-value pairs",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Optional request body (for POST, PUT, PATCH)",
			},
		},
		"required": []any{"method", "url"},
	}
}

func (h *HTTPRequestTool) Execute(ctx context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected object")
	}

	method, _ := args["method"].(string)
	method = strings.ToUpper(method)
	if !allowedMethods[method] {
		return nil, fmt.Errorf("unsupported HTTP method: %q", method)
	}

	url, _ := args["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	// Build request body
	var bodyReader io.Reader
	if body, ok := args["body"].(string); ok && body != "" {
		bodyReader = strings.NewReader(body)
	}

	// Apply timeout
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if hdrs, ok := args["headers"].(map[string]any); ok {
		for k, v := range hdrs {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read body with size cap
	limited := io.LimitReader(resp.Body, maxResponseBody+1)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := string(bodyBytes)
	if len(bodyBytes) > maxResponseBody {
		bodyStr = bodyStr[:maxResponseBody] + "\n... [truncated at 100KB]"
	}

	// Collect response headers
	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	return map[string]any{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     respHeaders,
		"body":        bodyStr,
	}, nil
}
