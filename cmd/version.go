package cmd

import (
  "fmt"

  "github.com/spf13/cobra"
)

func init() {
  rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version number of oh-my-dot",
    Long:  `Get informed about the current version of oh-my-dot.`,
    Run: func(cmd *cobra.Command, args []string) {
      fmt.Println("oh-my-dot v0.1")
    },
}