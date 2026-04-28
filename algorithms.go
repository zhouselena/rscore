package rscore

import (
	"maps"
	"slices"
)

func BetweennessCentrality(g *Graph, normal bool) (map[string]float64) {

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

func JointDegreeDistrib(g *Graph) (map[DegreePair]float64) {

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

func CountDirectedTriangles(g *Graph, nodes []string) (map[string][]int) {

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

func LocalClusteringCoeff(g *Graph, nodes []string) (map[string]float64) {

	if nodes == nil {
		nodes = slices.Collect(maps.Keys(g.Nodes))
	}
	var nodes_list []*Node
	for _, key := range nodes {
		if node, ok := g.Nodes[key]; ok {
			nodes_list = append(nodes_list, node)
		}
	}

	trianglesInfo := CountDirectedTriangles(g, nil)
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

func GlobalTransitivity(g *Graph) float64 {

	trianglesInfo := CountDirectedTriangles(g, nil)

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

func AvgClusteringCoeff(g *Graph) float64 {
	coeffs := LocalClusteringCoeff(g, nil)
	sum := 0.0

	for i := range coeffs {
		sum += coeffs[i]
	}

	return sum / float64(len(coeffs))
}