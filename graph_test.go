package rscore

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func loadGraph(t *testing.T, nodesPath, edgesPath string) *Graph {
	t.Helper()

	g, err := CreateGraph("Riot Infrastructure", false)
	if err != nil {
		t.Fatalf("CreateGraph: %v", err)
	}

	// Load nodes
	nf, err := os.Open(nodesPath)
	if err != nil {
		t.Fatalf("open nodes csv: %v", err)
	}
	defer nf.Close()

	nr := csv.NewReader(nf)
	rows, err := nr.ReadAll()
	if err != nil {
		t.Fatalf("read nodes csv: %v", err)
	}
	for _, row := range rows[1:] { // skip header
		if _, err := g.AddNode(strings.TrimSpace(row[0]), strings.TrimSpace(row[1])); err != nil {
			t.Fatalf("AddNode %q: %v", row[0], err)
		}
	}

	// Load edges
	ef, err := os.Open(edgesPath)
	if err != nil {
		t.Fatalf("open edges csv: %v", err)
	}
	defer ef.Close()

	er := csv.NewReader(ef)
	rows, err = er.ReadAll()
	if err != nil {
		t.Fatalf("read edges csv: %v", err)
	}
	for _, row := range rows[1:] { // skip header
		edgeType := strings.TrimSpace(row[0])
		from := strings.TrimSpace(row[1])
		to := strings.TrimSpace(row[2])
		name := fmt.Sprintf("%s->%s", from, to)
		if _, err := g.AddEdge(name, edgeType, from, to); err != nil {
			t.Fatalf("AddEdge %q: %v", name, err)
		}
	}

	return g
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreateGraph(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		g, err := CreateGraph("test-graph", false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if g.Name != "test-graph" {
			t.Errorf("expected name %q, got %q", "test-graph", g.Name)
		}
		if g.Nodes == nil || g.Edges == nil {
			t.Error("Nodes and Edges maps should be initialised")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := CreateGraph("", false)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
}

func TestAddNode(t *testing.T) {
	g, _ := CreateGraph("test", false)

	t.Run("add new node", func(t *testing.T) {
		n, err := g.AddNode("ServiceA", "functional")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.Name != "ServiceA" {
			t.Errorf("expected name %q, got %q", "ServiceA", n.Name)
		}
		if _, exists := g.Nodes["ServiceA"]; !exists {
			t.Error("node not found in graph after add")
		}
	})

	t.Run("duplicate node", func(t *testing.T) {
		_, err := g.AddNode("ServiceA", "functional")
		if err == nil {
			t.Error("expected error for duplicate node")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := g.AddNode("", "functional")
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
}

func TestAddEdge(t *testing.T) {
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "provider")

	t.Run("add valid edge", func(t *testing.T) {
		e, err := g.AddEdge("A->B", "dependency", "A", "B")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.FromNode.Name != "A" || e.ToNode.Name != "B" {
			t.Errorf("edge nodes incorrect: got %s->%s", e.FromNode.Name, e.ToNode.Name)
		}
		// Check neighbours are wired up
		if _, ok := g.Nodes["A"].OutNeighbours["B"]; !ok {
			t.Error("B should be in A's OutNeighbours")
		}
		if _, ok := g.Nodes["B"].InNeighbours["A"]; !ok {
			t.Error("A should be in B's InNeighbours")
		}
	})

	t.Run("duplicate edge", func(t *testing.T) {
		_, err := g.AddEdge("A->B", "dependency", "A", "B")
		if err == nil {
			t.Error("expected error for duplicate edge")
		}
	})

	t.Run("missing from node", func(t *testing.T) {
		_, err := g.AddEdge("", "dependency", "Ghost", "B")
		if err == nil {
			t.Error("expected error for missing source node")
		}
	})

	t.Run("missing to node", func(t *testing.T) {
		_, err := g.AddEdge("", "dependency", "A", "Ghost")
		if err == nil {
			t.Error("expected error for missing target node")
		}
	})

	t.Run("auto name", func(t *testing.T) {
		g.AddNode("C", "functional")
		e, err := g.AddEdge("", "dependency", "A", "C")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Name != "A->C" {
			t.Errorf("expected auto name %q, got %q", "A->C", e.Name)
		}
	})
}

func TestRemoveEdge(t *testing.T) {
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "provider")
	g.AddEdge("A->B", "dependency", "A", "B")

	t.Run("remove existing edge", func(t *testing.T) {
		if err := g.RemoveEdge("A->B"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, exists := g.Edges["A->B"]; exists {
			t.Error("edge should be removed from graph")
		}
		if _, ok := g.Nodes["A"].OutNeighbours["B"]; ok {
			t.Error("B should be removed from A's OutNeighbours")
		}
		if _, ok := g.Nodes["B"].InNeighbours["A"]; ok {
			t.Error("A should be removed from B's InNeighbours")
		}
	})

	t.Run("remove non-existent edge", func(t *testing.T) {
		if err := g.RemoveEdge("ghost-edge"); err == nil {
			t.Error("expected error for non-existent edge")
		}
	})
}

func TestRemoveNode(t *testing.T) {
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	g.AddNode("C", "provider")
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "hosted-on", "B", "C")

	t.Run("remove connected node", func(t *testing.T) {
		if err := g.RemoveNode("B"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, exists := g.Nodes["B"]; exists {
			t.Error("node B should be removed")
		}
		// Both edges touching B should be gone
		if _, exists := g.Edges["A->B"]; exists {
			t.Error("edge A->B should be removed")
		}
		if _, exists := g.Edges["B->C"]; exists {
			t.Error("edge B->C should be removed")
		}
		// A and C's neighbour maps should be clean
		if _, ok := g.Nodes["A"].OutNeighbours["B"]; ok {
			t.Error("B should be removed from A's OutNeighbours")
		}
		if _, ok := g.Nodes["C"].InNeighbours["B"]; ok {
			t.Error("B should be removed from C's InNeighbours")
		}
	})

	t.Run("remove non-existent node", func(t *testing.T) {
		if err := g.RemoveNode("Ghost"); err == nil {
			t.Error("expected error for non-existent node")
		}
	})
}

func TestNodeCount(t *testing.T) {
	g, _ := CreateGraph("test", false)
	if g.NodeCount() != 0 {
		t.Errorf("expected 0, got %d", g.NodeCount())
	}
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	if g.NodeCount() != 2 {
		t.Errorf("expected 2, got %d", g.NodeCount())
	}
	g.RemoveNode("A")
	if g.NodeCount() != 1 {
		t.Errorf("expected 1 after removal, got %d", g.NodeCount())
	}
}

func TestEdgeCount(t *testing.T) {
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	if g.EdgeCount() != 0 {
		t.Errorf("expected 0, got %d", g.EdgeCount())
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	if g.EdgeCount() != 1 {
		t.Errorf("expected 1, got %d", g.EdgeCount())
	}
	g.RemoveEdge("A->B")
	if g.EdgeCount() != 0 {
		t.Errorf("expected 0 after removal, got %d", g.EdgeCount())
	}
}

// ── Integration: full Riot graph ──────────────────────────────────────────────

func TestRiotGraph(t *testing.T) {
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	t.Run("node count", func(t *testing.T) {
		if g.NodeCount() != 13 {
			t.Errorf("expected 13 nodes, got %d", g.NodeCount())
		}
	})

	t.Run("edge count", func(t *testing.T) {
		if g.EdgeCount() != 26 {
			t.Errorf("expected 26 edges, got %d", g.EdgeCount())
		}
	})

	t.Run("riot direct has 7 in-neighbours", func(t *testing.T) {
		rd := g.Nodes["Riot Direct"]
		if rd == nil {
			t.Fatal("Riot Direct node not found")
		}
		if len(rd.InNeighbours) != 7 {
			t.Errorf("expected 7 in-neighbours, got %d", len(rd.InNeighbours))
		}
	})

	t.Run("riotso has 4 aws regions as out-neighbours", func(t *testing.T) {
		sso := g.Nodes["RiotSignOn"]
		if sso == nil {
			t.Fatal("RiotSignOn node not found")
		}
		awsCount := 0
		for name := range sso.OutNeighbours {
			if strings.HasPrefix(name, "AWS") {
				awsCount++
			}
		}
		if awsCount != 4 {
			t.Errorf("expected 4 AWS region neighbours, got %d", awsCount)
		}
	})

	t.Run("add and remove a test node", func(t *testing.T) {
		before := g.NodeCount()
		g.AddNode("TestService", "functional")
		if g.NodeCount() != before+1 {
			t.Error("node count should increase by 1 after add")
		}
		g.RemoveNode("TestService")
		if g.NodeCount() != before {
			t.Error("node count should return to original after remove")
		}
	})

	t.Run("add and remove a test edge", func(t *testing.T) {
		before := g.EdgeCount()
		g.AddEdge("", "dependency", "Matchmaking", "Loot Service")
		if g.EdgeCount() != before+1 {
			t.Error("edge count should increase by 1 after add")
		}
		g.RemoveEdge("Matchmaking->Loot Service")
		if g.EdgeCount() != before {
			t.Error("edge count should return to original after remove")
		}
	})

	t.Run("print full graph", func(t *testing.T) {
		g.Print()
	})
}

// ── Print helper (registered on Graph) ───────────────────────────────────────

func (g *Graph) Print() {
	fmt.Printf("\nGraph: %s (weighted=%v, nodes=%d, edges=%d)\n",
		g.Name, g.Weighted, g.NodeCount(), g.EdgeCount())
	fmt.Println(strings.Repeat("─", 54))

	nodeNames := make([]string, 0, len(g.Nodes))
	for name := range g.Nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	for _, name := range nodeNames {
		node := g.Nodes[name]
		fmt.Printf("\n  ┌─ [%s] %s\n", node.Type, node.Name)
		if node.Betweenness > 0 {
			fmt.Printf("  │  betweenness: %.4f\n", node.Betweenness)
		}

		if len(node.OutNeighbours) == 0 {
			fmt.Printf("  │  (no outgoing edges)\n")
			continue
		}

		outNames := make([]string, 0, len(node.OutNeighbours))
		for n := range node.OutNeighbours {
			outNames = append(outNames, n)
		}
		sort.Strings(outNames)

		fmt.Printf("  │\n")
		for i, neighbourName := range outNames {
			edgeKey := fmt.Sprintf("%s->%s", name, neighbourName)
			connector := "├──▶"
			if i == len(outNames)-1 {
				connector = "└──▶"
			}
			if edge, ok := g.Edges[edgeKey]; ok {
				fmt.Printf("  │  %s [%s] %s\n", connector, edge.Type, neighbourName)
			} else {
				fmt.Printf("  │  %s %s\n", connector, neighbourName)
			}
		}
	}
	fmt.Println("\n" + strings.Repeat("─", 54))
}
