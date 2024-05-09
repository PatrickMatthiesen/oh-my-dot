package main

import (
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/viper"
)

func main() {
	// fmt.Println(filepath.Abs("~\\dotfiles"))
	// fmt.Println(util.IsDir("~\\dotfiles"))
	// fmt.Println(util.IsDir("C:/Users/patr7/Desktop/Ting/My projects/oh-my-dot"))
	// fmt.Println(util.IsDir("C:\\Users\\patr7\\Desktop\\Ting\\My projects\\oh-my-dot"))
	// fmt.Println(util.ExpandPath("~\\dotfiles"))
	// stat, err := os.Stat("")
	// fmt.Println(stat, err)

	// fmt.Println(filepath.Abs(filepath.ToSlash("C:\\Users\\patr7\\Desktop\\Ting\\My projects\\oh-my-dot")))
	// fmt.Println(filepath.ToSlash("C:\\Users\\patr7\\Desktop\\Ting\\My projects\\oh-my-dot"))
	// fmt.Println(filepath.Join("C:/Users/patr7/Desktop/Ting/My projects/oh-my-dot"))

	// return
	home, err := os.UserHomeDir()
	util.CheckIfErrorWithMessage(err, "Error getting home directory")

	configFile := filepath.Join(home, ".oh-my-dot", "config.json")

	go util.EnsureConfigFolder(configFile)

	viper.SetDefault("dot-home", configFile)
	viper.SetDefault("repo-path", filepath.Join(home, "dotfiles"))
	// TODO: Set a viper config variable for the files folder in the repo-path, and update the strings in the util/repo.go file to use this variable

	viper.SetConfigFile(configFile)

	viper.ReadInConfig()

	viper.AutomaticEnv()
	cmd.Execute()

	//TODO: make execute return an error. Redirect the error to a log file or print it to the console if env var or flag is set.
	//Update the commands to use RunE and tests to check for the error
}
