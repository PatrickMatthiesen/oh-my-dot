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

// Check if folder has git repo
func IsGitRepo(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
}

// LinkAndAddFile takes a file path as an argument, makes a har-link to the git repo,
// adds the file to the git repo, and commits the changes.
func LinkAndAddFile(file string) error {
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

// GetWorktree returns the worktree of the git repository located at the specified path.
func GetWorktree(rootGitRepoPath string) (*git.Worktree, error) {
	r, err := git.PlainOpen(rootGitRepoPath)
	if err != nil {
		return nil, err
	}

	worktree, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	return worktree, nil
}

// InitGitRepo initializes a new git repository at the specified path,
// with an optional remote URL and bare repository flag.
func InitGitRepo(rootGitRepoPath string, remoteUrl string, opts ...bool) (*git.Repository, error) {
	bare := false
	if len(opts) > 0 {
		bare = opts[0]
	}

	r, err := git.PlainInit(rootGitRepoPath, bare)
	if err != nil {
		return nil, err
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteUrl},
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

// InitFromExistingRepo initializes a git repository from an existing repository located at the specified path.
func InitFromExistingRepo(rootGitRepoPath string) error {
	r, err := git.PlainOpen(rootGitRepoPath)
	if err != nil {
		return err
	}

	// set remote in config
	remote, err := r.Remote("origin")
	if err != nil {
		return err
	}

	remoteConfig := remote.Config()
	viper.Set("remote-url", remoteConfig.URLs[0])

	return nil
}

// AddFileToRepo adds the specified file to the git repository.
func AddFileToRepo(file string) error {
	worktree, err := GetWorktree(viper.GetString("repo-path"))
	if err != nil{
		return err
	}

	_, err = worktree.Add(file)
	if err != nil{
		return err
	}

	_, err = worktree.Commit(fmt.Sprint("Add ", file), &git.CommitOptions{})

	return err
}

func UrlIsGitRepo(url string) bool {
	stat, _ := os.Stat(url)
	if stat != nil && stat.IsDir() {
		_, err := git.PlainOpen(url)
		if err == nil {
			return true
		}
	}

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

// PushRepo pushes the changes in the git repository located at the specified path to the remote repository.
func PushRepo() error {
	r, err := git.PlainOpen(viper.GetString("repo-path"))
	CheckIfError(err)

	remote, err := r.Remote("origin")
	if err != nil {
		return err
	}

	err = remote.Push(&git.PushOptions{})
	if err != nil {
		return err
	}

	return nil
}

func ListFiles() ([]string, error) {
	r, err := git.PlainOpen(viper.GetString("repo-path"))
	if err != nil {
		return nil, err
	}

	worktree, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, err
	}

	var files []string
	for file := range status {
		files = append(files, file)
	}

	return files, nil
}