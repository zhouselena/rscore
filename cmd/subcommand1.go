package cmd

import (
    "fmt"

    "github.com/spf13/cobra"
)

var count int

var subcommand1Cmd = &cobra.Command{
    Use:   "subcommand1",
    Short: "A subcommand under command1",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("Running subcommand1 with count=%d\n", count)
    },
}

func init() {
    // Add a flag specific to this subcommand
    subcommand1Cmd.Flags().IntVarP(&count, "count", "c", 1, "Number of iterations")

    // Attach to command1 as a subcommand
    command1Cmd.AddCommand(subcommand1Cmd)
}