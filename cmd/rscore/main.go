package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
    cmd := &cobra.Command{
        Use:   "rscore",
        Short: "A tool to evaluate the resilience of the infrastructure of your app.",
        Long:  "r(esilience)score is a tool that takes in the infrastructure of your app, and runs calculations to evaluate the resilience based on dependence on features, service providers, and layout.",
    }

    cmd.AddCommand(newLoadCommand())

    if err := cmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
