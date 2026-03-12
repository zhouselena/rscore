package main

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/zhouselena/rscore"
)

func newLoadCommand() *cobra.Command {

    cmd := &cobra.Command{
        Use:   "load",
        Short: "Load application infrastructure from files.",
    }

    // optional flags
    var nodespath, edgespath string
    cmd.Flags().StringVarP(&nodespath, "nodes-path", "n", "", `specify path to nodes csv file, e.g. "public/templates/nodes.csv"`)
    cmd.Flags().StringVarP(&edgespath, "edges-path", "e", "", `specify path to edges csv file, e.g. "public/templates/edges.csv"`)

    cmd.Run = func(cmd *cobra.Command, args []string) {
        err := runLoadCommand(cmd, args, nodespath, edgespath)
        if err != nil {
            log.Fatalf("failed to load infrastructure: %v", err)
        }
    }

    return cmd
}

func runLoadCommand(cmd *cobra.Command, args []string, nodespath string, edgespath string) error {
    _, err := rscore.Load(nodespath, edgespath)
    if (err != nil) {
        return err
    }
    return nil
}