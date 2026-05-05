package rscore

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
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
		if _, err := g.AddNode(name, nodeType); err != nil {
			return nil, fmt.Errorf("nodes csv line %d: failed to add node: %w", i+2, err)
		}
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

	AppInfraGraph = g
	return g, nil
}