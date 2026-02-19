package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
)

type A2ARunner struct {
	eventBus *EventBus
	sessions *SessionManager
	client   *a2aclient.Client
}

func NewA2ARunner(eventBus *EventBus, sessions *SessionManager, client *a2aclient.Client) *A2ARunner {
	return &A2ARunner{eventBus: eventBus, sessions: sessions, client: client}
}

func (r *A2ARunner) Run(ctx context.Context, wf *WorkflowDefinition, nodeURLs map[string]string, userInputs map[string]any) (*Session, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dag, err := BuildDAG(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	sess := r.sessions.Create(wf.Name)

	if userInputs != nil {
		for k, v := range userInputs {
			r.sessions.SetState(sess.ID, "__user_input__"+k, v)
			textVal := fmt.Sprintf("%v", v)
			r.sessions.SetArtifacts(sess.ID, "__user_input__"+k, []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart(textVal)},
				Index: 0,
			}})
		}
	}

	r.eventBus.Publish(Event{
		ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
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
				errOnce.Do(func() {
					execErr = fmt.Errorf("no URL for node %q", nodeID)
					cancel()
				})
				close(done[nodeID])
				return
			}

			r.eventBus.Publish(Event{
				ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeStarted, Timestamp: time.Now(),
			})

			allArtifacts := r.sessions.GetAllArtifacts(sess.ID)

			// Build combined artifact map with user input aliases.
			// User inputs are aliased so {{input1}} resolves to __user_input__input1.
			combinedArtifacts := make(map[string][]a2atypes.Artifact, len(allArtifacts))
			for k, v := range allArtifacts {
				combinedArtifacts[k] = v
			}
			for k, v := range allArtifacts {
				if strings.HasPrefix(k, "__user_input__") {
					alias := strings.TrimPrefix(k, "__user_input__")
					combinedArtifacts[alias] = v
				}
			}

			// Check for prompt template in node config
			nodeDef := dag.Node(nodeID)
			var messageText string
			if prompt, ok := nodeDef.Config["prompt"].(string); ok && prompt != "" {
				messageText = a2atypes.ResolveTemplate(prompt, combinedArtifacts)
			} else {
				messageText = buildMessageText(nodeID, allArtifacts, dag)
			}
			if messageText == "" {
				messageText = fmt.Sprintf("Execute node %s", nodeID)
			}

			msg := a2atypes.Message{
				Role:  "user",
				Parts: []a2atypes.Part{a2atypes.TextPart(messageText)},
			}
			task, err := r.client.SendMessage(ctx, nodeURL, msg)
			if err != nil {
				r.eventBus.Publish(Event{
					ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
					NodeID: nodeID, Type: EventNodeError, Payload: map[string]any{"error": err.Error()}, Timestamp: time.Now(),
				})
				errOnce.Do(func() {
					execErr = fmt.Errorf("node %q: %w", nodeID, err)
					cancel()
				})
				close(done[nodeID])
				return
			}

			if len(task.Artifacts) > 0 {
				r.sessions.SetArtifacts(sess.ID, nodeID, task.Artifacts)
			}
			// Legacy state compat — extract first text from typed artifacts
			if len(task.Artifacts) > 0 {
				text := task.Artifacts[0].FirstText()
				if text != "" {
					r.sessions.SetState(sess.ID, nodeID, text)
				}
			}

			r.eventBus.Publish(Event{
				ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeCompleted,
				Payload: map[string]any{"status": string(task.Status)}, Timestamp: time.Now(),
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

func buildMessageText(nodeID string, allArtifacts map[string][]a2atypes.Artifact, dag *DAG) string {
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
		text := arts[0].FirstText()
		if text != "" {
			texts = append(texts, text)
		}
	}
	if len(texts) == 0 {
		return ""
	}
	return strings.Join(texts, "\n\n")
}
