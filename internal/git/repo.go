package git

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/spf13/viper"
)

// Check if folder has git repo
func IsGitRepo(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
}

// GetWorktree returns the worktree of the git repository located at the specified path.
func GetWorktree(rootGitRepoPath string) (*git.Worktree, error) {
	r, err := git.PlainOpen(rootGitRepoPath)
	if err != nil {
		return nil, err
	}

	return r.Worktree()
}

// InitGitRepo initializes a new git repository at the specified path,
// with an optional remote URL and bare repository flag.
// If a remote URL is provided and the remote repository exists, it clones the repository.
// Otherwise, it initializes a new empty repository and sets up the remote.
func InitGitRepo(rootGitRepoPath string, remoteUrl string, opts ...bool) (*git.Repository, error) {
	bare := false
	if len(opts) > 0 {
		bare = opts[0]
	}

	// If a remote URL is provided, try to clone the repository first
	if remoteUrl != "" && !bare {
		// Attempt to clone the remote repository
		r, err := git.PlainClone(rootGitRepoPath, false, &git.CloneOptions{
			URL: remoteUrl,
		})
		if err == nil {
			// Clone succeeded, return the cloned repository
			return r, nil
		}
		// If clone fails, fall back to initializing a new repository
		// This handles cases where the remote doesn't exist yet or is inaccessible
	}

	// Fall back to creating a new empty repository
	r, err := git.PlainInit(rootGitRepoPath, bare)
	if err != nil {
		return nil, err
	}

	// Only set up the remote if a URL was provided
	if remoteUrl != "" {
		_, err = r.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{remoteUrl},
		})
		if err != nil {
			return nil, err
		}
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

// LinkAndAddFile takes a file path as an argument, makes a har-link to the git repo and adds the file to the git repo.
func LinkAndAddFile(file string) error {
	fileName := filepath.Base(file)
	fileRepoPath := fmt.Sprint("files/", fileName)

	newFile := filepath.Join(viper.GetString("repo-path"), fileRepoPath)
	fmt.Println("Linking", fileops.SColorPrint(file, fileops.Blue), "to", fileops.SColorPrint(newFile, fileops.Cyan))

	fileops.EnsureDir(filepath.Dir(newFile))
	err := os.Link(file, newFile)
	if err != nil {
		return err
	}

	return StageChange(fileRepoPath)
}

// CopyAndAddFile takes a file path as an argument, copies the file to the git repo and adds the file to the git repo.
func CopyAndAddFile(file, destination string) error {
	fileName := filepath.Base(file)
	fileRepoPath := fmt.Sprint("files/", fileName)

	if fileops.IsDir(destination) {
		destination = filepath.Join(destination, fileName)
	} else if !fileops.IsDir(filepath.Dir(destination)) { // TODO: consider if we sould have a force or create-dir flag to force the copy
		return fmt.Errorf("file cannot be coppied to %s. Is not a valid path or dirrectory does not exist", destination)
	}

	err := fileops.CopyFile(file, destination)
	if err != nil {
		return err
	}

	newFile := filepath.Join(viper.GetString("repo-path"), fileRepoPath)
	log.Println("Copying", file, "to", newFile)

	fileops.EnsureDir(filepath.Dir(newFile))
	err = fileops.CopyFile(file, newFile)
	if err != nil {
		return err
	}

	return StageChange(fileRepoPath)
}

func RemoveFile(file string) error {
	repoPath := viper.GetString("repo-path")
	worktree, err := GetWorktree(repoPath)
	if err != nil {
		return err
	}

	filesPath := filepath.Join(repoPath, "files")
	
	// Normalize the path: if it's absolute and within the repo, keep it as-is
	// Otherwise, treat it as relative to filesPath
	var fullPath string
	if filepath.IsAbs(file) && strings.HasPrefix(file, repoPath) {
		// Already an absolute path within the repo
		fullPath = file
	} else {
		// Treat as relative path - join with filesPath
		// Strip leading separators to handle accidental user input like "/myfile.txt"
		file = strings.TrimPrefix(file, string(filepath.Separator))
		fullPath = filepath.Join(filesPath, file)
	}

	is, err := fileops.IsFileErr(fullPath)
	if err != nil {
		return fmt.Errorf("cannot inspect %s: %w", fullPath, err)
	}
	if !is {
		return fmt.Errorf("%s exists but is not a file", fullPath)
	}

	relativeFilePath, err := filepath.Rel(repoPath, fullPath)
	if err != nil {
		return fmt.Errorf("file %s is not in the repository\nInternal error: %v", fullPath, err)
	}

	_, err = worktree.Remove(relativeFilePath)
	return err
}

// StageChange adds the specified file to the git repository.
func StageChange(file string) error {
	worktree, err := GetWorktree(viper.GetString("repo-path"))
	if err != nil {
		return err
	}

	_, err = worktree.Add(file)
	if err != nil {
		return err
	}

	return err
}

// Commits the changes in the git repository located at the specified path.
func Commit(message string) error {
	worktree, err := GetWorktree(viper.GetString("repo-path"))
	if err != nil {
		return err
	}

	_, err = worktree.Commit(message, &git.CommitOptions{})
	if err != nil {
		return err
	}

	return nil
}

// PushRepo pushes the changes in the git repository located at the specified path to the remote repository.
func PushRepo() error {
	r, err := git.PlainOpen(viper.GetString("repo-path"))
	fileops.CheckIfError(err)

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
	worktree, err := GetWorktree(viper.GetString("repo-path"))
	if err != nil {
		return nil, err
	}

	infos, err := worktree.Filesystem.ReadDir("files")
	if err != nil {
		return nil, err
	}

	files := make([]string, len(infos))
	for i, info := range infos {
		files[i] = info.Name()
	}

	return files, nil
}

func UrlIsGitRepo(url string) bool { // unused
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

func ReadyForClone(folderPath string) (bool, error) { // unused
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
