package util

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var homeDir, _ = os.UserHomeDir()
var DefaultRepoPath = filepath.Join(homeDir, "dotfiles")

func EnsureConfigFile() {
	file := viper.GetString("dot-home")
	if !IsFile(file) {
		dir := filepath.Dir(file)
		err := EnsureDir(dir)
		CheckIfError(err)
		err = viper.SafeWriteConfigAs(file)
		CheckIfError(err)
	}
}

func EnsureConfigFolder(file string) {
	folder := filepath.Dir(file)
	if !IsDir(folder) {
		err := EnsureDir(folder)
		CheckIfError(err)
	}
}