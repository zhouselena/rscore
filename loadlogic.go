package rscore

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

var AppInfraGraph *Graph

// description
func Load(nodespath string, edgespath string) (*Graph, error) {

	if (nodespath == "") {
		nodespath = "public/templates/nodes.csv"
	}
	
	if (edgespath == "") {
		edgespath = "public/templates/edges.csv"
	}

	g, err := CreateGraph("AppInfrastructure", false)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate graph: %w", err)
	}

	// Nodes

	nodesFile, err := os.Open(nodespath)
	if err != nil {
		return nil, fmt.Errorf("failed to open nodes file %q: %w", nodespath, err)
	}
	defer nodesFile.Close()

	nodesReader := csv.NewReader(nodesFile)
	if _, err := nodesReader.Read(); err != nil { // skip header
		return nil, fmt.Errorf("failed to read nodes header: %w", err)
	}
	nodesRecords, err := nodesReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse nodes csv; %w", err)
	}

	for i, record := range nodesRecords {
		if len(record) < 2 {
			return nil, fmt.Errorf("nodes csv line %d: expected at least 2 fields, got %d", i+2, len(record))
		}
		name := strings.TrimSpace(record[0])
		nodeType := strings.TrimSpace(record[1])
		serviceTier := ""
		if len(record) >= 3 {
			serviceTier = strings.TrimSpace(record[2])
		}
		node, err := g.AddNode(name, nodeType)
		if err != nil {
			return nil, fmt.Errorf("nodes csv line %d: failed to add node: %w", i+2, err)
		}
		node.ServiceTier = serviceTier
	}

	log.Printf("Loaded %d nodes from %q", g.NodeCount(), nodespath)

	// Edges

	edgesFile, err := os.Open(edgespath)
	if err != nil {
		return nil, fmt.Errorf("failed to open edges file %q: %w", edgespath, err)
	}
	defer edgesFile.Close()

	edgesReader := csv.NewReader(edgesFile)
	if _, err := edgesReader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read edges header: %w", err)
	}
	edgesRecords, err := edgesReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse edges csv: %w", err)
	}

	for i, record := range edgesRecords {
		if len(record) < 3 {
			return nil, fmt.Errorf("edges csv line %d: expected at least 3 fields, got %d", i+2, len(record))
		}
		edgeType := strings.TrimSpace(record[0])
		from := strings.TrimSpace(record[1])
		to := strings.TrimSpace(record[2])
		name := fmt.Sprintf("%s->%s", from, to)
		if _, err := g.AddEdge(name, edgeType, from, to); err != nil {
			return nil, fmt.Errorf("edges csv line %d: failed to add edge %q: %w", i+2, name, err)
		}
	}

	log.Printf("Loaded %d edges from %q", g.EdgeCount(), edgespath)

	if err := LoadTechScores(g, "public/templates/providers.csv"); err != nil {
        log.Printf("Warning: could not load tech scores: %v", err)
    }

	AppInfraGraph = g
	return g, nil
}

func LoadTechScores(g *Graph, providerspath string) error {
    f, err := os.Open(providerspath)
    if err != nil {
        return fmt.Errorf("failed to open providers file %q: %w", providerspath, err)
    }
    defer f.Close()

    r := csv.NewReader(f)
    if _, err := r.Read(); err != nil { // skip header
        return fmt.Errorf("failed to read providers header: %w", err)
    }
    records, err := r.ReadAll()
    if err != nil {
        return fmt.Errorf("failed to parse providers csv: %w", err)
    }

    // Build lookup: "Service/Tier" -> score
    lookup := make(map[string]float64)
    for _, record := range records {
        if len(record) < 3 {
            continue
        }
        tier := strings.TrimSpace(record[1])
        score, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
        if err != nil {
            continue
        }
        lookup[tier] = score
    }

    // Assign scores to nodes
    for _, node := range g.Nodes {
        if node.Type == "functional" || node.ServiceTier == "" {
            node.TechScore = 1.0 // functional nodes are not penalized
            continue
        }
        if score, ok := lookup[node.ServiceTier]; ok {
            node.TechScore = score
        } else {
            log.Printf("Warning: no tech score found for service tier %q on node %q", node.ServiceTier, node.Name)
            node.TechScore = 0.5 // neutral default if tier not found
        }
    }

    // Compute graph-level weighted average (providers weighted 2x)
    weightedSum := 0.0
    totalWeight := 0.0
    for _, node := range g.Nodes {
        w := 1.0
        if node.Type == "provider" {
            w = 2.0
        }
        weightedSum += w * node.TechScore
        totalWeight += w
    }
    if totalWeight > 0 {
        g.AvgTechScore = weightedSum / totalWeight
    }

    log.Printf("Tech scores loaded for %d nodes, graph AvgTechScore=%.3f", len(g.Nodes), g.AvgTechScore)
    return nil
}