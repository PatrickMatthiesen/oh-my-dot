package fileops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDir creates a directory if it does not exist.
// The dirName parameter should only contain the directory path and should not include the filename.
func EnsureDir(dirName string) error {
	return os.MkdirAll(dirName, 0700)
}

func IsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func IsFile(file string) bool {
	is, _ := IsFileErr(file)
	return is
}

func IsFileErr(file string) (bool, error) {
	fi, err := os.Stat(file)
	if err != nil {
		return false, err
	}
	return !fi.IsDir(), nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func ExpandPath(path string) (string, error) {
	if len(path) == 0 {
		return "", fmt.Errorf("could not expand empty path")
	}
	if path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		// Support: "~", "~/sub", "~\\sub"
		if len(path) == 1 {
			return homeDir, nil
		}
		// Trim optional path separators after ~
		rest := path[1:]
		rest = strings.TrimLeft(rest, "/\\")
		// Join using OS-specific separator
		joined := filepath.Join(homeDir, rest)
		return joined, nil
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
	return CopyFile(src, filepath.Join(dst, filepath.Base(src)))
}
