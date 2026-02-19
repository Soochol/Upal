package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
)

// Client sends A2A JSON-RPC requests to agent endpoints over HTTP.
type Client struct {
	httpClient *http.Client
	nextID     atomic.Int64
}

// NewClient creates a new A2A client with the given HTTP client.
func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

// SendMessage sends an a2a.sendMessage JSON-RPC request to the given URL
// and returns the resulting Task.
func (c *Client) SendMessage(ctx context.Context, url string, msg Message) (*Task, error) {
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID.Add(1),
		Method:  "a2a.sendMessage",
		Params: SendMessageParams{
			Message: msg,
			Configuration: &SendMessageConfig{
				AcceptedOutputModes: []string{"text/plain", "application/json"},
				Blocking:            true,
			},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("a2a error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	resultData, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	var task Task
	if err := json.Unmarshal(resultData, &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &task, nil
}
