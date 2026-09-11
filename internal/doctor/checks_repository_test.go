package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
)

func TestRepositoryCommitIdentity(t *testing.T) {
	tests := []struct {
		name       string
		local      string
		global     string
		system     string
		wantStatus string
	}{
		{name: "missing identity", wantStatus: statusError},
		{name: "missing email", local: "[user]\nname = Local\n", wantStatus: statusError},
		{name: "missing name", local: "[user]\nemail = local@example.invalid\n", wantStatus: statusError},
		{name: "local identity", local: "[user]\nname = Local\nemail = local@example.invalid\n", wantStatus: statusOK},
		{name: "global identity", global: "[user]\nname = Global\nemail = global@example.invalid\n", wantStatus: statusOK},
		{name: "system identity", system: "[user]\nname = System\nemail = system@example.invalid\n", wantStatus: statusOK},
		{name: "merged identity", local: "[user]\nname = Local\n", global: "[user]\nemail = global@example.invalid\n", wantStatus: statusOK},
		{name: "author override", local: "[author]\nname = Author\nemail = author@example.invalid\n", wantStatus: statusOK},
		{name: "committer alone is insufficient", local: "[committer]\nname = Committer\nemail = committer@example.invalid\n", wantStatus: statusError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGitConfig(t)
			for key, contents := range map[string]string{"GIT_CONFIG_GLOBAL": tt.global, "GIT_CONFIG_SYSTEM": tt.system} {
				path := filepath.Join(t.TempDir(), "config")
				writeDoctorTestFile(t, path, contents)
				t.Setenv(key, path)
			}
			t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
			path := t.TempDir()
			repo, err := git.PlainInit(path, false)
			if err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(path, ".git", "config")
			original, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			contents := string(original) + tt.local
			writeDoctorTestFile(t, configPath, contents)
			results := checkRepository(context{repoPath: path, fix: true})
			identity := findDoctorResult(t, results, "Git commit identity")
			if identity.status != tt.wantStatus {
				t.Fatalf("identity = %+v, want status %s", identity, tt.wantStatus)
			}
			if tt.wantStatus == statusError && (!strings.Contains(identity.message, "user.name") || !strings.Contains(identity.message, "user.email")) {
				t.Fatalf("missing identity setup instructions: %s", identity.message)
			}
			after, err := os.ReadFile(configPath)
			if err != nil || string(after) != contents {
				t.Fatalf("doctor --fix changed identity config: %v", err)
			}
			// Verify the diagnosis matches an actual commit using go-git's defaults.
			worktree, err := repo.Worktree()
			if err != nil {
				t.Fatal(err)
			}
			_, err = worktree.Commit("identity check", &git.CommitOptions{AllowEmptyCommits: true})
			if (err != nil) != (tt.wantStatus == statusError) {
				t.Fatalf("commit error = %v, doctor status = %s", err, identity.status)
			}
		})
	}
}

func TestRepositoryRemote(t *testing.T) {
	for _, mode := range []string{"absent", "inaccessible", "readable"} {
		t.Run(mode, func(t *testing.T) {
			isolateGitConfig(t)
			path := t.TempDir()
			repo, err := git.PlainInit(path, false)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "absent" {
				remotePath := filepath.Join(t.TempDir(), "remote.git")
				if mode == "readable" {
					if _, err := git.PlainInit(remotePath, true); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{remotePath}}); err != nil {
					t.Fatal(err)
				}
			}
			results := checkRepository(context{repoPath: path})
			name, wantStatus := "Git remote", statusWarning
			if mode == "readable" {
				name, wantStatus = "Git remote read access", statusOK
			}
			if got := findDoctorResult(t, results, name); got.status != wantStatus {
				t.Fatalf("remote = %+v, want %s", got, wantStatus)
			}
		})
	}
}

func TestRunChecksRepositoryWithoutShells(t *testing.T) {
	for _, validIdentity := range []bool{false, true} {
		name := "missing identity"
		if validIdentity {
			name = "valid identity with remote warning"
		}
		t.Run(name, func(t *testing.T) {
			isolateGitConfig(t)
			path := t.TempDir()
			if _, err := git.PlainInit(path, false); err != nil {
				t.Fatal(err)
			}
			if validIdentity {
				global := filepath.Join(t.TempDir(), "config")
				writeDoctorTestFile(t, global, "[user]\nname = Test\nemail = test@example.invalid\n")
				t.Setenv("GIT_CONFIG_GLOBAL", global)
			}
			output, runErr := captureDoctorRun(t, path)
			if (runErr != nil) == validIdentity {
				t.Fatalf("Run error = %v, valid identity = %v", runErr, validIdentity)
			}
			for _, expected := range []string{"No shell features configured", "Git commit identity", "No origin remote configured"} {
				if !strings.Contains(output, expected) {
					t.Fatalf("missing %q in output: %s", expected, output)
				}
			}
			if strings.Contains(output, "All checks passed") || strings.Contains(output, "Warnings are optional") {
				t.Fatalf("misleading summary: %s", output)
			}
			if strings.Count(output, "Checking repository...") != 1 {
				t.Fatalf("expected one repository check: %s", output)
			}
		})
	}
}

func TestRepositoryCannotBeOpened(t *testing.T) {
	results := checkRepository(context{repoPath: t.TempDir()})
	if len(results) != 1 || results[0].name != "Git repository" || results[0].status != statusError {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func isolateGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func writeDoctorTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func findDoctorResult(t *testing.T, results []result, name string) result {
	t.Helper()
	for _, item := range results {
		if item.name == name {
			return item
		}
	}
	t.Fatalf("missing %s result: %+v", name, results)
	return result{}
}

func captureDoctorRun(t *testing.T, path string) (string, error) {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "doctor-output")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	original := os.Stdout
	os.Stdout = output
	defer func() { os.Stdout = original }()
	runErr := Run(path, nil, "oh-my-dot", false)
	data, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data), runErr
}
