package util

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/spf13/viper"
)

func MoveAndAddFile(file string) {
	//TODO: take a file path as argument and move it to the git repo
	//add the file to the git repo
	os.Rename(file, viper.GetString("repo-path"))
	
	AddFileToRepo(file)
}

// TODO: get the repo object
func getWorktree(rootGitRepoPath string) *git.Worktree {
	r, _ := git.PlainOpen(rootGitRepoPath)

	fmt.Println("r", r)

	worktree, _ := r.Worktree()

	fmt.Println("Worktree", worktree)
	return worktree
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

func AddFileToRepo(file string) {
	worktree := getWorktree(viper.GetString("repo-path"))

	worktree.Add(file)
	//TODO: remove dir from file path
	worktree.Commit(fmt.Sprint("Add", file), &git.CommitOptions{})
}