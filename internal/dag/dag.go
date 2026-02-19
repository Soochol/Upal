package dag

import (
	"fmt"
	"sort"

	"github.com/soochol/upal/internal/upal"
)

type DAG struct {
	nodes     map[string]*upal.NodeDefinition
	children  map[string][]string
	parents   map[string][]string
	edges     map[string]upal.EdgeDefinition
	backEdges []upal.EdgeDefinition
	topoOrder []string
}

func Build(wf *upal.WorkflowDefinition) (*DAG, error) {
	d := &DAG{
		nodes:    make(map[string]*upal.NodeDefinition),
		children: make(map[string][]string),
		parents:  make(map[string][]string),
		edges:    make(map[string]upal.EdgeDefinition),
	}

	for i := range wf.Nodes {
		n := &wf.Nodes[i]
		if _, exists := d.nodes[n.ID]; exists {
			return nil, fmt.Errorf("duplicate node ID: %s", n.ID)
		}
		d.nodes[n.ID] = n
	}

	for _, e := range wf.Edges {
		if _, ok := d.nodes[e.From]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.From)
		}
		if _, ok := d.nodes[e.To]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.To)
		}
		key := e.From + "->" + e.To
		d.edges[key] = e
		if e.Loop != nil {
			d.backEdges = append(d.backEdges, e)
			continue
		}
		d.children[e.From] = append(d.children[e.From], e.To)
		d.parents[e.To] = append(d.parents[e.To], e.From)
	}

	order, err := d.topoSort()
	if err != nil {
		return nil, err
	}
	d.topoOrder = order
	return d, nil
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
	sort.Strings(queue)
	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		children := d.children[node]
		for _, c := range children {
			inDegree[c]--
			if inDegree[c] == 0 {
				queue = append(queue, c)
			}
		}
		sort.Strings(queue)
	}
	if len(order) != len(d.nodes) {
		return nil, fmt.Errorf("cycle detected in workflow graph (excluding back-edges)")
	}
	return order, nil
}

func (d *DAG) TopologicalOrder() []string        { return d.topoOrder }
func (d *DAG) Children(nodeID string) []string    { return d.children[nodeID] }
func (d *DAG) Parents(nodeID string) []string     { return d.parents[nodeID] }
func (d *DAG) Node(id string) *upal.NodeDefinition { return d.nodes[id] }
func (d *DAG) BackEdges() []upal.EdgeDefinition   { return d.backEdges }

func (d *DAG) Roots() []string {
	var roots []string
	for id := range d.nodes {
		if len(d.parents[id]) == 0 {
			roots = append(roots, id)
		}
	}
	return roots
}

func (d *DAG) Edge(from, to string) (upal.EdgeDefinition, bool) {
	e, ok := d.edges[from+"->"+to]
	return e, ok
}
