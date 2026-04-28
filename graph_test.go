package rscore

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"testing"
)

// ── CLI formatting helpers ────────────────────────────────────────────────────

const (
	clrReset  = "\033[0m"
	clrGreen  = "\033[32m"
	clrRed    = "\033[31m"
	clrCyan   = "\033[36m"
	clrYellow = "\033[33m"
)

// section prints a top-level section header (e.g. "Graph Construction").
func section(label string) {
	fmt.Printf("\n%s══ %s %s\n", clrCyan, label, clrReset)
}

// init prints a short indented line describing something being initialised.
func init_(label string) {
	fmt.Printf("  %s▸ init:%s %s\n", clrYellow, clrReset, label)
}

// pass / fail are called by the result helpers below.
func pass(t *testing.T, label string) {
	t.Helper()
	fmt.Printf("    %s✔%s %s\n", clrGreen, clrReset, label)
}

func fail(t *testing.T, label, msg string) {
	t.Helper()
	fmt.Printf("    %s✘%s %s — %s\n", clrRed, clrReset, label, msg)
	t.Fail()
}

// check is a one-liner: if cond is false it calls fail, otherwise pass.
func check(t *testing.T, cond bool, label, failMsg string) {
	t.Helper()
	if cond {
		pass(t, label)
	} else {
		fail(t, label, failMsg)
	}
}

// ── Graph loader ──────────────────────────────────────────────────────────────

