package cmd

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
)

func TestInitIdentityRepositoryScopeAndDeferral(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	global := filepath.Join(t.TempDir(), "global")
	if err := os.WriteFile(global, []byte("# unchanged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	for _, scenario := range []string{"set", "defer"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			repo, err := gogit.PlainInit(root, false)
			if err != nil {
				t.Fatal(err)
			}
			repo.Close()
			answers := []string{"Example User", "user@example.invalid"}
			if scenario == "defer" {
				answers = []string{""}
			}
			calls := 0
			ready, err := collectRepositoryIdentity(root, func(string, string) (string, error) { answer := answers[calls]; calls++; return answer, nil })
			if err != nil || ready != (scenario == "set") {
				t.Fatalf("identity setup %v %v", ready, err)
			}
			repo, err = gogit.PlainOpen(root)
			if err != nil {
				t.Fatal(err)
			}
			defer repo.Close()
			cfg, err := repo.Config()
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "set" && (cfg.User.Name != "Example User" || cfg.User.Email != "user@example.invalid") {
				t.Fatal("missing repository identity")
			}
			if scenario == "defer" && (cfg.User.Name != "" || cfg.User.Email != "") {
				t.Fatal("partial identity written")
			}
			data, _ := os.ReadFile(global)
			if string(data) != "# unchanged\n" {
				t.Fatal("global identity changed")
			}
		})
	}
}
