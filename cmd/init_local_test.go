package cmd

import (
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestInitLocalRepository(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("OMDOT_NON_INTERACTIVE", "")
	oldInput := inputInitRemote
	t.Cleanup(func() { inputInitRemote = oldInput; viper.Reset() })
	for _, scenario := range []string{"noninteractive", "interactive blank", "explicit blank"} {
		t.Run(scenario, func(t *testing.T) {
			viper.Reset()
			path := filepath.Join(t.TempDir(), "dotfiles")
			viper.SetConfigFile(filepath.Join(t.TempDir(), "config.json"))
			viper.Set("repo-path", path)
			command := &cobra.Command{}
			command.Flags().Bool("force", false, "")
			command.Flags().Bool("interactive", scenario != "noninteractive", "")
			command.Flags().Bool("no-interactive", scenario == "noninteractive", "")
			command.Flags().String("remote", "", "")
			if scenario == "explicit blank" {
				if err := command.Flags().Set("remote", ""); err != nil {
					t.Fatal(err)
				}
			}
			prompts := 0
			inputInitRemote = func(question, defaultValue string) (string, error) { prompts++; return "", nil }
			if err := initcmd.RunE(command, nil); err != nil {
				t.Fatal(err)
			}
			wantPrompts := 0
			if scenario == "interactive blank" {
				wantPrompts = 1
			}
			if prompts != wantPrompts {
				t.Fatalf("URL prompts = %d, want %d", prompts, wantPrompts)
			}
			repo, err := gogit.PlainOpen(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := repo.Remote("origin"); err != gogit.ErrRemoteNotFound {
				t.Fatalf("unexpected origin: %v", err)
			}
			saved := viper.New()
			saved.SetConfigFile(viper.ConfigFileUsed())
			if err := saved.ReadInConfig(); err != nil {
				t.Fatal(err)
			}
			if !saved.GetBool("initialized") || saved.GetString("remote-url") != "" {
				t.Fatal("local initialization not saved")
			}
		})
	}
}
