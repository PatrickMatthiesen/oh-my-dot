package git

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/viper"
)

// RemoteSyncState describes local/remote relationship for the current branch.
type RemoteSyncState string

const (
	// RemoteSyncUpToDate indicates local and remote point to the same commit.
	RemoteSyncUpToDate RemoteSyncState = "up-to-date"
	// RemoteSyncRemoteAhead indicates remote contains commits missing locally.
	RemoteSyncRemoteAhead RemoteSyncState = "remote-ahead"
	// RemoteSyncRemoteSignificantlyAhead indicates local is at least 100 commits behind remote.
	RemoteSyncRemoteSignificantlyAhead RemoteSyncState = "remote-significantly-ahead"
	// RemoteSyncLocalAhead indicates local contains commits missing on remote.
	RemoteSyncLocalAhead RemoteSyncState = "local-ahead"
	// RemoteSyncDiverged indicates both sides contain unique commits.
	RemoteSyncDiverged RemoteSyncState = "diverged"
)

const maxAncestorSearchDepth = 100

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

// LinkAndAddFile takes a file path as an argument, makes a hard-link to the git repo and adds the file to the git repo.
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
	} else if !fileops.IsDir(filepath.Dir(destination)) { // TODO: consider if we should have a force or create-dir flag to force the copy
		return fmt.Errorf("file cannot be copied to %s. Is not a valid path or directory does not exist", destination)
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

// PullRepo pulls changes from the origin remote for the current branch.
// Returns true if updates were applied, false if already up to date.
func PullRepo() (bool, error) {
	repoPath := viper.GetString("repo-path")
	if repoPath == "" {
		return false, fmt.Errorf("repository path is not set")
	}

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open repository: %w", err)
	}

	headRef, err := r.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get current branch: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return false, fmt.Errorf("cannot pull from detached HEAD")
	}

	worktree, err := r.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: headRef.Name(),
		SingleBranch:  true,
	})
	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return false, nil
		}
		return false, fmt.Errorf("failed to pull repository: %w", err)
	}

	return true, nil
}

// HasRemoteUpdates checks if the current branch needs pull from origin.
// Returns true when remote is ahead or diverged.
func HasRemoteUpdates() (bool, error) {
	state, err := GetRemoteSyncState()
	if err != nil {
		return false, err
	}

	return state == RemoteSyncRemoteAhead || state == RemoteSyncRemoteSignificantlyAhead || state == RemoteSyncDiverged, nil
}

// GetRemoteSyncState returns local/remote relationship for the current branch.
// It uses a lightweight remote reference list and local commit graph traversal.
func GetRemoteSyncState() (RemoteSyncState, error) {
	repoPath := viper.GetString("repo-path")
	if repoPath == "" {
		return "", fmt.Errorf("repository path is not set")
	}

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	headRef, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return "", fmt.Errorf("cannot check updates from detached HEAD")
	}

	remote, err := r.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("no remote 'origin' configured: %w", err)
	}

	remoteRefs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("unable to access remote repository: %w", err)
	}

	remoteBranchRefName := plumbing.NewBranchReferenceName(headRef.Name().Short())
	var remoteBranchHash plumbing.Hash
	found := false
	for _, ref := range remoteRefs {
		if ref.Name() == remoteBranchRefName {
			remoteBranchHash = ref.Hash()
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("remote branch %s not found", remoteBranchRefName.String())
	}

	localHash := headRef.Hash()
	if remoteBranchHash == localHash {
		return RemoteSyncUpToDate, nil
	}

	remoteCommit, err := r.CommitObject(remoteBranchHash)
	if err != nil {
		if errors.Is(err, plumbing.ErrObjectNotFound) {
			return RemoteSyncRemoteAhead, nil
		}
		return "", fmt.Errorf("failed to inspect remote commit %s: %w", remoteBranchHash.String(), err)
	}

	localCommit, err := r.CommitObject(localHash)
	if err != nil {
		return "", fmt.Errorf("failed to inspect local commit %s: %w", localHash.String(), err)
	}

	remoteHasLocal, remoteDepthExceeded, err := commitContainsAncestor(remoteCommit, localHash, maxAncestorSearchDepth)
	if err != nil {
		return "", fmt.Errorf("failed to compare local and remote commits: %w", err)
	}
	if remoteHasLocal {
		return RemoteSyncRemoteAhead, nil
	}
	if remoteDepthExceeded {
		return RemoteSyncRemoteSignificantlyAhead, nil
	}

	localHasRemote, localDepthExceeded, err := commitContainsAncestor(localCommit, remoteBranchHash, maxAncestorSearchDepth)
	if err != nil {
		return "", fmt.Errorf("failed to compare local and remote commits: %w", err)
	}
	if localHasRemote {
		return RemoteSyncLocalAhead, nil
	}
	if localDepthExceeded {
		return RemoteSyncLocalAhead, nil
	}

	return RemoteSyncDiverged, nil
}

