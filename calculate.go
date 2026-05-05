package rscore

import (
	"math"
	"slices"
	"fmt"
)

func LoadAllAlgorithms() error {
	if AppInfraGraph == nil {
		return fmt.Errorf("no graph found, run Load first")
	}
	
	AppInfraGraph.Betweenness = BetweennessCentrality(AppInfraGraph, true)
	AppInfraGraph.Entropy = DegreeEntropy(AppInfraGraph)
	AppInfraGraph.LocalClustering = LocalClusteringCoeff(AppInfraGraph, nil)
	AppInfraGraph.Transitivity = GlobalTransitivity(AppInfraGraph)
	AppInfraGraph.AvgClustering = AvgClusteringCoeff(AppInfraGraph)
	AppInfraGraph.ArtPoints = FindArticulationPoints(AppInfraGraph)

	AppInfraGraph.FielderValue = AlgebraicConnectivity(AppInfraGraph)
	return nil
}

func CalculateGraphResiliency() (float64, string) {
	if AppInfraGraph.Betweenness == nil {
		LoadAllAlgorithms()
	}
	n := AppInfraGraph.NodeCount()

	c_connectivity := min(AppInfraGraph.FielderValue / math.Log(float64(n)), 1.0) // λ₂ is unbounded
	c_artpts := 1.0 - (float64(len(AppInfraGraph.ArtPoints)) / float64(n))
	c_clustering := AppInfraGraph.AvgClustering
	bMax := 0.0
	for _, bcScore := range AppInfraGraph.Betweenness {
		if bcScore > bMax {
			bMax = bcScore
		}
	}
	c_betweenness := 1.0 - bMax
	c_degree := AppInfraGraph.Entropy

	w1, w2, w3, w4, w5 := 0.30, 0.25, 0.20, 0.15, 0.10
	rScore := w1 * c_connectivity + w2 * c_artpts + w3 * c_clustering + w4 * c_betweenness + w5 * c_degree

	return rScore, "placeholder for recommendation"
}

func CalculateNodeCriticalness() map[string]float64 {
	if AppInfraGraph.Betweenness == nil {
		LoadAllAlgorithms()
	}

	scores := make(map[string]float64)
	for node := range AppInfraGraph.Nodes {
		k_betweenness := AppInfraGraph.Betweenness[node]
		k_artpt := 0.0
		if slices.Contains(AppInfraGraph.ArtPoints, node) {
			k_artpt = 1.0
		}
		inDeg, outDeg := float64(AppInfraGraph.InDegree(node)), float64(AppInfraGraph.OutDegree(node))
		k_degreeasym :=  inDeg / (inDeg + outDeg + 1)

		w1, w2, w3 := 0.4, 0.35, 0.25
		scores[node] = w1 * k_betweenness + w2 * k_artpt + w3 * k_degreeasym
	}

	return scores
}