package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/spf13/viper"
)

func MoveAndAddFile(file string) error {
	//TODO: take a file path as argument and move it to the git repo
	//add the file to the git repo
	fileName := filepath.Base(file)

	newFile := filepath.Join(viper.GetString("repo-path"), fileName)
	log.Println("Moving", file, "to", newFile)

	err := os.Rename(file, newFile)
	if err != nil {
		return err
	}

	return AddFileToRepo(file)
}

// TODO: get the repo object
func getWorktree(rootGitRepoPath string) *git.Worktree {
	r, err := git.PlainOpen(rootGitRepoPath)
	if err != nil {
		fmt.Println("Error opening git repo")
		return nil
	}

	worktree, err := r.Worktree()
	if err != nil {
		fmt.Println("Error getting worktree")
	}

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

func AddFileToRepo(file string) error {
	worktree := getWorktree(viper.GetString("repo-path"))

	worktree.Add(file)
	//TODO: remove dir from file path
	_, err := worktree.Commit(fmt.Sprint("Add ", file), &git.CommitOptions{})

	return err
}
