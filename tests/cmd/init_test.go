package cmd_test

import (
	"os"
	"path/filepath"

	"log"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/tests/testutil"
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"testing"
)

func mockHomeDir(t testing.TB) {
	t.Helper()
	// Mock the home directory to a temporary directory
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	os.Setenv("home", tempHome) // For Plan9
	os.Setenv("USERPROFILE", tempHome) // For Windows
}

func Test_Plain_Init_cmd(t *testing.T) {
	fakeGitRepoPath := t.TempDir()
	_, err := git.PlainInit(fakeGitRepoPath, true)
	testutil.TBErrorIfNotNil(t, err)

	invokeCommand(t, []string{"init", fakeGitRepoPath})

	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}
}

func Test_Existing_Init_cmd(t *testing.T) {
	fakeGitRepoPath := t.TempDir()
	_, err := git.PlainInit(fakeGitRepoPath, true)
	testutil.TBErrorIfNotNil(t, err)

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
	remoteRepoPath := testutil.CreateRemoteRepo(t, "main")

	// Now initialize with this remote repository
	invokeCommand(t, []string{"init", remoteRepoPath})

	// Verify the cloned repository exists
	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}

	// Verify the default branch is 'main'
	clonedRepo, err := git.PlainOpen(viper.GetString("repo-path"))
	testutil.TBErrorIfNotNil(t, err)

	clonedHead, err := clonedRepo.Head()
	testutil.TBErrorIfNotNil(t, err)

	if clonedHead.Name().Short() != "main" {
		t.Errorf("Expected default branch to be 'main', got '%s'", clonedHead.Name().Short())
	}
}

// Fuzz_Init_With_Random_Branch_Names tests that the init command works correctly
// with remote repositories that have various random branch names as their default branch
func Fuzz_Init_With_Random_Branch_Names(f *testing.F) {
	log.Println("Viper config file used:", viper.ConfigFileUsed())
    log.Println("repo-path value:", viper.GetString("repo-path"))
    log.Println("All settings:", viper.AllSettings())

	// Seed with common branch names and some random strings
	f.Add("main")
	f.Add("master")
	f.Add("develop")
	f.Add("435kb234kb")
	f.Add("branch-name-123")
	f.Add("v1.0")
	f.Add("release")

	f.Fuzz(func(t *testing.T, branchName string) {
		viper.Reset()
		// Mock home directory for isolation
		mockHomeDir(t)

		// Skip empty strings or strings with invalid characters
		if branchName == "" || len(branchName) > 100 {
			t.Skip()
		}

		// Skip branch names with invalid characters for git refs
		// Git ref naming rules: https://git-scm.com/docs/git-check-ref-format
		for _, c := range branchName {
			// Skip control characters and invalid git ref characters
			if c < 32 || c == 127 || c > 127 || // Control chars and non-ASCII
				c == ' ' || c == '~' || c == '^' || c == ':' || c == '?' ||
				c == '*' || c == '[' || c == '\\' || c == '"' {
				t.Skip()
			}
		}

		// Skip names starting or ending with '/', '.', or '-'
		if branchName[0] == '/' || branchName[0] == '.' ||
			branchName[len(branchName)-1] == '/' || branchName[len(branchName)-1] == '.' {
			t.Skip()
		}

		// Skip names containing "..", "//", or "@{"
		if len(branchName) >= 2 {
			for i := 0; i < len(branchName)-1; i++ {
				if (branchName[i] == '.' && branchName[i+1] == '.') ||
					(branchName[i] == '/' && branchName[i+1] == '/') ||
					(branchName[i] == '@' && branchName[i+1] == '{') {
					t.Skip()
				}
			}
		}

		// Create a fake remote repository with the random branch name
		remoteRepoPath := testutil.CreateRemoteRepo(t, branchName)

		// set viper remote-url to avoid issues with command args
		viper.Set("remote-url", remoteRepoPath)

		// Initialize with this remote repository
		invokeCommand(t, []string{"init"}, remoteRepoPath)

		// Verify the cloned repository exists
		isRepo := util.IsGitRepo(viper.GetString("repo-path"))
		if !isRepo {
			t.Error("repo-path is not a git repo")
		}

		// Verify the default branch matches the remote's branch
		clonedRepo, err := git.PlainOpen(viper.GetString("repo-path"))
		testutil.TBErrorIfNotNil(t, err)

		clonedHead, err := clonedRepo.Head()
		if err != nil {
			t.Fatalf("Failed to get HEAD reference: %v", err)
		}

		if clonedHead.Name().Short() != branchName {
			t.Errorf("Expected default branch to be '%s', got '%s'", branchName, clonedHead.Name().Short())
		}
	})
}

func invokeCommand(t *testing.T, args []string, remote ...string) {
	// Set default values for viper BEFORE reset to preserve some settings
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	
	repoFolder := t.TempDir()
	repoPath := filepath.Join(repoFolder, "dotfiles")
	
	viper.Reset()

	util.InitializeConfig(configFile)
	
	viper.SetDefault("repo-path", repoPath)
	viper.SetDefault("dot-home", configFile)
	if len(remote) > 0 {
		viper.SetDefault("remote-url", remote[0])
	}
	viper.SetConfigFile(configFile)
	viper.ReadInConfig()
	// viper.AutomaticEnv()

	cmd.Execute(func(c *cobra.Command) {
		c.SetArgs(args)
	})
}
