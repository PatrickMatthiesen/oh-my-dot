package cmd

import (
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	gitconfig "github.com/go-git/go-git/v6/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
)

var inputInitIdentity = interactive.PromptInput

func offerInitIdentity(cmd *cobra.Command) {
	if !interactive.ShouldPrompt(cmd, false) || interactive.ValidateMode(cmd) != nil {
		return
	}
	ready, err := collectRepositoryIdentity(viper.GetString("repo-path"), inputInitIdentity)
	if err != nil {
		fileops.ColorPrintfn(fileops.Yellow, "Repository initialized; Git author setup is incomplete: %v", err)
	} else if !ready {
		fileops.ColorPrintln("Git author setup deferred. Run doctor before making commits.", fileops.Yellow)
	}
}

func collectRepositoryIdentity(path string, prompt func(string, string) (string, error)) (bool, error) {
	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return false, fmt.Errorf("open repository identity: %w", err)
	}
	defer repo.Close()
	if (&gogit.CommitOptions{}).Validate(repo) == nil {
		return true, nil
	}
	effective, err := repo.ConfigScoped(gitconfig.SystemScope)
	if err != nil {
		return false, fmt.Errorf("read Git identity: %w", err)
	}
	name, err := prompt("Name for commits in this dotfiles repository (blank to set later):", effective.User.Name)
	if err != nil {
		return false, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	email, err := prompt("Email for commits in this repository (a Git hosting noreply address is fine):", effective.User.Email)
	if err != nil {
		return false, err
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return false, nil
	}
	if strings.ContainsAny(name+email, "\r\n\x00") {
		return false, fmt.Errorf("Git author fields cannot contain line breaks or NUL characters")
	}
	local, err := repo.Config()
	if err != nil {
		return false, fmt.Errorf("read repository Git settings: %w", err)
	}
	local.User.Name, local.User.Email = name, email
	if err := repo.SetConfig(local); err != nil {
		return false, fmt.Errorf("save repository Git identity: %w", err)
	}
	fileops.ColorPrintln("Git author saved for this repository. Global Git settings were not changed.", fileops.Green)
	return true, nil
}