func loadGraph(t *testing.T, nodesPath, edgesPath string) *Graph {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	init_(fmt.Sprintf("graph from %s + %s", nodesPath, edgesPath))

	g, err := CreateGraph("Riot Infrastructure", false)
	if err != nil {
		t.Fatalf("CreateGraph: %v", err)
	}

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
	for _, row := range rows[1:] {
		if _, err := g.AddNode(strings.TrimSpace(row[0]), strings.TrimSpace(row[1])); err != nil {
			t.Fatalf("AddNode %q: %v", row[0], err)
		}
	}

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
	for _, row := range rows[1:] {
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

// approxEqual returns true when |a-b| < tol.
func approxEqual(a, b, tol float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < tol
}

// ── Graph construction ────────────────────────────────────────────────────────

func TestCreateGraph(t *testing.T) {
	section("Graph Construction")

	t.Run("valid name", func(t *testing.T) {
		init_("CreateGraph(\"test-graph\", false)")
		g, err := CreateGraph("test-graph", false)
		check(t, err == nil, "no error returned", fmt.Sprintf("got %v", err))
		if err == nil {
			check(t, g.Name == "test-graph", "name is set correctly", fmt.Sprintf("got %q", g.Name))
			check(t, g.Nodes != nil && g.Edges != nil, "Nodes and Edges maps initialised", "one or both are nil")
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		_, err := CreateGraph("", false)
		check(t, err != nil, "error returned for empty name", "expected an error but got nil")
	})
}

// ── Node operations ───────────────────────────────────────────────────────────

func TestAddNode(t *testing.T) {
	section("Node Operations — AddNode")
	init_("CreateGraph(\"test\", false)")
	g, _ := CreateGraph("test", false)

	t.Run("add new node", func(t *testing.T) {
		init_("AddNode(\"ServiceA\", \"functional\")")
		n, err := g.AddNode("ServiceA", "functional")
		check(t, err == nil, "no error returned", fmt.Sprintf("got %v", err))
		if err == nil {
			check(t, n.Name == "ServiceA", "node name is set", fmt.Sprintf("got %q", n.Name))
			_, exists := g.Nodes["ServiceA"]
			check(t, exists, "node present in graph map", "missing after add")
		}
	})

	t.Run("duplicate node rejected", func(t *testing.T) {
		_, err := g.AddNode("ServiceA", "functional")
		check(t, err != nil, "error returned for duplicate", "expected an error but got nil")
	})

	t.Run("empty name rejected", func(t *testing.T) {
		_, err := g.AddNode("", "functional")
		check(t, err != nil, "error returned for empty name", "expected an error but got nil")
	})
}

// ── Edge operations ───────────────────────────────────────────────────────────

func TestAddEdge(t *testing.T) {
	section("Edge Operations — AddEdge")
	init_("graph with nodes A (functional) and B (provider)")
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "provider")

	t.Run("add valid edge", func(t *testing.T) {
		init_("AddEdge(\"A->B\", \"dependency\", \"A\", \"B\")")
		e, err := g.AddEdge("A->B", "dependency", "A", "B")
		check(t, err == nil, "no error returned", fmt.Sprintf("got %v", err))
		if err == nil {
			check(t, e.FromNode.Name == "A" && e.ToNode.Name == "B",
				"edge endpoints are correct",
				fmt.Sprintf("got %s->%s", e.FromNode.Name, e.ToNode.Name))
			_, okOut := g.Nodes["A"].OutNeighbours["B"]
			check(t, okOut, "B in A's OutNeighbours", "missing")
			_, okIn := g.Nodes["B"].InNeighbours["A"]
			check(t, okIn, "A in B's InNeighbours", "missing")
		}
	})

	t.Run("duplicate edge rejected", func(t *testing.T) {
		_, err := g.AddEdge("A->B", "dependency", "A", "B")
		check(t, err != nil, "error returned for duplicate edge", "expected an error but got nil")
	})

	t.Run("missing from-node rejected", func(t *testing.T) {
		_, err := g.AddEdge("", "dependency", "Ghost", "B")
		check(t, err != nil, "error returned for missing source node", "expected an error but got nil")
	})

	t.Run("missing to-node rejected", func(t *testing.T) {
		_, err := g.AddEdge("", "dependency", "A", "Ghost")
		check(t, err != nil, "error returned for missing target node", "expected an error but got nil")
	})

	t.Run("auto-name when name is empty", func(t *testing.T) {
		g.AddNode("C", "functional")
		e, err := g.AddEdge("", "dependency", "A", "C")
		check(t, err == nil, "no error returned", fmt.Sprintf("got %v", err))
		if err == nil {
			check(t, e.Name == "A->C", "auto-name is \"A->C\"", fmt.Sprintf("got %q", e.Name))
		}
	})
}

func TestRemoveEdge(t *testing.T) {
	section("Edge Operations — RemoveEdge")
	init_("graph with A->B edge")
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "provider")
	g.AddEdge("A->B", "dependency", "A", "B")

	t.Run("remove existing edge", func(t *testing.T) {
		err := g.RemoveEdge("A->B")
		check(t, err == nil, "no error returned", fmt.Sprintf("got %v", err))
		_, inMap := g.Edges["A->B"]
		check(t, !inMap, "edge removed from graph map", "still present")
		_, okOut := g.Nodes["A"].OutNeighbours["B"]
		check(t, !okOut, "B removed from A's OutNeighbours", "still present")
		_, okIn := g.Nodes["B"].InNeighbours["A"]
		check(t, !okIn, "A removed from B's InNeighbours", "still present")
	})

	t.Run("non-existent edge rejected", func(t *testing.T) {
		err := g.RemoveEdge("ghost-edge")
		check(t, err != nil, "error returned for non-existent edge", "expected an error but got nil")
	})
}

func TestRemoveNode(t *testing.T) {
	section("Node Operations — RemoveNode")
	init_("graph: A->B->C")
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	g.AddNode("C", "provider")
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "hosted-on", "B", "C")

	t.Run("remove connected node cascades to edges", func(t *testing.T) {
		err := g.RemoveNode("B")
		check(t, err == nil, "no error returned", fmt.Sprintf("got %v", err))
		_, nodeGone := g.Nodes["B"]
		check(t, !nodeGone, "node B removed from graph", "still present")
		_, e1Gone := g.Edges["A->B"]
		check(t, !e1Gone, "edge A->B removed", "still present")
		_, e2Gone := g.Edges["B->C"]
		check(t, !e2Gone, "edge B->C removed", "still present")
		_, okOut := g.Nodes["A"].OutNeighbours["B"]
		check(t, !okOut, "B removed from A's OutNeighbours", "still present")
		_, okIn := g.Nodes["C"].InNeighbours["B"]
		check(t, !okIn, "B removed from C's InNeighbours", "still present")
	})

	t.Run("non-existent node rejected", func(t *testing.T) {
		err := g.RemoveNode("Ghost")
		check(t, err != nil, "error returned for non-existent node", "expected an error but got nil")
	})
}

