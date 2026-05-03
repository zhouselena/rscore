package rscore

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"testing"
)

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

	scores := BetweennessCentrality(g, false)

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

	scores := BetweennessCentrality(g, true)
	check(t, approxEqual(scores["B"], 0.5, 1e-9), "B normalised = 0.5", fmt.Sprintf("got %f", scores["B"]))
}

func TestBetweennessCentrality_Disconnected(t *testing.T) {
	section("Betweenness Centrality — Disconnected Graph")
	init_("two isolated nodes A and B")
	g, _ := CreateGraph("disconnected", false)
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")

	scores := BetweennessCentrality(g, false)
	for name, score := range scores {
		check(t, approxEqual(score, 0.0, 1e-9),
			fmt.Sprintf("%s = 0.0 (isolated)", name),
			fmt.Sprintf("got %f", score))
	}
}

func TestBetweennessCentrality_RiotGraph_Unnormalised(t *testing.T) {
	section("Betweenness Centrality — Riot Graph (unnormalised, ground truth via NetworkX)")
	g := loadGraph(t,
		nodesCSV,
		edgesCSV,
	)

	scores := BetweennessCentrality(g, false)

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
		nodesCSV,
		edgesCSV,
	)

	scores := BetweennessCentrality(g, true)

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
		nodesCSV,
		edgesCSV,
	)

	scores := BetweennessCentrality(g, false)

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
	dist := JointDegreeDistrib(g)

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
		nodesCSV,
		edgesCSV,
	)
	dist := JointDegreeDistrib(g)

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

	results := CountDirectedTriangles(g, nil)

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

	results := CountDirectedTriangles(g, nil)

	for name, info := range results {
		check(t, info[0] == 0, fmt.Sprintf("%s has 0 triangles (no edges)", name), fmt.Sprintf("got %d", info[0]))
	}
}

func TestCountDirectedTriangles_SubsetFilter(t *testing.T) {
	section("CountDirectedTriangles — Subset Node Filter")
	g := buildClusterGraph(t)

	// Request results for only ["A", "B"] — C must not appear in the map.
	results := CountDirectedTriangles(g, []string{"A", "B"})

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
		nodesCSV,
		edgesCSV,
	)

	results := CountDirectedTriangles(g, nil)

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

	coeffs := LocalClusteringCoeff(g, nil)

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

	coeffs := LocalClusteringCoeff(g, nil)

	for name, c := range coeffs {
		check(t, approxEqual(c, 0.0, 1e-9), fmt.Sprintf("%s = 0.0 (no edges)", name), fmt.Sprintf("got %f", c))
	}
}

func TestLocalClusteringCoeff_RiotGraph(t *testing.T) {
	section("LocalClusteringCoeff — Riot Graph")
	g := loadGraph(t,
		nodesCSV,
		edgesCSV,
	)

	coeffs := LocalClusteringCoeff(g, nil)

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

	gt := GlobalTransitivity(g)

	check(t, gt >= 0.0 && gt <= 1.0, "transitivity in [0,1]", fmt.Sprintf("got %f", gt))
	// The graph has a closed triangle so transitivity must be strictly positive.
	check(t, gt > 0.0, "transitivity > 0 (closed triangle exists)", fmt.Sprintf("got %f", gt))
}

func TestGlobalTransitivity_Isolated(t *testing.T) {
	section("GlobalTransitivity — Isolated Nodes")
	g := buildIsolatedGraph(t)

	gt := GlobalTransitivity(g)

	check(t, approxEqual(gt, 0.0, 1e-9), "transitivity = 0.0 (no edges)", fmt.Sprintf("got %f", gt))
}

func TestGlobalTransitivity_RiotGraph(t *testing.T) {
	section("GlobalTransitivity — Riot Graph")
	g := loadGraph(t,
		nodesCSV,
		edgesCSV,
	)

	gt := GlobalTransitivity(g)

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

	avg := AvgClusteringCoeff(g)

	check(t, avg >= 0.0 && avg <= 1.0, "avg coeff in [0,1]", fmt.Sprintf("got %f", avg))
	// At least A and B have non-zero coefficients, so the average must be > 0.
	check(t, avg > 0.0, "avg coeff > 0 (triangle graph)", fmt.Sprintf("got %f", avg))

	// Cross-check: avg must equal the mean of LocalClusteringCoeff values.
	coeffs := LocalClusteringCoeff(g, nil)
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

	avg := AvgClusteringCoeff(g)

	check(t, approxEqual(avg, 0.0, 1e-9), "avg coeff = 0.0 (no edges)", fmt.Sprintf("got %f", avg))
}

