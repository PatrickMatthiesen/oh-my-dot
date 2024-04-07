package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Linkings map[string]string

func AddLinking(name, link string) error {
	links, err := GetLinkings()
	if err != nil {
		return fmt.Errorf("error getting links: %w", err)
	}
	links[name] = link
	return SaveLinkings(links)
}

func RemoveLinking(name string) error {
	links, err := GetLinkings()
	if err != nil {
		return fmt.Errorf("error getting links for removal: %w", err)
	}
	
	delete(links, name)
	
	return SaveLinkings(links)
}

func GetLinkings() (Linkings, error) {
	linkFile := filepath.Join(viper.GetString("repo-path"), "linkings.json")
	if !IsFile(linkFile) {
		return Linkings{}, nil
	}
	
	file, err := os.ReadFile(linkFile)
	if err != nil {
		return nil, fmt.Errorf("error reading links file: %w", err)
	}
	
	var links Linkings
	err = json.Unmarshal(file, &links)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling links: %w", err)
	}
	
	return links, nil
}

func SaveLinkings(links Linkings) error {
	linkFile := filepath.Join(viper.GetString("repo-path"), "linkings.json")
	
	file, err := json.MarshalIndent(links, "", "  ")
	if err != nil{
		return fmt.Errorf("error marshalling links: %w", err)
	}
	
	err = os.WriteFile(linkFile, file, 0644)
	if err != nil {
		return fmt.Errorf("error writing links to file: %w", err)
	}

	err = StageChange("linkings.json")
	if err != nil {
		return fmt.Errorf("error staging changes: %w", err)
	}

	return nil
}

func BuildLinkPath(file string) (string, error) {
	absPath, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}

	absPath, found := strings.CutPrefix(absPath, home)
	if found {
		absPath = "~" + absPath
	}
	// relPath, err := filepath.Rel(viper.GetString("repo-path"), absPath)
	// if err != nil {
	// 	return "", fmt.Errorf("error getting relative path: %w", err)
	// }

	return filepath.ToSlash(absPath), nil
}