package util

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// EnsureDir creates a directory if it does not exist.
// The dirName parameter should only contain the directory path and should not include the filename.
func EnsureDir(dirName string) error {
	return os.MkdirAll(dirName, os.ModePerm)
}

func EnsureCreated(file string) error {
	if !IsDir(file) {
		file = filepath.Dir(file)
	}

	dirErr := EnsureDir(file)
	if dirErr != nil {
		return dirErr
	}

	_, err := os.Create(file)
	if err != nil {
		return err
	}

	return nil
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
	return os.IsNotExist(err)
}

func EnsureConfigFile(configFile string) {
	if !ConfigFileExists(viper.GetString("dot-home")) {
		viper.SafeWriteConfigAs(viper.GetString("dot-home"))
	}
}

