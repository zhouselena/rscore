package rscore

import (
	"fmt"
	"log"
	"maps"
	"slices"
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

func (g *Graph) BetweennessCentrality(normal bool) (map[string]float64) {

	betweenScores := make(map[string]float64)
	for name := range g.Nodes {
    	betweenScores[name] = 0.0
	}

	for sourceName, sourceNode := range g.Nodes{

		// forward pass

		var stack []*Node
		queue := []*Node{sourceNode} // seed BFS with source

		predecessors := make(map[string][]string) // make sure to use append(predecessors[node], "nodeName")
		sigma := make(map[string]int) // one path to self
		dist := make(map[string]int)  // dist to self
		sigma[sourceName] = 1
		dist[sourceName]  = 0

		for len(queue) > 0 { // while q not empty
			v := queue[0]
			queue = queue[1:]
			stack = append(stack, v)

			for wName, wNode := range v.OutNeighbours{
				// first time visiting neighbour
				if _, exists := dist[wName]; !exists {
					dist[wName] = dist[v.Name] + 1
					queue = append(queue, wNode)
				}
				// is path to w a shortest path
				if dist[wName] == dist[v.Name] + 1 {
					sigma[wName] += sigma[v.Name] // count the paths
					predecessors[wName] = append(predecessors[wName], v.Name) // v is a pred of w
				}
			}

		}

		// backward pass

		delta := make(map[string]float64)
		for s := range g.Nodes {
			delta[s] = 0.0
		}

		for len(stack) > 0 {
			wNode := stack[len(stack)-1]
			wName := wNode.Name
			stack = stack[:len(stack)-1]

			for _, vName := range predecessors[wName] {
				delta[vName] += (float64(sigma[vName])/float64(sigma[wName])) * float64(1+delta[wName])
			}
			if wName != sourceName {
					betweenScores[wName] += delta[wName]
			}
		}

	}

	// rescale / normalization

	N := len(g.Nodes)
	scale := 1.0 // default, no normalisation

	if normal && N > 2 {
		scale = 1.0 / float64((N-1) * (N-2))
	}

	for s := range g.Nodes {
		betweenScores[s] *= scale
	}


	return betweenScores

}

type DegreePair struct {
	In 		int
	Out  	int
}

func (g *Graph) JointDegreeDistrib() (map[DegreePair]float64) {

	inDegrees := make(map[string]int)
	outDegrees := make(map[string]int)
	for vName := range g.Nodes {
		vIn, _ := g.InDegree(vName)
		vOut, _ := g.OutDegree(vName)
		inDegrees[vName] = vIn
		outDegrees[vName] = vOut
	}

	// Collect pair counts
	pairCounts := make(map[DegreePair]int)
	for vName := range g.Nodes {
		pair := DegreePair {
			In: inDegrees[vName],
			Out: outDegrees[vName],
		}
		pairCounts[pair]++
	}

	// Calc probabilities
	N := float64(len(g.Nodes))
	jointDistrib := make(map[DegreePair]float64)
	for pair, count := range pairCounts {
		jointDistrib[pair] = float64(count) / N
	}

	return jointDistrib

}

func helperUnionMaps(map1 map[string]*Node, map2 map[string]*Node) (map[string]int) {

	union := make(map[string]int)
	for key := range map1 {
		union[key] = 1
	}

	for key := range map2 {
		union[key] = 1
	}

	return union

}

func helperIntersectMaps(map1 map[string]*Node, map2 map[string]*Node) (map[string]int) {

	intersect := make(map[string]int)
	for key, _ := range map1 {
		if _, ok := map2[key]; ok {
			intersect[key] = 1
		}
	}

	return intersect

}

func (g *Graph) CountDirectedTriangles(nodes []string) (map[string][]int) {

	if nodes == nil {
		nodes = slices.Collect(maps.Keys(g.Nodes))
	}
	var nodes_list []*Node
	for _, key := range nodes {
		if node, ok := g.Nodes[key]; ok {
			nodes_list = append(nodes_list, node)
		}
	}

	results := make(map[string][]int)

	for _, i := range nodes_list {

		triangle_count := 0

		for jName := range helperUnionMaps(i.InNeighbours, i.OutNeighbours) {
			j := g.Nodes[jName]
			triangle_count += len(helperIntersectMaps(i.InNeighbours, j.InNeighbours)) // k→i and k→j
			triangle_count += len(helperIntersectMaps(i.InNeighbours, j.OutNeighbours)) // k→i and j→k
			triangle_count += len(helperIntersectMaps(i.OutNeighbours, j.InNeighbours)) // i→k and k→j
			triangle_count += len(helperIntersectMaps(i.OutNeighbours, j.OutNeighbours)) // i→k and j→k
		}

		degTotal := len(i.InNeighbours) + len(i.OutNeighbours)
		degBidirect := len(helperIntersectMaps(i.InNeighbours, i.OutNeighbours))
		results[i.Name] = []int{triangle_count, degTotal, degBidirect}

	}

	return results

}

func (g *Graph) LocalClusteringCoeff(nodes []string) (map[string]float64) {

	if nodes == nil {
		nodes = slices.Collect(maps.Keys(g.Nodes))
	}
	var nodes_list []*Node
	for _, key := range nodes {
		if node, ok := g.Nodes[key]; ok {
			nodes_list = append(nodes_list, node)
		}
	}

	trianglesInfo := g.CountDirectedTriangles(nil)
	results := make(map[string]float64)

	for _, i := range nodes_list {

		iInfo := trianglesInfo[i.Name]
		tcount, dtot, dbi := iInfo[0], iInfo[1], iInfo[2]

		if tcount == 0 {
			results[i.Name] = 0.0
		} else {
			denom := (dtot * (dtot - 1) - 2 * dbi) * 2
			results[i.Name] = float64(tcount) / float64(denom)
		}
	}

	return results

}

func (g *Graph) GlobalTransitivity() float64 {

	trianglesInfo := g.CountDirectedTriangles(nil)

	totalTriangles := 0
	totalPossible := 0

	for _, iInfo := range trianglesInfo {
		tCount, dtot, dbi := iInfo[0], iInfo[1], iInfo[2]
		totalTriangles += tCount
		totalPossible += (dtot * (dtot - 1) - 2 * dbi) * 2
	}

	if totalPossible == 0 {
		return 0.0
	}

	return float64(totalTriangles) / float64(totalPossible)

}

func (g *Graph) AvgClusteringCoeff() float64 {
	coeffs := g.LocalClusteringCoeff(nil)
	sum := 0.0

	for i := range coeffs {
		sum += coeffs[i]
	}

	return sum / float64(len(coeffs))
}