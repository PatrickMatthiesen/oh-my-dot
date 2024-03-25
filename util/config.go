package util

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var homeDir, _ = os.UserHomeDir()
var DefaultRepoPath = filepath.Join(homeDir, "dotfiles")

// EnsureDir creates a directory if it does not exist.
// The dirName parameter should only contain the directory path and should not include the filename.
func EnsureDir(dirName string) error {
	return os.MkdirAll(dirName, os.ModePerm)
}

func IsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func ConfigFileExists(configFile string) bool {
	_, err := os.Stat(configFile)
	return !os.IsNotExist(err)
}

func EnsureConfigFile() {
	file := viper.GetString("dot-home")
	if !ConfigFileExists(file) {
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