package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type A2ARunner struct {
	eventBus   *EventBus
	sessions   *SessionManager
	httpClient *http.Client
	nextID     atomic.Int64
}

func NewA2ARunner(eventBus *EventBus, sessions *SessionManager, httpClient *http.Client) *A2ARunner {
	return &A2ARunner{eventBus: eventBus, sessions: sessions, httpClient: httpClient}
}

func (r *A2ARunner) Run(ctx context.Context, wf *WorkflowDefinition, nodeURLs map[string]string, userInputs map[string]any) (*Session, error) {
	dag, err := BuildDAG(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	sess := r.sessions.Create(wf.Name)

	if userInputs != nil {
		for k, v := range userInputs {
			r.sessions.SetState(sess.ID, "__user_input__"+k, v)
			textVal := fmt.Sprintf("%v", v)
			r.sessions.SetArtifacts(sess.ID, "__user_input__"+k, []any{
				map[string]any{
					"parts": []map[string]any{{"type": "text", "text": textVal, "mimeType": "text/plain"}},
					"index": 0,
				},
			})
		}
	}

	r.eventBus.Publish(Event{
		ID: GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
		Type: EventNodeStarted, Payload: map[string]any{"workflow": wf.Name}, Timestamp: time.Now(),
	})

	done := make(map[string]chan struct{})
	for _, n := range wf.Nodes {
		done[n.ID] = make(chan struct{})
	}

	var wg sync.WaitGroup
	var execErr error
	var errOnce sync.Once

	for _, nodeID := range dag.TopologicalOrder() {
		nodeID := nodeID
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, parentID := range dag.Parents(nodeID) {
				select {
				case <-done[parentID]:
				case <-ctx.Done():
					return
				}
			}

			nodeURL, ok := nodeURLs[nodeID]
			if !ok {
				errOnce.Do(func() { execErr = fmt.Errorf("no URL for node %q", nodeID) })
				close(done[nodeID])
				return
			}

			r.eventBus.Publish(Event{
				ID: GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeStarted, Timestamp: time.Now(),
			})

			allArtifacts := r.sessions.GetAllArtifacts(sess.ID)
			messageText := buildMessageText(nodeID, allArtifacts, dag)
			if messageText == "" {
				messageText = fmt.Sprintf("Execute node %s", nodeID)
			}

			task, err := r.sendMessage(ctx, nodeURL, messageText)
			if err != nil {
				r.eventBus.Publish(Event{
					ID: GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
					NodeID: nodeID, Type: EventNodeError, Payload: map[string]any{"error": err.Error()}, Timestamp: time.Now(),
				})
				errOnce.Do(func() { execErr = fmt.Errorf("node %q: %w", nodeID, err) })
				close(done[nodeID])
				return
			}

			if artifacts, ok := task["artifacts"].([]any); ok {
				r.sessions.SetArtifacts(sess.ID, nodeID, artifacts)
			}
			// Also store in legacy state for backward compat
			if arts, ok := task["artifacts"].([]any); ok && len(arts) > 0 {
				if art, ok := arts[0].(map[string]any); ok {
					if parts, ok := art["parts"].([]any); ok && len(parts) > 0 {
						if part, ok := parts[0].(map[string]any); ok {
							if text, ok := part["text"].(string); ok {
								r.sessions.SetState(sess.ID, nodeID, text)
							}
						}
					}
				}
			}

			r.eventBus.Publish(Event{
				ID: GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeCompleted, Payload: map[string]any{"task": task}, Timestamp: time.Now(),
			})
			close(done[nodeID])
		}()
	}

	wg.Wait()

	if execErr != nil {
		r.sessions.SetStatus(sess.ID, SessionFailed)
		finalSess, _ := r.sessions.Get(sess.ID)
		return finalSess, execErr
	}
	r.sessions.SetStatus(sess.ID, SessionCompleted)
	finalSess, _ := r.sessions.Get(sess.ID)
	return finalSess, nil
}

func (r *A2ARunner) sendMessage(ctx context.Context, url, text string) (map[string]any, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      r.nextID.Add(1),
		"method":  "a2a.sendMessage",
		"params": map[string]any{
			"message": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"type": "text", "text": text, "mimeType": "text/plain"},
				},
			},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcErr, ok := rpcResp["error"]; ok && rpcErr != nil {
		errMap, _ := rpcErr.(map[string]any)
		return nil, fmt.Errorf("a2a error: %v", errMap["message"])
	}
	result, ok := rpcResp["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result type")
	}
	status, _ := result["status"].(string)
	if status == "failed" {
		return nil, fmt.Errorf("task failed")
	}
	return result, nil
}

func buildMessageText(nodeID string, allArtifacts map[string][]any, dag *DAG) string {
	parents := dag.Parents(nodeID)
	if len(parents) == 0 {
		return ""
	}
	var texts []string
	for _, parentID := range parents {
		arts, ok := allArtifacts[parentID]
		if !ok {
			arts = allArtifacts["__user_input__"+parentID]
		}
		if len(arts) == 0 {
			continue
		}
		if art, ok := arts[0].(map[string]any); ok {
			if parts, ok := art["parts"].([]any); ok {
				for _, p := range parts {
					if part, ok := p.(map[string]any); ok {
						if text, ok := part["text"].(string); ok {
							texts = append(texts, text)
						}
					}
				}
			}
		}
	}
	if len(texts) == 0 {
		return ""
	}
	result := ""
	for i, t := range texts {
		if i > 0 {
			result += "\n\n"
		}
		result += t
	}
	return result
}
