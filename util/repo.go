package util

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

func MoveAndAddFile() {
	//TODO: take a file path as argument and move it to the git repo
	//add the file to the git repo
}

// TODO: get the repo object
func getRepo(rootGitRepoPath string) {
	r, _ := git.PlainOpen(rootGitRepoPath)

	fmt.Println("r", r)

	Worktree, _ := r.Worktree()

	fmt.Println("Worktree", Worktree)
}
