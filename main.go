package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/config"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)

func main() {
	home, err := os.UserHomeDir()
	fileops.CheckIfErrorWithMessage(err, "Error getting home directory")

	configFile := os.Getenv("OH_MY_DOT_CONFIG")
	if configFile == "" {
		configFile = filepath.Join(home, ".oh-my-dot", "config.json")
	}

	if err := config.LoadConfig(configFile, shouldCreateConfig(os.Args[1:])); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	viper.SetDefault("dot-home", configFile)
	viper.SetDefault("repo-path", filepath.Join(home, "dotfiles"))
	viper.SetDefault("restrict-paths-to-home", false) // Allow system-wide paths by default
	// TODO: Set a viper config variable for the files folder in the repo-path, and update the strings in the util/repo.go file to use this variable

	viper.SetConfigFile(configFile)

	viper.AutomaticEnv()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}

	//TODO: make execute return an error. Redirect the error to a log file or print it to the console if env var or flag is set.
	//Update the commands to use RunE and tests to check for the error
}

// Only commands that establish settings create a configuration file. A missing
// config for inspection or an invalid command should never write to the home.
func shouldCreateConfig(args []string) bool {
	var words []string
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return false
		}
		if !strings.HasPrefix(arg, "-") {
			words = append(words, arg)
		}
	}
	return len(words) > 0 && ((words[0] == "init" || words[0] == "i") || words[0] == "config" && len(words) > 1 && (words[1] == "set" || words[1] == "update"))
}
