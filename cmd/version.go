package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	versionCmd.Flags().BoolP("long", "l", false, "Print the commit hash")

	rootCmd.AddCommand(versionCmd)
}

var (
	// Version is set at build time and defaults to the constant from version.go
	Version = "0.0.20"
	CommitHash = "n/a"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version number of oh-my-dot",
	Long:  `Get informed about the current version of oh-my-dot.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(Version)

		long, _ := cmd.Flags().GetBool("long")
		if long {
			fmt.Print("+" + CommitHash)
		}
		fmt.Println()
	},
}