func TestAvgClusteringCoeff_RiotGraph(t *testing.T) {
	section("AvgClusteringCoeff — Riot Graph")
	g := loadGraph(t,
		nodesCSV,
		edgesCSV,
	)

	avg := AvgClusteringCoeff(g)

	check(t, avg >= 0.0 && avg <= 1.0, "avg coeff in [0,1]", fmt.Sprintf("got %f", avg))

	// Cross-check against the mean of LocalClusteringCoeff.
	coeffs := LocalClusteringCoeff(g, nil)
	sum := 0.0
	for _, c := range coeffs {
		sum += c
	}
	want := sum / float64(len(coeffs))
	check(t, approxEqual(avg, want, 1e-9), fmt.Sprintf("avg matches manual mean (%.6f)", want), fmt.Sprintf("got %f", avg))
}

// ── Articulation Points ───────────────────────────────────────────────────────

func TestFindArticulationPoints_Chain(t *testing.T) {
	section("FindArticulationPoints — Linear Chain (A→B→C)")
	init_("linear chain: A->B->C")
	g, _ := CreateGraph("chain", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"A", "B", "C"} {
		g.AddNode(n, "functional")
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")

	// Only B bridges A from C; A is an endpoint with no in-neighbours, not an AP.
	got := FindArticulationPoints(g)
	sort.Strings(got)

	gotSet := make(map[string]bool)
	for _, ap := range got {
		gotSet[ap] = true
	}
	check(t, gotSet["B"], "B is an articulation point (sole bridge)", fmt.Sprintf("got %v", got))
	check(t, !gotSet["A"], "A is not an articulation point (source endpoint)", fmt.Sprintf("got %v", got))
	check(t, !gotSet["C"], "C is not an articulation point (sink)", fmt.Sprintf("got %v", got))
}

func TestFindArticulationPoints_Cycle(t *testing.T) {
	section("FindArticulationPoints — Simple Cycle (A→B→C→A)")
	init_("directed cycle: A->B->C->A")
	g, _ := CreateGraph("cycle", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"A", "B", "C"} {
		g.AddNode(n, "functional")
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")
	g.AddEdge("C->A", "dependency", "C", "A")

	// The back edge C→A propagates low all the way to disc[A], so no node's
	// subtree is stranded — zero articulation points.
	got := FindArticulationPoints(g)

	check(t, len(got) == 0, "0 articulation points (fully cyclic)", fmt.Sprintf("got %v", got))
}

func TestFindArticulationPoints_Isolated(t *testing.T) {
	section("FindArticulationPoints — Isolated Nodes")
	init_("three isolated nodes, no edges")
	g, _ := CreateGraph("isolated", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"X", "Y", "Z"} {
		g.AddNode(n, "functional")
	}

	// No edges — removing any node changes nothing.
	got := FindArticulationPoints(g)

	check(t, len(got) == 0, "0 articulation points (no edges)", fmt.Sprintf("got %v", got))
}

func TestFindArticulationPoints_SingleNode(t *testing.T) {
	section("FindArticulationPoints — Single Node")
	init_("graph with a single node")
	g, _ := CreateGraph("single", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	g.AddNode("A", "functional")

	got := FindArticulationPoints(g)

	check(t, len(got) == 0, "0 articulation points (single node)", fmt.Sprintf("got %v", got))
}

func TestFindArticulationPoints_LongChain(t *testing.T) {
	section("FindArticulationPoints — Long Chain (A→B→C→D)")
	init_("chain: A->B->C->D")
	g, _ := CreateGraph("long-chain", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"A", "B", "C", "D"} {
		g.AddNode(n, "functional")
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")
	g.AddEdge("C->D", "dependency", "C", "D")

	// B and C each bridge the chain; A (source) and D (sink) are not APs.
	got := FindArticulationPoints(g)

	gotSet := make(map[string]bool)
	for _, ap := range got {
		gotSet[ap] = true
	}
	check(t, gotSet["B"], "B is an articulation point", fmt.Sprintf("got %v", got))
	check(t, gotSet["C"], "C is an articulation point", fmt.Sprintf("got %v", got))
	check(t, !gotSet["A"], "A is not an articulation point (source endpoint)", fmt.Sprintf("got %v", got))
	check(t, !gotSet["D"], "D is not an articulation point (sink)", fmt.Sprintf("got %v", got))
}

func TestFindArticulationPoints_NoDuplicates(t *testing.T) {
	section("FindArticulationPoints — No Duplicate Entries in Result")
	init_("chain A->B->C: B is an AP for multiple downstream nodes")
	g, _ := CreateGraph("no-dups", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"A", "B", "C"} {
		g.AddNode(n, "functional")
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->C", "dependency", "B", "C")

	// artPts is a map[string]int so each node can only appear once,
	// but the final slice must also contain no repeats.
	got := FindArticulationPoints(g)

	seen := make(map[string]int)
	for _, ap := range got {
		seen[ap]++
	}
	for node, count := range seen {
		check(t, count == 1,
			fmt.Sprintf("%s appears exactly once", node),
			fmt.Sprintf("appeared %d times", count))
	}
}

func TestFindArticulationPoints_RiotGraph(t *testing.T) {
	section("FindArticulationPoints — Riot Graph")
	g := loadGraph(t,
		nodesCSV,
		edgesCSV,
	)

	got := FindArticulationPoints(g)
	sort.Strings(got)

	gotSet := make(map[string]bool)
	for _, ap := range got {
		gotSet[ap] = true
	}

	// RiotSignOn is the structural hub connecting all functional services to
	// the AWS hosting layer; its removal fragments the graph and is the one
	// AP that holds regardless of DFS traversal order.
	check(t, gotSet["RiotSignOn"],
		"RiotSignOn is an articulation point (central hub)",
		fmt.Sprintf("missing from result %v", got))

	// Pure provider leaf nodes are never articulation points — they have no
	// out-neighbours (or only one in-neighbour) so removing them cannot
	// increase the number of connected components.
	for _, name := range []string{"AWS Region 1", "AWS Region 2", "AWS Region 3", "AWS Region 4", "Riot-owned provider"} {
		check(t, !gotSet[name],
			fmt.Sprintf("%q is not an articulation point (leaf provider)", name),
			fmt.Sprintf("unexpectedly present in %v", got))
	}

	// Result must contain no duplicates.
	seen := make(map[string]int)
	for _, ap := range got {
		seen[ap]++
	}
	for node, count := range seen {
		check(t, count == 1,
			fmt.Sprintf("%s appears exactly once in Riot result", node),
			fmt.Sprintf("appeared %d times", count))
	}
}

// ── Algebraic Connectivity ────────────────────────────────────────────────────

// buildPathGraph returns the undirected path A — B — C represented as a
// directed graph with edges in both directions.
// Laplacian eigenvalues: 0, 2−√2 ≈ 0.5858, 2+√2 ≈ 3.4142 → λ₂ = 2−√2.
func buildPathGraph(t *testing.T) *Graph {
    t.Helper()
    init_("path graph A—B—C (one-directional, symmetrised internally)")
    g, _ := CreateGraph("path", false)
    log.SetOutput(io.Discard)
    t.Cleanup(func() { log.SetOutput(os.Stderr) })
    for _, n := range []string{"A", "B", "C"} {
        g.AddNode(n, "functional")
    }
    g.AddEdge("A->B", "dependency", "A", "B")
    g.AddEdge("B->C", "dependency", "B", "C")
    return g
}

func TestAlgebraicConnectivity_SingleNode(t *testing.T) {
	section("AlgebraicConnectivity — Single Node")
	init_("graph with one node, no edges")
	g, _ := CreateGraph("single", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	g.AddNode("A", "functional")

	// 1×1 Laplacian has no second eigenvalue — expect 0.
	ac := AlgebraicConnectivity(g)
	check(t, approxEqual(ac, 0.0, 1e-9), "single node → λ₂ = 0.0", fmt.Sprintf("got %f", ac))
}

func TestAlgebraicConnectivity_Disconnected(t *testing.T) {
	section("AlgebraicConnectivity — Disconnected Graph")
	init_("two isolated nodes: A, B (no edges)")
	g, _ := CreateGraph("disconnected", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	g.AddNode("A", "functional")
	g.AddNode("B", "functional")

	// Disconnected graph has ≥2 zero eigenvalues → λ₂ = 0.
	ac := AlgebraicConnectivity(g)
	check(t, approxEqual(ac, 0.0, 1e-9), "disconnected → λ₂ = 0.0", fmt.Sprintf("got %f", ac))
}

func TestAlgebraicConnectivity_PathGraph(t *testing.T) {
    section("AlgebraicConnectivity — Path Graph A—B—C")
    g := buildPathGraph(t)

    // λ₂ of P₃ is exactly 1.0
    ac := AlgebraicConnectivity(g)

    check(t, ac > 0.0, "path graph is connected → λ₂ > 0", fmt.Sprintf("got %f", ac))
    check(t, approxEqual(ac, 1.0, 1e-6),
        "λ₂ = 1.0 (P₃ exact)",
        fmt.Sprintf("got %.10f", ac))
}

func TestAlgebraicConnectivity_CompleteGraph(t *testing.T) {
	section("AlgebraicConnectivity — Complete Graph K₃")
	init_("complete graph K3: all six directed edges")
	g, _ := CreateGraph("K3", false)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, n := range []string{"A", "B", "C"} {
		g.AddNode(n, "functional")
	}
	g.AddEdge("A->B", "dependency", "A", "B")
	g.AddEdge("B->A", "dependency", "B", "A")
	g.AddEdge("B->C", "dependency", "B", "C")
	g.AddEdge("C->B", "dependency", "C", "B")
	g.AddEdge("A->C", "dependency", "A", "C")
	g.AddEdge("C->A", "dependency", "C", "A")

	// Every node has degree 2; both non-zero eigenvalues equal 3 → λ₂ = 3.
	ac := AlgebraicConnectivity(g)
	check(t, approxEqual(ac, 3.0, 1e-6), "K₃ → λ₂ = 3.0", fmt.Sprintf("got %f", ac))
}

func TestAlgebraicConnectivity_RiotGraph(t *testing.T) {
	section("AlgebraicConnectivity — Riot Graph")
	g := loadGraph(t, nodesCSV, edgesCSV)

	ac := AlgebraicConnectivity(g)

	check(t, ac >= 0.0, "λ₂ ≥ 0 (Laplacian is positive semi-definite)", fmt.Sprintf("got %f", ac))
	// The Riot graph is connected through RiotSignOn when symmetrised → λ₂ > 0.
	check(t, ac > 0.0, "Riot graph is connected → λ₂ > 0", fmt.Sprintf("got %f", ac))
	// λ₂ is bounded above by n for any simple graph.
	check(t, ac < float64(len(g.Nodes)),
		"λ₂ < n (upper bound for simple graph)",
		fmt.Sprintf("got %f, n=%d", ac, len(g.Nodes)))
}

func TestAlgebraicConnectivity_RiotGraph_RemoveHub(t *testing.T) {
	section("AlgebraicConnectivity — Riot Graph λ₂ drops after hub removal")
	g := loadGraph(t, nodesCSV, edgesCSV)

	before := AlgebraicConnectivity(g)

	// RiotSignOn is the central articulation point; removing it must reduce
	// connectivity (lower λ₂) or fully disconnect the graph (λ₂ = 0).
	g.RemoveNode("RiotSignOn")
	after := AlgebraicConnectivity(g)

	check(t, after <= before,
		fmt.Sprintf("λ₂ drops after hub removal (%.6f → %.6f)", before, after),
		fmt.Sprintf("before=%.6f  after=%.6f", before, after))
}