// commitContainsAncestor reports whether targetHash is an ancestor of start by
// performing a depth-first search over the commit graph starting from start.
// Returns depthExceeded when maxDepth commits were inspected without finding targetHash.
func commitContainsAncestor(start *object.Commit, targetHash plumbing.Hash, maxDepth int) (bool, bool, error) {
	visited := map[plumbing.Hash]struct{}{}
	stack := []*object.Commit{start}
	inspected := 0

	for len(stack) > 0 {
		if maxDepth > 0 && inspected >= maxDepth {
			return false, true, nil
		}

		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		inspected++

		if current.Hash == targetHash {
			return true, false, nil
		}

		if _, seen := visited[current.Hash]; seen {
			continue
		}
		visited[current.Hash] = struct{}{}

		parents := current.Parents()
		err := func() error {
			defer parents.Close()

			return parents.ForEach(func(parent *object.Commit) error {
				stack = append(stack, parent)
				return nil
			})
		}()
		if err != nil {
			return false, false, fmt.Errorf("failed to iterate commit parents: %w", err)
		}
	}

	return false, false, nil
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

// permissionTestFileName is the name of the temporary file used to test write permissions.
// Uses a dot prefix to make it hidden on Unix systems, avoiding clutter in the repository directory.
const permissionTestFileName = ".oh-my-dot-permission-test"

// CheckRepoWritePermission checks if the user has write permissions on the dotfiles directory
func CheckRepoWritePermission() error {
	repoPath := viper.GetString("repo-path")
	if repoPath == "" {
		return fmt.Errorf("repository path is not set")
	}

	// Check if the directory exists
	info, err := os.Stat(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("repository directory does not exist: %s", repoPath)
		}
		return fmt.Errorf("failed to stat repository directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", repoPath)
	}

	// Try to create a temporary file to verify write permissions
	testFile := filepath.Join(repoPath, permissionTestFileName)
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("no write permission on repository directory: %s", repoPath)
	}
	f.Close()

	// Clean up the test file
	if err := os.Remove(testFile); err != nil {
		// Log the error but don't fail - permission check already succeeded
		log.Printf("Warning: failed to remove permission test file %s: %v", testFile, err)
	}

	return nil
}

// StageAndCommitShellFeatureChanges stages and commits shell framework changes under omd-shells.
// Device-local override manifests (enabled.local.json) are always excluded.
// Returns true when a commit was created, false when there were no committable changes.
func StageAndCommitShellFeatureChanges(message string) (bool, error) {
	worktree, err := GetWorktree(viper.GetString("repo-path"))
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to read worktree status: %w", err)
	}

	hasStagedChanges := false
	for path, fileStatus := range status {
		if fileStatus.Staging == git.Unmodified && fileStatus.Worktree == git.Unmodified {
			continue
		}

		if !isCommittableShellChangePath(path) {
			continue
		}

		if fileStatus.Worktree == git.Deleted || fileStatus.Staging == git.Deleted {
			if _, err := worktree.Remove(path); err != nil {
				return false, fmt.Errorf("failed to stage deleted file %s: %w", path, err)
			}
		} else {
			if _, err := worktree.Add(path); err != nil {
				return false, fmt.Errorf("failed to stage file %s: %w", path, err)
			}
		}

		hasStagedChanges = true
	}

	if !hasStagedChanges {
		return false, nil
	}

	if _, err := worktree.Commit(message, &git.CommitOptions{}); err != nil {
		return false, fmt.Errorf("failed to commit shell feature changes: %w", err)
	}

	return true, nil
}

func isCommittableShellChangePath(path string) bool {
	normalizedPath := filepath.ToSlash(filepath.Clean(path))
	if !strings.HasPrefix(normalizedPath, "omd-shells/") {
		return false
	}

	return !strings.HasSuffix(normalizedPath, "/"+shell.LocalManifestFileName())
}

// CheckRemotePushPermission checks if the user has valid git credentials for pushing to the remote repository.
// It uses the same authentication mechanism as git push (SSH keys, credential helpers, etc.) to verify access.
func CheckRemotePushPermission() error {
	repoPath := viper.GetString("repo-path")
	if repoPath == "" {
		return fmt.Errorf("repository path is not set")
	}

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	remote, err := r.Remote("origin")
	if err != nil {
		return fmt.Errorf("no remote 'origin' configured: %w", err)
	}

	// List references from the remote to check connectivity and credentials.
	// This is a lightweight operation that verifies we can authenticate without actually pushing.
	// Uses default git authentication (SSH keys, credential helpers, etc.).
	_, err = remote.List(&git.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to access remote repository (check credentials and network): %w", err)
	}

	return nil
}

// IsSSHAgentError checks if the error is related to missing SSH_AUTH_SOCK
func IsSSHAgentError(err error) bool {
	if err == nil {
		return false
	}
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "SSH_AUTH_SOCK") || strings.Contains(errorMsg, "SSH agent")
}
