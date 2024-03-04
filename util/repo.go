package util

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
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

func InitGitRepo(rootGitRepoPath string, remoteUrl string) {
	r, err := git.PlainInit(rootGitRepoPath, false)
	if err != nil {
		fmt.Println("Error initializing git repo")
	}
	r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteUrl},
	})
}