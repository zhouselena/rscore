package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var verbose bool

var command1Cmd = &cobra.Command{
    Use:   "command1",
    Short: "A top-level command",
    Run: func(cmd *cobra.Command, args []string) {
        if verbose {
            fmt.Println("Running command1 in verbose mode")
        } else {
            fmt.Println("Running command1")
        }
    },
}

func init() {
    // Add a flag to command1
    command1Cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

    // Add this command to the root
    rootCmd.AddCommand(command1Cmd)
}