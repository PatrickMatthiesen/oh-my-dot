package util

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/spf13/viper"
)

func MoveAndAddFile(file string) error {
	//TODO: take a file path as argument and move it to the git repo
	//add the file to the git repo
	fileName := filepath.Base(file)
	fileRepoPath := fmt.Sprint("files/", fileName)

	newFile := filepath.Join(viper.GetString("repo-path"), fileRepoPath)
	log.Println("Linking", file, "to", newFile)

	EnsureDir(filepath.Dir(newFile))
	err := os.Link(file, newFile)
	if err != nil {
		return err
	}

	err = AddFileToRepo(fileRepoPath)
	if err != nil {
		return err
	}

	return nil
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

func symlinkFiles(file string, dest string) error {
	return os.Symlink(file, dest)
}

func IsGitRepo(url string) bool {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	if !strings.HasSuffix(url, ".git") {
		url += ".git"
	}

	resp, err := http.Head(url)
	if err != nil {
		fmt.Println("Error:", err)
		return false
	}
	defer resp.Body.Close()

	// Check if the response status is OK (200)
	return resp.StatusCode == http.StatusOK
}

func isFolderEmpty(name string) (bool, error) {
    f, err := os.Open(name)
    if err != nil {
        return false, err
    }
    defer f.Close()

    _, err = f.Readdirnames(1) // Or f.Readdir(1)
    if err == io.EOF {
        return true, nil
    }
    return false, err // Either not empty or error, suits both cases
}

func ReadyForClone(folderPath string) (bool, error) {
	// Check if the folder exists
	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("folder %s does not exist", folderPath)
		}
		return false, err
	}

	// Verify that it's a directory
	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory", folderPath)
	}

	// Check if the folder is empty
	isEmpty, err := isFolderEmpty(folderPath)
	if err != nil {
		return false, err
	}

	if !isEmpty {
		return false, fmt.Errorf("folder %s is not empty", folderPath)
	}

	return true, nil
}