// ── Count helpers ─────────────────────────────────────────────────────────────

func TestNodeCount(t *testing.T) {
	section("Count Helpers — NodeCount")
	init_("empty graph")
	g, _ := CreateGraph("test", false)
	check(t, g.NodeCount() == 0, "empty graph has 0 nodes", fmt.Sprintf("got %d", g.NodeCount()))
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	check(t, g.NodeCount() == 2, "count is 2 after two adds", fmt.Sprintf("got %d", g.NodeCount()))
	g.RemoveNode("A")
	check(t, g.NodeCount() == 1, "count is 1 after one removal", fmt.Sprintf("got %d", g.NodeCount()))
}

func TestEdgeCount(t *testing.T) {
	section("Count Helpers — EdgeCount")
	init_("graph with A and B")
	g, _ := CreateGraph("test", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	check(t, g.EdgeCount() == 0, "empty graph has 0 edges", fmt.Sprintf("got %d", g.EdgeCount()))
	g.AddEdge("A->B", "dependency", "A", "B")
	check(t, g.EdgeCount() == 1, "count is 1 after add", fmt.Sprintf("got %d", g.EdgeCount()))
	g.RemoveEdge("A->B")
	check(t, g.EdgeCount() == 0, "count is 0 after removal", fmt.Sprintf("got %d", g.EdgeCount()))
}

// ── Integration: full Riot graph ──────────────────────────────────────────────

func TestRiotGraph(t *testing.T) {
	section("Integration — Riot Infrastructure Graph")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	t.Run("node count is 13", func(t *testing.T) {
		check(t, g.NodeCount() == 13,
			"graph has 13 nodes", fmt.Sprintf("got %d", g.NodeCount()))
	})

	t.Run("edge count is 26", func(t *testing.T) {
		check(t, g.EdgeCount() == 26,
			"graph has 26 edges", fmt.Sprintf("got %d", g.EdgeCount()))
	})

	t.Run("Riot Direct has 7 in-neighbours", func(t *testing.T) {
		rd := g.Nodes["Riot Direct"]
		if rd == nil {
			t.Fatal("Riot Direct node not found")
		}
		check(t, len(rd.InNeighbours) == 7,
			"Riot Direct has 7 in-neighbours",
			fmt.Sprintf("got %d", len(rd.InNeighbours)))
	})

	t.Run("RiotSignOn has 4 AWS region out-neighbours", func(t *testing.T) {
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
		check(t, awsCount == 4,
			"RiotSignOn has 4 AWS region out-neighbours",
			fmt.Sprintf("got %d", awsCount))
	})

	t.Run("add and remove a node", func(t *testing.T) {
		before := g.NodeCount()
		g.AddNode("TestService", "functional")
		check(t, g.NodeCount() == before+1, "count +1 after add", fmt.Sprintf("got %d", g.NodeCount()))
		g.RemoveNode("TestService")
		check(t, g.NodeCount() == before, "count restored after remove", fmt.Sprintf("got %d", g.NodeCount()))
	})

	t.Run("add and remove an edge", func(t *testing.T) {
		before := g.EdgeCount()
		g.AddEdge("", "dependency", "Matchmaking", "Loot Service")
		check(t, g.EdgeCount() == before+1, "count +1 after add", fmt.Sprintf("got %d", g.EdgeCount()))
		g.RemoveEdge("Matchmaking->Loot Service")
		check(t, g.EdgeCount() == before, "count restored after remove", fmt.Sprintf("got %d", g.EdgeCount()))
	})

	t.Run("print full graph", func(t *testing.T) {
		g.Print()
	})
}

// ── Print helper ──────────────────────────────────────────────────────────────

func (g *Graph) Print() {
	fmt.Printf("\n%sGraph: %s%s  (weighted=%v  nodes=%d  edges=%d)\n",
		clrCyan, g.Name, clrReset, g.Weighted, g.NodeCount(), g.EdgeCount())
	fmt.Println(strings.Repeat("─", 54))

	nodeNames := make([]string, 0, len(g.Nodes))
	for name := range g.Nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	for _, name := range nodeNames {
		node := g.Nodes[name]
		fmt.Printf("\n  ┌─ %s[%s]%s %s\n", clrYellow, node.Type, clrReset, node.Name)
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
				fmt.Printf("  │  %s %s[%s]%s %s\n", connector, clrYellow, edge.Type, clrReset, neighbourName)
			} else {
				fmt.Printf("  │  %s %s\n", connector, neighbourName)
			}
		}
	}
	fmt.Println("\n" + strings.Repeat("─", 54))
}

