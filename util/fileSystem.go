package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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

func IsFile(configFile string) bool {
	fi, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		return false
	}
	return !fi.IsDir()
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func ExpandPath(path string) (string, error) {
	if len(path) == 0 {
		return "", fmt.Errorf("could not expand empty path")
	}
	if path[:1] == "~" {
		return filepath.Join(homeDir, string(filepath.Separator), path[1:]), nil
	}
	return path, nil
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}
	return nil
}

func CopyFileToDir(src, dst string) error {
	return CopyFile(src, dst+"/"+filepath.Base(src))
}

