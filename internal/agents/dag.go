package agents

import (
	"fmt"
	"iter"
	"sync"

	"github.com/soochol/upal/internal/dag"
	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
)

// NewDAGAgent creates an ADK Custom Agent that executes a workflow DAG.
//
// It builds a DAG from the workflow definition, creates an ADK agent for each
// node via BuildAgent, and returns a custom agent whose Run function executes
// the DAG with goroutine fan-out: each node waits for its parent nodes to
// complete before running.
func NewDAGAgent(wf *upal.WorkflowDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	// 1. Build the DAG from workflow definition.
	d, err := dag.Build(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	// 2. Build agents for each node via BuildAgent.
	nodeAgents := make(map[string]agent.Agent, len(wf.Nodes))
	subAgents := make([]agent.Agent, 0, len(wf.Nodes))

	for i := range wf.Nodes {
		nd := &wf.Nodes[i]
		a, err := BuildAgent(nd, llms, toolReg)
		if err != nil {
			return nil, fmt.Errorf("build agent for node %q: %w", nd.ID, err)
		}
		nodeAgents[nd.ID] = a
		subAgents = append(subAgents, a)
	}

	topoOrder := d.TopologicalOrder()

	// 3. Return agent.New() with Run function implementing DAG execution.
	return agent.New(agent.Config{
		Name:        wf.Name,
		Description: fmt.Sprintf("DAG workflow agent: %s", wf.Name),
		SubAgents:   subAgents,
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				// Create done channels for each node.
				done := make(map[string]chan struct{}, len(topoOrder))
				for _, nodeID := range topoOrder {
					done[nodeID] = make(chan struct{})
				}

				var wg sync.WaitGroup
				var execErr error
				var errOnce sync.Once

				// Channel to collect events from goroutines in order.
				type nodeEvent struct {
					event *session.Event
					err   error
				}
				eventCh := make(chan nodeEvent, len(topoOrder)*2)

				// Launch goroutines per topological order.
				for _, nodeID := range topoOrder {
					nodeID := nodeID
					nodeAgent := nodeAgents[nodeID]

					wg.Add(1)
					go func() {
						defer wg.Done()
						defer close(done[nodeID])

						// Wait for parent channels.
						for _, parentID := range d.Parents(nodeID) {
							select {
							case <-done[parentID]:
							case <-ctx.Done():
								return
							}
						}

						// Check if we already have an error.
						if ctx.Err() != nil {
							return
						}

						// Run the node agent and collect events.
						for ev, err := range nodeAgent.Run(ctx) {
							if err != nil {
								errOnce.Do(func() {
									execErr = fmt.Errorf("node %q: %w", nodeID, err)
								})
								eventCh <- nodeEvent{nil, fmt.Errorf("node %q: %w", nodeID, err)}
								return
							}
							eventCh <- nodeEvent{ev, nil}
						}
					}()
				}

				// Close event channel when all goroutines complete.
				go func() {
					wg.Wait()
					close(eventCh)
				}()

				// Yield events as they arrive.
				for ne := range eventCh {
					if !yield(ne.event, ne.err) {
						return
					}
				}

				// If there was an error, yield it.
				if execErr != nil && ctx.Err() == nil {
					yield(nil, execErr)
				}
			}
		},
	})
}
