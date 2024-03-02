package cmd

import (
	"fmt"
	"github.com/PatrickMatthiesen/oh-my-dot/util"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

func init() {
	addCommand.Flags().StringP("file", "f", "", "file to add")
	rootCmd.AddCommand(addCommand)
}

var addCommand = &cobra.Command{
	Use:              "add [file]",
	Short:            "Add config files to the repository",
	Long:             `Add config files to the repository.`,
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Add called", args[0])
		
		r, err := git.PlainOpen("")
		//TODO: should add filename to error message
		util.CheckIfError(err)

		w, err := r.Worktree()
		util.CheckIfError(err)

		w.Add(args[0])
		w.Commit("Add file", &git.CommitOptions{})
	},
}

