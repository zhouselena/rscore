package main

import (
	"cmp"
	"fmt"
	"log"
	"slices"

	"github.com/spf13/cobra"
	"github.com/zhouselena/rscore"
)

func newScoreCommand() *cobra.Command {

    cmd := &cobra.Command{
        Use:   "score",
        Short: "Calculate resilience score for application infrastructure, and node criticality.",
    }

    // optional flags
    var nodespath, edgespath string
    cmd.Flags().StringVarP(&nodespath, "nodes-path", "n", "", `specify path to nodes csv file, e.g. "public/templates/nodes.csv"`)
    cmd.Flags().StringVarP(&edgespath, "edges-path", "e", "", `specify path to edges csv file, e.g. "public/templates/edges.csv"`)

    cmd.Run = func(cmd *cobra.Command, args []string) {
        err := runScoreCommand(cmd, args, nodespath, edgespath)
        if err != nil {
            log.Fatalf("failed to calculate score: %v", err)
        }
    }

    return cmd
}

func runScoreCommand(cmd *cobra.Command, args []string, nodespath string, edgespath string) error {

	// load application infrastructure
    _, err := rscore.Load(nodespath, edgespath)
    if (err != nil) {
        return fmt.Errorf("was unable to load infra: %q", err)
    }

	// run resilience score
	if rscore.LoadAllAlgorithms() != nil {
		return fmt.Errorf("was unable to calculate scores: %q", err)
	}

	// calculate scores
	rScore, recommendation := rscore.CalculateGraphResiliency()
	nodeScores := rscore.CalculateNodeCriticalness()

	// display results
	fmt.Printf("=== Infrastructure Resilience Score: %.4f ===\n", rScore)
    fmt.Printf("Recommendation: %s\n\n", recommendation)

    fmt.Println("=== Node Criticality Scores ===")
    // sort nodes by criticality descending for readability
    type nodeScore struct {
        name  string
        score float64
    }
    ranked := make([]nodeScore, 0, len(nodeScores))
    for node, score := range nodeScores {
        ranked = append(ranked, nodeScore{node, score})
    }
    slices.SortFunc(ranked, func(a, b nodeScore) int {
        return cmp.Compare(b.score, a.score) // descending
    })
    for _, ns := range ranked {
        fmt.Printf("  %-30s %.4f\n", ns.name, ns.score)
    }

    return nil
}