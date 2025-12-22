package cmd_test

import (
	"os"
	"path/filepath"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"testing"
)

func Test_Plain_Init_cmd(t *testing.T) {
	fakeGitRepoPath := t.TempDir()
	_, err := git.PlainInit(fakeGitRepoPath, true)
	TBErrorIfNotNil(t, err)

	invokeCommand(t, []string{"init", fakeGitRepoPath})

	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}
}

func Test_Existing_Init_cmd(t *testing.T) {
	fakeGitRepoPath := t.TempDir()
	_, err := git.PlainInit(fakeGitRepoPath, true)
	TBErrorIfNotNil(t, err)

	invokeCommand(t, []string{"init", fakeGitRepoPath})

	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}
}

// Test_Init_Clone_Remote_With_Main_Branch tests that when initializing with a remote repository
// that has 'main' as the default branch, the cloned repository also uses 'main'
func Test_Init_Clone_Remote_With_Main_Branch(t *testing.T) {
	// Create a fake remote repository with 'main' as the default branch
	remoteRepoPath := t.TempDir()
	remoteRepo, err := git.PlainInit(remoteRepoPath, false)
	TBErrorIfNotNil(t, err)

	// Create a file and commit to establish the main branch
	testFilePath := filepath.Join(remoteRepoPath, "test.txt")
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	TBErrorIfNotNil(t, err)

	wt, err := remoteRepo.Worktree()
	TBErrorIfNotNil(t, err)

	_, err = wt.Add("test.txt")
	TBErrorIfNotNil(t, err)

	// Commit with author information
	commit, err := wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	TBErrorIfNotNil(t, err)

	// Create main branch reference
	mainRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/main"), commit)
	err = remoteRepo.Storer.SetReference(mainRef)
	TBErrorIfNotNil(t, err)

	// Update HEAD to point to main
	err = remoteRepo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main")))
	TBErrorIfNotNil(t, err)

	// Now initialize with this remote repository
	invokeCommand(t, []string{"init", remoteRepoPath})

	// Verify the cloned repository exists
	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}

	// Verify the default branch is 'main'
	clonedRepo, err := git.PlainOpen(viper.GetString("repo-path"))
	TBErrorIfNotNil(t, err)

	clonedHead, err := clonedRepo.Head()
	TBErrorIfNotNil(t, err)

	if clonedHead.Name().Short() != "main" {
		t.Errorf("Expected default branch to be 'main', got '%s'", clonedHead.Name().Short())
	}
}

func invokeCommand(t *testing.T, args []string) {
	viper.Reset()

	// Set default values for viper
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	viper.SetDefault("dot-home", configFile)

	repoFolder := t.TempDir()
	viper.SetDefault("repo-path", filepath.Join(repoFolder, "dotfiles"))

	viper.SetConfigFile(configFile)

	viper.AutomaticEnv()

	cmd.Execute(func(c *cobra.Command) {
		c.SetArgs(args)
	})
}

func TBErrorIfNotNil(t testing.TB, err error) {
	if err != nil {
		t.Error(err)
	}
}

func FErrorIfNotNil(f *testing.F, err error) {
	if err != nil {
		f.Error(err)
	}
}