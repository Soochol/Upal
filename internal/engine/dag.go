package engine

import "fmt"

type DAG struct {
	nodes     map[string]*NodeDefinition
	children  map[string][]string
	parents   map[string][]string
	edges     map[string]EdgeDefinition
	backEdges []EdgeDefinition
	topoOrder []string
}

func BuildDAG(wf *WorkflowDefinition) (*DAG, error) {
	dag := &DAG{
		nodes:    make(map[string]*NodeDefinition),
		children: make(map[string][]string),
		parents:  make(map[string][]string),
		edges:    make(map[string]EdgeDefinition),
	}

	for i := range wf.Nodes {
		n := &wf.Nodes[i]
		if _, exists := dag.nodes[n.ID]; exists {
			return nil, fmt.Errorf("duplicate node ID: %s", n.ID)
		}
		dag.nodes[n.ID] = n
	}

	for _, e := range wf.Edges {
		if _, ok := dag.nodes[e.From]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.From)
		}
		if _, ok := dag.nodes[e.To]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.To)
		}
		key := e.From + "->" + e.To
		dag.edges[key] = e
		if e.Loop != nil {
			dag.backEdges = append(dag.backEdges, e)
			continue
		}
		dag.children[e.From] = append(dag.children[e.From], e.To)
		dag.parents[e.To] = append(dag.parents[e.To], e.From)
	}

	order, err := dag.topoSort()
	if err != nil {
		return nil, err
	}
	dag.topoOrder = order
	return dag, nil
}

func (d *DAG) topoSort() ([]string, error) {
	inDegree := make(map[string]int)
	for id := range d.nodes {
		inDegree[id] = 0
	}
	for _, children := range d.children {
		for _, c := range children {
			inDegree[c]++
		}
	}
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		for _, c := range d.children[node] {
			inDegree[c]--
			if inDegree[c] == 0 {
				queue = append(queue, c)
			}
		}
	}
	if len(order) != len(d.nodes) {
		return nil, fmt.Errorf("cycle detected in workflow graph (excluding back-edges)")
	}
	return order, nil
}

func (d *DAG) TopologicalOrder() []string { return d.topoOrder }
func (d *DAG) Children(nodeID string) []string { return d.children[nodeID] }
func (d *DAG) Parents(nodeID string) []string  { return d.parents[nodeID] }
func (d *DAG) Roots() []string {
	var roots []string
	for id := range d.nodes {
		if len(d.parents[id]) == 0 {
			roots = append(roots, id)
		}
	}
	return roots
}
func (d *DAG) BackEdges() []EdgeDefinition { return d.backEdges }
func (d *DAG) Node(id string) *NodeDefinition { return d.nodes[id] }
func (d *DAG) Edge(from, to string) (EdgeDefinition, bool) {
	e, ok := d.edges[from+"->"+to]
	return e, ok
}