// ── Betweenness Centrality ────────────────────────────────────────────────────

func TestBetweennessCentrality_Simple(t *testing.T) {
	section("Betweenness Centrality — Simple Chain (A→B→C)")
	init_("linear chain: A->B->C")
	g, _ := CreateGraph("chain", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	g.AddNode("C", "functional")
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")

	scores := g.BetweennessCentrality(false)

	check(t, approxEqual(scores["A"], 0.0, 1e-9), "A = 0.0 (endpoint)", fmt.Sprintf("got %f", scores["A"]))
	check(t, approxEqual(scores["B"], 1.0, 1e-9), "B = 1.0 (only intermediary)", fmt.Sprintf("got %f", scores["B"]))
	check(t, approxEqual(scores["C"], 0.0, 1e-9), "C = 0.0 (endpoint)", fmt.Sprintf("got %f", scores["C"]))
}

func TestBetweennessCentrality_Normalised(t *testing.T) {
	section("Betweenness Centrality — Normalised (N=3, scale=0.5)")
	init_("same chain, normalised=true")
	g, _ := CreateGraph("chain", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")
	g.AddNode("C", "functional")
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")

	scores := g.BetweennessCentrality(true)
	check(t, approxEqual(scores["B"], 0.5, 1e-9), "B normalised = 0.5", fmt.Sprintf("got %f", scores["B"]))
}

func TestBetweennessCentrality_Disconnected(t *testing.T) {
	section("Betweenness Centrality — Disconnected Graph")
	init_("two isolated nodes A and B")
	g, _ := CreateGraph("disconnected", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")

	scores := g.BetweennessCentrality(false)
	for name, score := range scores {
		check(t, approxEqual(score, 0.0, 1e-9),
			fmt.Sprintf("%s = 0.0 (isolated)", name),
			fmt.Sprintf("got %f", score))
	}
}

func TestBetweennessCentrality_RiotGraph_Unnormalised(t *testing.T) {
	section("Betweenness Centrality — Riot Graph (unnormalised, ground truth via NetworkX)")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	scores := g.BetweennessCentrality(false)

	expected := map[string]float64{
		"RiotSignOn":             24.0,
		"Matchmaking":            4.5,
		"Riot Direct":            4.0,
		"rCluster":               2.5,
		"Player Profile":         2.0,
		"Loot Service":           0.0,
		"Riot Messaging Service": 0.0,
		"OpenContrail SDN":       0.0,
		"Riot-owned provider":    0.0,
		"AWS Region 1":           0.0,
		"AWS Region 2":           0.0,
		"AWS Region 3":           0.0,
		"AWS Region 4":           0.0,
	}

	for name, want := range expected {
		got, ok := scores[name]
		check(t, ok, fmt.Sprintf("%s present in scores map", name), "missing")
		if ok {
			check(t, approxEqual(got, want, 1e-4),
				fmt.Sprintf("%-25s = %.4f", name, want),
				fmt.Sprintf("got %.6f", got))
		}
	}
}

func TestBetweennessCentrality_RiotGraph_Normalised(t *testing.T) {
	section("Betweenness Centrality — Riot Graph (normalised, ÷132)")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	scores := g.BetweennessCentrality(true)

	expected := map[string]float64{
		"RiotSignOn":             0.181818,
		"Matchmaking":            0.034091,
		"Riot Direct":            0.030303,
		"rCluster":               0.018939,
		"Player Profile":         0.015152,
		"Loot Service":           0.0,
		"Riot Messaging Service": 0.0,
		"OpenContrail SDN":       0.0,
		"Riot-owned provider":    0.0,
		"AWS Region 1":           0.0,
		"AWS Region 2":           0.0,
		"AWS Region 3":           0.0,
		"AWS Region 4":           0.0,
	}

	for name, want := range expected {
		got, ok := scores[name]
		check(t, ok, fmt.Sprintf("%s present in scores map", name), "missing")
		if ok {
			check(t, approxEqual(got, want, 1e-4),
				fmt.Sprintf("%-25s = %.6f", name, want),
				fmt.Sprintf("got %.6f", got))
		}
	}
}

func TestBetweennessCentrality_RiotGraph_Ranking(t *testing.T) {
	section("Betweenness Centrality — Riot Graph (top-5 ranking)")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	scores := g.BetweennessCentrality(false)

	type kv struct {
		Name  string
		Score float64
	}
	ranked := make([]kv, 0, len(scores))
	for name, score := range scores {
		ranked = append(ranked, kv{name, score})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })

	expectedOrder := []string{"RiotSignOn", "Matchmaking", "Riot Direct", "rCluster", "Player Profile"}
	for i, want := range expectedOrder {
		check(t, ranked[i].Name == want,
			fmt.Sprintf("rank %d is %q", i+1, want),
			fmt.Sprintf("got %q (score=%.4f)", ranked[i].Name, ranked[i].Score))
	}
}

// ── Joint Degree Distribution ─────────────────────────────────────────────────

// buildJointGraph returns a small hand-crafted graph:
//
//	A ──▶ B ──▶ C
//	      │
//	      ▼
//	      D
//
// Expected distribution: {0,1}=0.25  {1,2}=0.25  {1,0}=0.50
func buildJointGraph(t *testing.T) *Graph {
	t.Helper()
	init_("joint graph: A->B->C, B->D")
	g, _ := CreateGraph("joint-test", false)
	for _, n := range []string{"A", "B", "C", "D"} {
		g.AddNode(n, "functional")
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")
	g.AddEdge("B->D", "dependency", "B", "D")
	return g
}

func TestJointDegreeDistrib_Simple(t *testing.T) {
	section("Joint Degree Distribution — Simple Graph (A→B→C,D)")
	g := buildJointGraph(t)
	dist := g.JointDegreeDistrib()

	check(t, dist != nil, "returns non-nil map", "got nil")
	check(t, len(dist) == 3, "3 distinct degree pairs", fmt.Sprintf("got %d", len(dist)))

	cases := []struct {
		pair DegreePair
		want float64
		desc string
	}{
		{DegreePair{In: 0, Out: 1}, 0.25, "A  → {0,1} = 0.25"},
		{DegreePair{In: 1, Out: 2}, 0.25, "B  → {1,2} = 0.25"},
		{DegreePair{In: 1, Out: 0}, 0.50, "C,D → {1,0} = 0.50"},
	}
	for _, tc := range cases {
		got, ok := dist[tc.pair]
		check(t, ok, fmt.Sprintf("pair {%d,%d} present", tc.pair.In, tc.pair.Out), "missing")
		if ok {
			check(t, approxEqual(got, tc.want, 1e-9), tc.desc, fmt.Sprintf("got %.4f", got))
		}
	}

	total := 0.0
	for _, p := range dist {
		total += p
	}
	check(t, approxEqual(total, 1.0, 1e-9), "probabilities sum to 1.0", fmt.Sprintf("got %.10f", total))
}

func TestJointDegreeDistrib_RiotGraph(t *testing.T) {
	section("Joint Degree Distribution — Riot Graph")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)
	dist := g.JointDegreeDistrib()

	check(t, len(dist) == 10, "10 distinct degree pairs", fmt.Sprintf("got %d", len(dist)))

	total := 0.0
	for _, p := range dist {
		total += p
	}
	check(t, approxEqual(total, 1.0, 1e-9), "probabilities sum to 1.0", fmt.Sprintf("got %.10f", total))

	const N = 13.0
	spotChecks := []struct {
		pair DegreePair
		k    float64
		desc string
	}{
		{DegreePair{In: 1, Out: 0}, 4, "AWS regions 1-4 share {1,0}"},
		{DegreePair{In: 7, Out: 1}, 1, "Riot Direct is the only {7,1} node"},
		{DegreePair{In: 6, Out: 5}, 1, "RiotSignOn is the only {6,5} node"},
	}
	for _, tc := range spotChecks {
		want := tc.k / N
		got, ok := dist[tc.pair]
		check(t, ok, fmt.Sprintf("pair {%d,%d} present", tc.pair.In, tc.pair.Out), "missing")
		if ok {
			check(t, approxEqual(got, want, 1e-9), tc.desc, fmt.Sprintf("got %.6f, want %.6f", got, want))
		}
	}
}

// ── Clustering Coefficients ───────────────────────────────────────────────────

// buildClusterGraph returns a hand-crafted graph whose clustering values are
// easy to verify by inspection:
//
//	A ──▶ B ──▶ C
//	 ╲         ╱
//	  ╲──────▶╱
//
// Triangles (directed, per node using the Watts-Strogatz generalisation):
//   A: i→B, i→C; B is not a neighbour of C in the direction that closes a
//      triangle through A, but A→B→C and A→C gives one cycle. Computed below.
//   B: receives from A, sends to C; A also sends to C → triangle A,B,C.
//   C: pure sink, no outgoing edges → 0 triangles.
//
// We use this graph to spot-check CountDirectedTriangles and
// LocalClusteringCoeff; GlobalTransitivity and AvgClusteringCoeff are
// sanity-checked (non-negative, in [0,1]) rather than pinned to exact values
// because they depend on the full triangle/denominator formula in the
// implementation.
func buildClusterGraph(t *testing.T) *Graph {
	t.Helper()
	init_("cluster graph: A->B, B->C, A->C (one directed triangle)")
	g, _ := CreateGraph("cluster-test", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"A", "B", "C"} {
		g.AddNode(n, "functional")
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")
	g.AddEdge("A->C", "dependency", "A", "C")
	return g
}

// buildIsolatedGraph returns three nodes with no edges — every clustering
// metric should be 0.
func buildIsolatedGraph(t *testing.T) *Graph {
	t.Helper()
	init_("isolated graph: three nodes, no edges")
	g, _ := CreateGraph("isolated-test", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"X", "Y", "Z"} {
		g.AddNode(n, "functional")
	}
	return g
}

// ── CountDirectedTriangles ────────────────────────────────────────────────────

func TestCountDirectedTriangles_Simple(t *testing.T) {
	section("CountDirectedTriangles — Simple Triangle Graph")
	g := buildClusterGraph(t)

	results := g.CountDirectedTriangles(nil)

	// Every node must appear in the result map.
	for _, name := range []string{"A", "B", "C"} {
		_, ok := results[name]
		check(t, ok, fmt.Sprintf("%s present in results map", name), "missing")
	}

	// All three nodes participate in the A→B→C / A→C cycle. The algorithm
	// counts triangles at i by checking shared neighbours across all four
	// in/out combinations, so even C (a sink) gets a non-zero count because
	// A and B are both its in-neighbours and are themselves connected.
	check(t, results["C"][0] > 0, "C has > 0 triangles (in-neighbours A and B are connected)", fmt.Sprintf("got %d", results["C"][0]))

	// A and B both participate in the A→B→C / A→C cycle, so each must have > 0.
	check(t, results["A"][0] > 0, "A has at least 1 triangle", fmt.Sprintf("got %d", results["A"][0]))
	check(t, results["B"][0] > 0, "B has at least 1 triangle", fmt.Sprintf("got %d", results["B"][0]))
}

func TestCountDirectedTriangles_Isolated(t *testing.T) {
	section("CountDirectedTriangles — Isolated Nodes")
	g := buildIsolatedGraph(t)

	results := g.CountDirectedTriangles(nil)

	for name, info := range results {
		check(t, info[0] == 0, fmt.Sprintf("%s has 0 triangles (no edges)", name), fmt.Sprintf("got %d", info[0]))
	}
}

func TestCountDirectedTriangles_SubsetFilter(t *testing.T) {
	section("CountDirectedTriangles — Subset Node Filter")
	g := buildClusterGraph(t)

	// Request results for only ["A", "B"] — C must not appear in the map.
	results := g.CountDirectedTriangles([]string{"A", "B"})

	check(t, len(results) == 2, "result contains exactly 2 nodes", fmt.Sprintf("got %d", len(results)))
	_, hasA := results["A"]
	_, hasB := results["B"]
	_, hasC := results["C"]
	check(t, hasA, "A present in filtered results", "missing")
	check(t, hasB, "B present in filtered results", "missing")
	check(t, !hasC, "C absent from filtered results", "unexpectedly present")
}

func TestCountDirectedTriangles_RiotGraph(t *testing.T) {
	section("CountDirectedTriangles — Riot Graph")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	results := g.CountDirectedTriangles(nil)

	// Every node must appear.
	for name := range g.Nodes {
		_, ok := results[name]
		check(t, ok, fmt.Sprintf("%s present in results map", name), "missing")
	}

	// Leaf provider nodes (AWS regions, Riot-owned provider) are pure sinks or
	// sources with no shared neighbours → 0 triangles.
	for _, name := range []string{"AWS Region 1", "AWS Region 2", "AWS Region 3", "AWS Region 4"} {
		info, ok := results[name]
		if ok {
			check(t, info[0] == 0, fmt.Sprintf("%s has 0 triangles (leaf provider)", name), fmt.Sprintf("got %d", info[0]))
		}
	}

	// RiotSignOn sits at the centre of many paths and must have > 0 triangles.
	check(t, results["RiotSignOn"][0] > 0, "RiotSignOn has at least 1 triangle", fmt.Sprintf("got %d", results["RiotSignOn"][0]))
}

// ── LocalClusteringCoeff ──────────────────────────────────────────────────────

func TestLocalClusteringCoeff_Simple(t *testing.T) {
	section("LocalClusteringCoeff — Simple Triangle Graph")
	g := buildClusterGraph(t)

	coeffs := g.LocalClusteringCoeff(nil)

	// All nodes must be present.
	for _, name := range []string{"A", "B", "C"} {
		_, ok := coeffs[name]
		check(t, ok, fmt.Sprintf("%s present in coeffs map", name), "missing")
	}

	// C has non-zero triangles (its in-neighbours A and B are connected), so
	// its clustering coefficient is also non-zero despite having no out-edges.
	check(t, coeffs["C"] > 0.0, "C > 0.0 (in-neighbours form a triangle)", fmt.Sprintf("got %f", coeffs["C"]))

	// All coefficients must be in [0, 1].
	for name, c := range coeffs {
		check(t, c >= 0.0 && c <= 1.0, fmt.Sprintf("%s coeff in [0,1]", name), fmt.Sprintf("got %f", c))
	}
}

func TestLocalClusteringCoeff_Isolated(t *testing.T) {
	section("LocalClusteringCoeff — Isolated Nodes")
	g := buildIsolatedGraph(t)

	coeffs := g.LocalClusteringCoeff(nil)

	for name, c := range coeffs {
		check(t, approxEqual(c, 0.0, 1e-9), fmt.Sprintf("%s = 0.0 (no edges)", name), fmt.Sprintf("got %f", c))
	}
}

func TestLocalClusteringCoeff_RiotGraph(t *testing.T) {
	section("LocalClusteringCoeff — Riot Graph")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	coeffs := g.LocalClusteringCoeff(nil)

	// Every node must have an entry.
	for name := range g.Nodes {
		_, ok := coeffs[name]
		check(t, ok, fmt.Sprintf("%s present in coeffs map", name), "missing")
	}

	// All values must be in [0, 1].
	for name, c := range coeffs {
		check(t, c >= 0.0 && c <= 1.0, fmt.Sprintf("%s coeff in [0,1]", name), fmt.Sprintf("got %f", c))
	}

	// Pure sink nodes with a single provider parent form no triangles → 0.
	for _, name := range []string{"AWS Region 1", "AWS Region 2", "AWS Region 3", "AWS Region 4"} {
		c, ok := coeffs[name]
		if ok {
			check(t, approxEqual(c, 0.0, 1e-9), fmt.Sprintf("%s = 0.0 (leaf)", name), fmt.Sprintf("got %f", c))
		}
	}
}

// ── GlobalTransitivity ────────────────────────────────────────────────────────

func TestGlobalTransitivity_Simple(t *testing.T) {
	section("GlobalTransitivity — Simple Triangle Graph")
	g := buildClusterGraph(t)

	gt := g.GlobalTransitivity()

	check(t, gt >= 0.0 && gt <= 1.0, "transitivity in [0,1]", fmt.Sprintf("got %f", gt))
	// The graph has a closed triangle so transitivity must be strictly positive.
	check(t, gt > 0.0, "transitivity > 0 (closed triangle exists)", fmt.Sprintf("got %f", gt))
}

func TestGlobalTransitivity_Isolated(t *testing.T) {
	section("GlobalTransitivity — Isolated Nodes")
	g := buildIsolatedGraph(t)

	gt := g.GlobalTransitivity()

	check(t, approxEqual(gt, 0.0, 1e-9), "transitivity = 0.0 (no edges)", fmt.Sprintf("got %f", gt))
}

func TestGlobalTransitivity_RiotGraph(t *testing.T) {
	section("GlobalTransitivity — Riot Graph")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	gt := g.GlobalTransitivity()

	check(t, gt >= 0.0 && gt <= 1.0, "transitivity in [0,1]", fmt.Sprintf("got %f", gt))
	// The Riot graph has cycles (e.g. through RiotSignOn), so > 0 is expected.
	check(t, gt > 0.0, "transitivity > 0 (cycles present in Riot graph)", fmt.Sprintf("got %f", gt))
}

// ── AvgClusteringCoeff ────────────────────────────────────────────────────────

// NOTE: AvgClusteringCoeff in graphlogic.go contains a bug on the accumulation
// line: `sum = coeffs[i]` should be `sum += coeffs[i]`. The tests below are
// written against the correct behaviour; they will fail until that is fixed.

func TestAvgClusteringCoeff_Simple(t *testing.T) {
	section("AvgClusteringCoeff — Simple Triangle Graph")
	g := buildClusterGraph(t)

	avg := g.AvgClusteringCoeff()

	check(t, avg >= 0.0 && avg <= 1.0, "avg coeff in [0,1]", fmt.Sprintf("got %f", avg))
	// At least A and B have non-zero coefficients, so the average must be > 0.
	check(t, avg > 0.0, "avg coeff > 0 (triangle graph)", fmt.Sprintf("got %f", avg))

	// Cross-check: avg must equal the mean of LocalClusteringCoeff values.
	coeffs := g.LocalClusteringCoeff(nil)
	sum := 0.0
	for _, c := range coeffs {
		sum += c
	}
	want := sum / float64(len(coeffs))
	check(t, approxEqual(avg, want, 1e-9), fmt.Sprintf("avg matches manual mean (%.6f)", want), fmt.Sprintf("got %f", avg))
}

func TestAvgClusteringCoeff_Isolated(t *testing.T) {
	section("AvgClusteringCoeff — Isolated Nodes")
	g := buildIsolatedGraph(t)

	avg := g.AvgClusteringCoeff()

	check(t, approxEqual(avg, 0.0, 1e-9), "avg coeff = 0.0 (no edges)", fmt.Sprintf("got %f", avg))
}

func TestAvgClusteringCoeff_RiotGraph(t *testing.T) {
	section("AvgClusteringCoeff — Riot Graph")
	g := loadGraph(t,
		"./public/templates/nodes.csv",
		"./public/templates/edges.csv",
	)

	avg := g.AvgClusteringCoeff()

	check(t, avg >= 0.0 && avg <= 1.0, "avg coeff in [0,1]", fmt.Sprintf("got %f", avg))

	// Cross-check against the mean of LocalClusteringCoeff.
	coeffs := g.LocalClusteringCoeff(nil)
	sum := 0.0
	for _, c := range coeffs {
		sum += c
	}
	want := sum / float64(len(coeffs))
	check(t, approxEqual(avg, want, 1e-9), fmt.Sprintf("avg matches manual mean (%.6f)", want), fmt.Sprintf("got %f", avg))
}