package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/soochol/upal/internal/a2atypes"
)

type Runner struct {
	eventBus *EventBus
	sessions *SessionManager
}

func NewRunner(eventBus *EventBus, sessions *SessionManager) *Runner {
	return &Runner{eventBus: eventBus, sessions: sessions}
}

func (r *Runner) Run(ctx context.Context, wf *WorkflowDefinition, executors map[NodeType]NodeExecutorInterface, userInputs map[string]any) (*Session, error) {
	dag, err := BuildDAG(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	sess := r.sessions.Create(wf.Name)

	if userInputs != nil {
		for k, v := range userInputs {
			r.sessions.SetState(sess.ID, "__user_input__"+k, v)
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

			stateCopy := r.sessions.GetStateCopy(sess.ID)

			nodeDef := dag.Node(nodeID)
			executor, ok := executors[nodeDef.Type]
			if !ok {
				errOnce.Do(func() { execErr = fmt.Errorf("no executor for node type %q", nodeDef.Type) })
				close(done[nodeID])
				return
			}

			r.eventBus.Publish(Event{
				ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeStarted, Timestamp: time.Now(),
			})

			result, err := executor.Execute(ctx, nodeDef, stateCopy)
			if err != nil {
				r.eventBus.Publish(Event{
					ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
					NodeID: nodeID, Type: EventNodeError, Payload: map[string]any{"error": err.Error()}, Timestamp: time.Now(),
				})
				errOnce.Do(func() { execErr = fmt.Errorf("node %q: %w", nodeID, err) })
				close(done[nodeID])
				return
			}

			r.sessions.SetState(sess.ID, nodeID, result)

			r.eventBus.Publish(Event{
				ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeCompleted, Payload: map[string]any{"result": result}, Timestamp: time.Now(),
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
