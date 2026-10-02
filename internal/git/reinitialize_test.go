package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
)

func TestReinitializeRepository(t *testing.T) {
	for _, scenario := range []string{"recover empty", "preserve committed files", "reject untracked files", "failed remote", "empty remote"} {
		t.Run(scenario, func(t *testing.T) {
			path := t.TempDir()
			repo, err := gogit.PlainInit(path, false)
			if err != nil {
				t.Fatal(err)
			}
			oldURL := filepath.Join(t.TempDir(), "old-remote")
			if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{oldURL}}); err != nil {
				t.Fatal(err)
			}
			cfg, err := repo.Config()
			if err != nil {
				t.Fatal(err)
			}
			cfg.User.Name, cfg.User.Email = "Local Name", "local@example.invalid"
			if err := repo.SetConfig(cfg); err != nil {
				t.Fatal(err)
			}
			selected := createRemoteRepo(t)
			switch scenario {
			case "preserve committed files":
				commitFile(t, path, "local.txt", "Local history")
			case "reject untracked files":
				if err := os.WriteFile(filepath.Join(path, "initial.txt"), []byte("keep"), 0644); err != nil {
					t.Fatal(err)
				}
			case "empty remote":
				selected = t.TempDir()
				if _, err := gogit.PlainInit(selected, true); err != nil {
					t.Fatal(err)
				}
			case "failed remote":
				selected = filepath.Join(t.TempDir(), "missing")
			}
			err = ReinitializeRepository(path, selected)
			wantError := scenario == "reject untracked files" || scenario == "failed remote" || scenario == "empty remote"
			if (err != nil) != wantError {
				t.Fatalf("reinitialize error = %v, want error %v", err, wantError)
			}
			cfg, err = repo.Config()
			if err != nil {
				t.Fatal(err)
			}
			wantURL := selected
			if wantError {
				wantURL = oldURL
			}
			if cfg.Remotes["origin"].URLs[0] != wantURL {
				t.Fatalf("origin = %v, want %s", cfg.Remotes["origin"].URLs, wantURL)
			}
			if cfg.User.Name != "Local Name" || cfg.User.Email != "local@example.invalid" {
				t.Fatal("local identity changed")
			}
			if scenario == "recover empty" {
				data, err := os.ReadFile(filepath.Join(path, "initial.txt"))
				if err != nil || string(data) != "initial.txt" {
					t.Fatalf("remote file = %q, %v", data, err)
				}
				head, err := repo.Head()
				if err != nil || head.Name().Short() != "main" {
					t.Fatalf("default branch = %v, %v", head, err)
				}
			}
			if scenario == "preserve committed files" {
				if _, err := os.Stat(filepath.Join(path, "local.txt")); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "reject untracked files" {
				data, err := os.ReadFile(filepath.Join(path, "initial.txt"))
				if err != nil || string(data) != "keep" {
					t.Fatalf("local file changed: %q, %v", data, err)
				}
			}
		})
	}
}
