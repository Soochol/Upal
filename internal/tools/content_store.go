// internal/tools/content_store.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type ContentStoreTool struct {
	mu   sync.RWMutex
	path string
	data map[string]string
}

func NewContentStoreTool(path string) *ContentStoreTool {
	t := &ContentStoreTool{
		path: path,
		data: make(map[string]string),
	}
	t.load()
	return t
}

func (c *ContentStoreTool) Name() string { return "content_store" }
func (c *ContentStoreTool) Description() string {
	return "Persistent key-value store for tracking state across pipeline runs. Use for deduplication (seen URLs), timestamps (last collection), counters, and any data that must survive between executions."
}

func (c *ContentStoreTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []any{"get", "set", "list", "delete"},
				"description": "Operation to perform",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "Key to get, set, or delete",
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Value to store (required for 'set' action)",
			},
			"prefix": map[string]any{
				"type":        "string",
				"description": "Key prefix filter for 'list' action",
			},
		},
		"required": []any{"action"},
	}
}

func (c *ContentStoreTool) Execute(_ context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected object")
	}

	action, _ := args["action"].(string)
	key, _ := args["key"].(string)

	switch action {
	case "get":
		if key == "" {
			return nil, fmt.Errorf("key is required for get")
		}
		c.mu.RLock()
		val, exists := c.data[key]
		c.mu.RUnlock()
		if !exists {
			return map[string]any{"value": nil, "found": false}, nil
		}
		return map[string]any{"value": val, "found": true}, nil

	case "set":
		if key == "" {
			return nil, fmt.Errorf("key is required for set")
		}
		value, _ := args["value"].(string)
		c.mu.Lock()
		c.data[key] = value
		err := c.save()
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("failed to persist: %w", err)
		}
		return map[string]any{"status": "ok"}, nil

	case "list":
		prefix, _ := args["prefix"].(string)
		c.mu.RLock()
		var keys []string
		for k := range c.data {
			if prefix == "" || strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}
		c.mu.RUnlock()
		sort.Strings(keys)
		return map[string]any{"keys": keys, "count": len(keys)}, nil

	case "delete":
		if key == "" {
			return nil, fmt.Errorf("key is required for delete")
		}
		c.mu.Lock()
		delete(c.data, key)
		err := c.save()
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("failed to persist: %w", err)
		}
		return map[string]any{"status": "ok"}, nil

	default:
		return nil, fmt.Errorf("unknown action %q: use get, set, list, or delete", action)
	}
}

func (c *ContentStoreTool) load() {
	raw, err := os.ReadFile(c.path)
	if err != nil {
		return // file doesn't exist yet — start empty
	}
	json.Unmarshal(raw, &c.data)
}

func (c *ContentStoreTool) save() error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	raw, err := json.Marshal(c.data)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, raw, 0644)
}
