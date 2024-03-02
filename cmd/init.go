package cmd

import (
	"fmt"
	// "os"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

func init() {
	// addCommand.Flags().StringP("file", "f", "", "file to add")
	rootCmd.AddCommand(initcmd)
}

var initcmd = &cobra.Command{
	Use: "init",
	Short: "Initialize dotfiles management",
	Long: `Initialize dotfiles management.
Makes a git repository and sets remote origin to the specified URL.
Default URL is $HOME/dotfiles but can be changed with the --url flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		r, _ := git.PlainOpen(rootGitRepoPath)

		fmt.Println("r", r)

		b, _ := r.Branches()
		b.ForEach(func(b *plumbing.Reference) error {
			fmt.Println("Branch", b.Name())
			return nil
		})
	},
}