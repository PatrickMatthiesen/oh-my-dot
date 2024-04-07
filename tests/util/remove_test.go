package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"
)

func Fuzz_RemoveFile(f *testing.F) {
	f.Add("", "")
	f.Add(".\\", "")
	f.Add("", "\\")
	f.Add("", "/")
	f.Add("/", "")
	f.Add("./", "")

	f.Fuzz(func(t *testing.T, testPrefix string, testSufix string) {
		r, err := SetupTestRepo(t)
		util.CheckIfError(err)

		// create files dir
		err = os.MkdirAll(filepath.Join(viper.GetString("repo-path"), "files"), os.ModePerm)
		// TODO: Remove when fix has been implemented in go-git https://github.com/go-git/go-git/pull/1050
		paddingFile, err := os.CreateTemp(filepath.Join(viper.GetString("repo-path"),"files"), "keepsRepoFromBeingEmpty.txt")
		if err != nil {
			t.Error(err)
		}
		paddingFile.WriteString("test")
		w, _ := r.Worktree()
		w.Add("./files")
		_, err = w.Commit("Test commit", &git.CommitOptions{})
		if err != nil {
			t.Error(err)
		}
		defer paddingFile.Close()

		// Create a file
		tempSourceDir := t.TempDir()
		file, err := os.CreateTemp(tempSourceDir, "test.txt")
		if err != nil {
			t.Error(err)
		}
		file.WriteString("test")
		defer file.Close()

		// Link the file to the git repo
		err = util.LinkAndAddFile(file.Name())
		util.CheckIfError(err)

		commits, err := r.Log(&git.LogOptions{})
		util.CheckIfError(err)
		commit, err := commits.Next()
		util.CheckIfError(err)
		files, err := commit.Files()
		util.CheckIfError(err)
		_, err = files.Next()
		util.CheckIfError(err)
		t.Run("Test config push", func(t *testing.T) {
			// Make a bare repo to push to
			_, err := git.PlainInit(viper.GetString("remote-url"), true)
			util.CheckIfError(err)
			
			// Push the repo
			err = util.PushRepo()
			util.CheckIfError(err)
		})
		
		// Remove the file from the git repo
		err = util.RemoveFile(testPrefix + filepath.Base(file.Name()) + testSufix)
		util.CheckIfError(err)

		// Check if the file exists in the git repo
		// _, err = os.Stat(testFilePath)
		// if err == nil {
		// 	t.Error("File was not removed from the git repo")
		// }
	})
}