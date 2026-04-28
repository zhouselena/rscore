package rscore

import (
	"fmt"
	"log"
)

type Node struct {
	Name 			string
	Type 			string // "functional" or "provider"
	OutNeighbours 	map[string]*Node
	InNeighbours 	map[string]*Node
	Betweenness 	float64
	// add further calculated information here
}

type Edge struct {
	Name 		string
	Type 		string // "dependency" or "hosted-on"
	FromNode 	*Node
	ToNode		*Node
	Weight 		float64 // existing resilience dependency here maybe?
}

// Directed graph
type Graph struct {
	Name 		string // application name
	Nodes		map[string]*Node
	Edges 		map[string]*Edge
	Weighted 	bool
}

// Initialise and return new Graph
func CreateGraph(name string, weighted bool) (*Graph, error) {

	if name == "" {
		return nil, fmt.Errorf("graph name cannot be empty")
	}

	g := &Graph{
		Name: name,
		Nodes: make(map[string]*Node),
		Edges: make(map[string]*Edge),
		Weighted: weighted,

	}

	log.Printf("Graph %q created (weighted=%v)", name, weighted)
	return g, nil
}

func (g *Graph) NodeCount() int {
	return len(g.Nodes)
}

func (g *Graph) EdgeCount() int {
	return len(g.Edges)
}

func (g *Graph) AddNode(name, nodeType string) (*Node, error) {
	if name == "" {
		return nil, fmt.Errorf("node name cannot be empty")
	}
	if _, exists := g.Nodes[name]; exists {
		return nil, fmt.Errorf("node %q already exists", name)
	}

	n := &Node{
		Name: name,
		Type: nodeType,
		OutNeighbours: make(map[string]*Node),
		InNeighbours: make(map[string]*Node),
	}
	g.Nodes[name] = n

	log.Printf("Node %q (%s) added to graph %q", name, nodeType, g.Name)
	return n, nil
}

func (g *Graph) RemoveNode(name string) error {
	node, exists := g.Nodes[name]
	if !exists {
		return fmt.Errorf("node %q not found", name)
	}

	// Remove from node outNeighbours
	for _, outN := range node.OutNeighbours {
		delete(outN.InNeighbours, name)
	}

	// Remove from node inNeighbours
	for _, inN := range node.InNeighbours {
		delete(inN.OutNeighbours, name)
	}

	// Remove from edges
	for edgeName, edge := range g.Edges {
		if (edge.FromNode != nil && edge.FromNode.Name == name) || (edge.ToNode != nil && edge.ToNode.Name == name) {
			delete(g.Edges, edgeName)
		}
	}

	delete(g.Nodes, name)
	log.Printf("Node %q removed from graph %q", name, g.Name)
	return nil
}

// Add optional weight later
func (g *Graph) AddEdge(name, edgeType, from, to string) (*Edge, error) {
	fromNode, ok := g.Nodes[from]
	if !ok {
		return nil, fmt.Errorf("source node %q not found", from)
	}
	toNode, ok := g.Nodes[to]
	if !ok {
		return nil, fmt.Errorf("target node %q not found", to)
	}
	if name == "" {
		name = fmt.Sprintf("%s->%s", from, to)
	}
	if _, exists := g.Edges[name]; exists {
		return nil, fmt.Errorf("edge %q already exists", name)
	}

	e := &Edge{
		Name: name,
		Type: edgeType,
		FromNode: fromNode,
		ToNode: toNode,
	}
	g.Edges[name] = e

	fromNode.OutNeighbours[to] = toNode
	toNode.InNeighbours[from] = fromNode

	log.Printf("Edge %q (%s) added to graph %q", name, edgeType, g.Name)
	return e, nil
}

func (g *Graph) RemoveEdge(name string) error {
	edge, exists := g.Edges[name]
	if !exists {
		return fmt.Errorf("edge %q not found", name)
	}

	// remove from connected nodes in/outneighbours
	delete(edge.FromNode.OutNeighbours, edge.ToNode.Name)
	delete(edge.ToNode.InNeighbours, edge.FromNode.Name)

	// remove edge
	delete(g.Edges, name)
	log.Printf("Edge %q removed from graph %q", name, g.Name)

	return nil
}

func (g *Graph) InDegree(name string) (int, error) {
	node, exists := g.Nodes[name]
	if !exists {
		return 0, fmt.Errorf("node %q not found", name)
	}

	return len(node.InNeighbours), nil
}

func (g *Graph) OutDegree(name string) (int, error) {
	node, exists := g.Nodes[name]
	if !exists {
		return 0, fmt.Errorf("node %q not found", name)
	}

	return len(node.OutNeighbours), nil
}
