package agents

import (
	"fmt"
	"iter"
	"sync"

	"github.com/soochol/upal/internal/dag"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
)

// nodeOutcome records the execution result of a single node.
type nodeOutcome struct {
	Status upal.NodeStatus
	Err    error
}

// shouldRun evaluates whether a node should execute based on its incoming
// edges' TriggerRule and Condition fields. A node runs if at least one
// incoming edge is "active" (trigger rule matches parent outcome AND
// condition expression evaluates to true). Edges with no TriggerRule
// default to on_success. Edges with no Condition always pass.
func shouldRun(d *dag.DAG, nodeID string, outcomes map[string]*nodeOutcome, mu *sync.RWMutex, state session.State) bool {
	parents := d.Parents(nodeID)
	if len(parents) == 0 {
		return true // root nodes always run
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, parentID := range parents {
		edge, ok := d.Edge(parentID, nodeID)
		if !ok {
			continue
		}

		parentOutcome := outcomes[parentID]

		// Check trigger rule against parent outcome.
		if !triggerMatches(edge.TriggerRule, parentOutcome) {
			continue
		}

		// Check condition expression against session state.
		if edge.Condition != "" {
			ok, err := evaluateCondition(edge.Condition, state)
			if err != nil || !ok {
				continue
			}
		}

		return true // at least one active edge found
	}

	return false
}

// triggerMatches returns true if the edge's trigger rule is satisfied by the
// parent node's outcome. Default (empty) trigger rule behaves as on_success.
func triggerMatches(rule upal.TriggerRule, parent *nodeOutcome) bool {
	if parent == nil {
		// Parent completed without recording an outcome (legacy path);
		// treat as success since done channel was closed.
		return rule == "" || rule == upal.TriggerOnSuccess || rule == upal.TriggerAlways
	}
	switch rule {
	case upal.TriggerAlways:
		return true
	case upal.TriggerOnFailure:
		return parent.Status == upal.NodeStatusFailed
	default: // "" or on_success
		return parent.Status == upal.NodeStatusCompleted
	}
}

// NewDAGAgent creates an ADK Custom Agent that executes a workflow DAG.
//
// It builds a DAG from the workflow definition, creates an ADK agent for each
// node via the NodeRegistry, and returns a custom agent whose Run function
// executes the DAG with goroutine fan-out: each node waits for its parent
// nodes to complete before running.
func NewDAGAgent(wf *upal.WorkflowDefinition, registry *NodeRegistry, deps BuildDeps) (agent.Agent, error) {
	// 1. Build the DAG from workflow definition.
	d, err := dag.Build(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	// 2. Build agents for each node via the registry.
	nodeAgents := make(map[string]agent.Agent, len(wf.Nodes))
	subAgents := make([]agent.Agent, 0, len(wf.Nodes))

	for i := range wf.Nodes {
		nd := &wf.Nodes[i]
		a, err := registry.Build(nd, deps)
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

				// Outcome tracking per node (protected by mu).
				var mu sync.RWMutex
				outcomes := make(map[string]*nodeOutcome, len(topoOrder))

				// Cancel channel: closed on first unrecoverable error.
				cancelCh := make(chan struct{})
				var cancelOnce sync.Once
				cancelFn := func() { cancelOnce.Do(func() { close(cancelCh) }) }

				// hasFailureEdge returns true if the given node has at least
				// one outgoing edge with TriggerOnFailure or TriggerAlways,
				// meaning a downstream node can handle its failure.
				hasFailureEdge := func(nodeID string) bool {
					for _, childID := range d.Children(nodeID) {
						if e, ok := d.Edge(nodeID, childID); ok {
							if e.TriggerRule == upal.TriggerOnFailure || e.TriggerRule == upal.TriggerAlways {
								return true
							}
						}
					}
					return false
				}

				var wg sync.WaitGroup

				// Channel to collect events from goroutines.
				type nodeEvent struct {
					event *session.Event
					err   error
				}
				eventCh := make(chan nodeEvent, len(topoOrder)*24)

				// Launch goroutines per topological order.
				for _, nodeID := range topoOrder {
					nodeID := nodeID
					nodeAgent := nodeAgents[nodeID]

					wg.Add(1)
					go func() {
						defer wg.Done()
						defer close(done[nodeID])

						// Wait for all parent channels to close.
						for _, parentID := range d.Parents(nodeID) {
							select {
							case <-done[parentID]:
							case <-ctx.Done():
								return
							case <-cancelCh:
								return
							}
						}

						// Check cancellation.
						select {
						case <-cancelCh:
							return
						default:
						}
						if ctx.Err() != nil {
							return
						}

						// Evaluate incoming edge conditions.
						if !shouldRun(d, nodeID, outcomes, &mu, ctx.Session().State()) {
							mu.Lock()
							outcomes[nodeID] = &nodeOutcome{Status: upal.NodeStatusSkipped}
							mu.Unlock()

							// Emit a "skipped" event so the frontend can show this node as skipped.
							skipEv := session.NewEvent(ctx.InvocationID())
							skipEv.Author = nodeID
							skipEv.Branch = ctx.Branch()
							skipEv.Actions.StateDelta["__status__"] = string(upal.NodeStatusSkipped)
							eventCh <- nodeEvent{skipEv, nil}
							return
						}

						// Emit a lightweight "started" event.
						startEv := session.NewEvent(ctx.InvocationID())
						startEv.Author = nodeID
						startEv.Branch = ctx.Branch()
						eventCh <- nodeEvent{startEv, nil}

						// Run the node agent and collect events.
						var nodeErr error
						for ev, err := range nodeAgent.Run(ctx) {
							if err != nil {
								nodeErr = fmt.Errorf("node %q: %w", nodeID, err)
								break
							}
							eventCh <- nodeEvent{ev, nil}
						}

						if nodeErr != nil {
							mu.Lock()
							outcomes[nodeID] = &nodeOutcome{Status: upal.NodeStatusFailed, Err: nodeErr}
							mu.Unlock()

							// Only cancel if this node has no failure-handling edges.
							if !hasFailureEdge(nodeID) {
								eventCh <- nodeEvent{nil, nodeErr}
								cancelFn()
							}
							return
						}

						mu.Lock()
						outcomes[nodeID] = &nodeOutcome{Status: upal.NodeStatusCompleted}
						mu.Unlock()
					}()
				}

				// Close event channel when all goroutines complete.
				go func() {
					wg.Wait()
					close(eventCh)
				}()

				// Yield events as they arrive.
				for ne := range eventCh {
					if ne.err != nil {
						cancelFn()
					}
					if !yield(ne.event, ne.err) {
						cancelFn()
						return
					}
				}
			}
		},
	})
}
