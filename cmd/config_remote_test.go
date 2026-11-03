package cmd

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/spf13/viper"
)

func TestConfigRemoteURL(t *testing.T) {
	for _, scenario := range []string{"add", "replace", "remove", "write failure"} {
		t.Run(scenario, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			path := t.TempDir()
			repo, err := gogit.PlainInit(path, false)
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "add" {
				if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{"old"}}); err != nil {
					t.Fatal(err)
				}
				cfg, err := repo.Config()
				if err != nil {
					t.Fatal(err)
				}
				cfg.Branches["main"] = &config.Branch{Name: "main", Remote: "origin", Merge: plumbing.NewBranchReferenceName("main")}
				if err := repo.SetConfig(cfg); err != nil {
					t.Fatal(err)
				}
				viper.Set("remote-url", "old")
			}
			configPath := filepath.Join(t.TempDir(), "config.json")
			if scenario == "write failure" {
				configPath = filepath.Join(t.TempDir(), "missing", "config.json")
			}
			viper.SetConfigFile(configPath)
			viper.Set("repo-path", path)
			value := "new"
			if scenario == "remove" {
				value = ""
			}
			err = updateConfigValue("remote-url", value)
			if scenario == "write failure" {
				if err == nil {
					t.Fatal("expected write failure")
				}
				origin, err := repo.Remote("origin")
				if err != nil || origin.Config().URLs[0] != "old" {
					t.Fatalf("origin not restored: %v, %v", origin, err)
				}
				if viper.GetString("remote-url") != "old" {
					t.Fatal("application config not restored")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			origin, err := repo.Remote("origin")
			if scenario == "remove" {
				if err != gogit.ErrRemoteNotFound {
					t.Fatalf("origin remains: %v", err)
				}
				cfg, err := repo.Config()
				if err != nil {
					t.Fatal(err)
				}
				if cfg.Branches["main"].Remote != "" {
					t.Fatal("removed origin still tracked")
				}
			} else if err != nil || origin.Config().URLs[0] != value {
				t.Fatalf("origin = %v, %v", origin, err)
			}
			saved := viper.New()
			saved.SetConfigFile(configPath)
			if err := saved.ReadInConfig(); err != nil {
				t.Fatal(err)
			}
			if saved.GetString("remote-url") != value {
				t.Fatal("saved URL mismatch")
			}
			if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
				t.Fatal(err)
			}
		})
	}
}
