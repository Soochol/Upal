package agents

import (
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"
	"time"

	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// SensorNodeBuilder creates agents that wait for an external condition
// before allowing downstream nodes to proceed.
//
// Config:
//   - mode: "poll" (default) or "webhook"
//   - url: (poll mode) HTTP endpoint to poll
//   - connection_id: (poll mode, optional) connection for auth headers
//   - condition: (poll mode) expression evaluated against response body
//   - interval: (poll mode) poll interval in seconds (default 10)
//   - timeout: timeout in seconds (default 300)
type SensorNodeBuilder struct{}

func (b *SensorNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeSensor }

func (b *SensorNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	mode, _ := nd.Config["mode"].(string)
	if mode == "" {
		mode = "poll"
	}
	urlTpl, _ := nd.Config["url"].(string)
	condition, _ := nd.Config["condition"].(string)

	intervalSec := 10
	if v, ok := nd.Config["interval"].(float64); ok && v > 0 {
		intervalSec = int(v)
	}
	timeoutSec := 300
	if v, ok := nd.Config["timeout"].(float64); ok && v > 0 {
		timeoutSec = int(v)
	}

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Sensor node %s (%s)", nodeID, mode),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()
				timeout := time.Duration(timeoutSec) * time.Second

				var result string
				var err error

				switch mode {
				case "webhook":
					result, err = sensorWebhook(ctx, nodeID, timeout)
				default: // "poll"
					resolvedURL := resolveTemplateFromState(urlTpl, state)
					resolvedCond := resolveTemplateFromState(condition, state)
					interval := time.Duration(intervalSec) * time.Second
					result, err = sensorPoll(ctx, nodeID, resolvedURL, resolvedCond, interval, timeout)
				}

				if err != nil {
					yield(nil, fmt.Errorf("sensor node %q: %w", nodeID, err))
					return
				}

				_ = state.Set(nodeID, result)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(result)},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = result
				yield(event, nil)
			}
		},
	})
}

// sensorPoll polls an HTTP URL at the given interval until the condition
// expression evaluates to true or the timeout is reached.
func sensorPoll(ctx agent.InvocationContext, nodeID, url, condition string, interval, timeout time.Duration) (string, error) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	client := &http.Client{Timeout: 30 * time.Second}

	for {
		body, err := httpGet(ctx, client, url)
		if err == nil && condition != "" {
			// Put the response body into a temporary state for condition evaluation.
			state := ctx.Session().State()
			_ = state.Set(nodeID+"_response", body)
			resolved := resolveTemplateFromState(condition, state)
			ok, evalErr := evaluateCondition(resolved, state)
			if evalErr == nil && ok {
				return body, nil
			}
		} else if err == nil && condition == "" {
			// No condition — any successful response satisfies the sensor.
			return body, nil
		}

		select {
		case <-deadline:
			return "", fmt.Errorf("timeout after %v", timeout)
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			// Continue polling.
		}
	}
}

// sensorWebhook emits a "waiting" event and blocks until the execution
// is resumed via the external resume API.
func sensorWebhook(ctx agent.InvocationContext, nodeID string, timeout time.Duration) (string, error) {
	handle := ExecutionHandleFromContext(ctx)
	if handle == nil {
		return "", fmt.Errorf("webhook mode requires an ExecutionHandle in context")
	}

	// Emit a waiting event so the frontend knows this node is paused.
	waitEvent := session.NewEvent(ctx.InvocationID())
	waitEvent.Author = nodeID
	waitEvent.Branch = ctx.Branch()
	waitEvent.Actions.StateDelta["__status__"] = string(upal.NodeStatusWaiting)

	done := make(chan map[string]any, 1)
	go func() {
		done <- handle.WaitForResume(nodeID)
	}()

	select {
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for webhook resume after %v", timeout)
	case <-ctx.Done():
		return "", ctx.Err()
	case payload := <-done:
		result := fmt.Sprintf("%v", payload)
		return result, nil
	}
}

// httpGet performs a simple GET request and returns the response body as a string.
func httpGet(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", err
	}
	return string(body), nil
}
