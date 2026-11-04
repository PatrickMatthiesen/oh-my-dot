package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestSharedFeatureCommitExcludesPrivateRuntimeAndLegacyScripts(t *testing.T) {
	repoPath := setupCompactableRepo(t)
	if err := shell.AddFeatureToShell(repoPath, "bash", "oh-my-posh", "eager", nil, false, map[string]any{"theme": "pure"}); err != nil {
		t.Fatal(err)
	}
	if committed, err := StageAndCommitShellFeatureChanges("Add shared prompt"); err != nil || !committed {
		t.Fatalf("baseline commit: %v, %v", committed, err)
	}
	const privateName = "OMD_PRIVATE_FEATURE_2048"
	const privateOption = "OMD_PRIVATE_OPTION_2048"
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{
		{Name: "oh-my-posh", Options: map[string]any{"theme": privateOption}},
		{Name: privateName, Strategy: "eager"},
	}}
	if err := manifest.WriteLocalManifest(shell.GetLocalManifestPath(repoPath, "bash"), local); err != nil {
		t.Fatal(err)
	}
	// Keep a user-owned local-only implementation in the older shared-directory
	// location. Regeneration must preserve it while automatic commits exclude it.
	legacy, _ := shell.GetFeatureFilePath(repoPath, "bash", privateName)
	if err := os.WriteFile(legacy, []byte("# "+privateOption+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := shell.RegenerateInitScript(repoPath, "bash"); err != nil {
		t.Fatal(err)
	}
	if err := shell.AddFeatureToShell(repoPath, "bash", "core-aliases", "eager", nil, false, nil); err != nil {
		t.Fatal(err)
	}
	if committed, err := StageAndCommitShellFeatureChanges("Add unrelated shared aliases"); err != nil || !committed {
		t.Fatalf("unrelated shared commit: %v, %v", committed, err)
	}
	repository, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	head, err := repository.Head()
	if err != nil {
		t.Fatal(err)
	}
	commit, err := repository.CommitObject(head.Hash())
	if err != nil {
		t.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatal(err)
	}
	if err := tree.Files().ForEach(func(file *object.File) error {
		contents, err := file.Contents()
		if err != nil {
			return err
		}
		for _, sentinel := range []string{privateName, privateOption} {
			if strings.Contains(file.Name, sentinel) || strings.Contains(contents, sentinel) {
				t.Errorf("private sentinel %q appears in tracked file %s", sentinel, file.Name)
			}
		}
		if shell.IsDeviceLocalShellPath(file.Name) {
			t.Errorf("private runtime file was committed: %s", file.Name)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	private, _ := shell.GetLocalFeatureFilePath(repoPath, "bash", "oh-my-posh")
	if data, err := os.ReadFile(private); err != nil || !strings.Contains(string(data), privateOption) {
		t.Fatalf("private option variant missing: %v", err)
	}
	if data, err := os.ReadFile(legacy); err != nil || !strings.Contains(string(data), privateOption) {
		t.Fatalf("legacy user implementation was changed: %v", err)
	}
}

func TestTrackedPrivateRuntimeRejectedBeforeRegeneration(t *testing.T) {
	for _, name := range []string{"enabled.local.json", "init.local.sh", ".local/generated.json", "features/private-legacy.sh"} {
		t.Run(name, func(t *testing.T) {
			repoPath := setupCompactableRepo(t)
			if err := shell.InitializeShellDirectory(repoPath, "bash"); err != nil {
				t.Fatal(err)
			}
			local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: "private-legacy"}}}
			if err := manifest.WriteLocalManifest(shell.GetLocalManifestPath(repoPath, "bash"), local); err != nil {
				t.Fatal(err)
			}
			relative := "omd-shells/bash/" + name
			path := filepath.Join(repoPath, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if name != "enabled.local.json" {
				if err := os.WriteFile(path, []byte("private sentinel\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			repository, err := gogit.PlainOpen(repoPath)
			if err != nil {
				t.Fatal(err)
			}
			defer repository.Close()
			worktree, err := repository.Worktree()
			if err != nil {
				t.Fatal(err)
			}
			if err := worktree.AddWithOptions(&gogit.AddOptions{Path: relative, SkipStatus: true}); err != nil {
				t.Fatal(err)
			}
			public, _ := shell.GetInitScriptPath(repoPath, "bash")
			before, err := os.ReadFile(public)
			if err != nil {
				t.Fatal(err)
			}
			if err := shell.RegenerateInitScript(repoPath, "bash"); err == nil || !strings.Contains(err.Error(), "already tracked") {
				t.Fatalf("tracked private state accepted: %v", err)
			}
			after, err := os.ReadFile(public)
			if err != nil || string(before) != string(after) {
				t.Fatalf("public runtime changed before rejection: %v", err)
			}
			if committed, err := StageAndCommitShellFeatureChanges("must refuse"); err == nil || committed {
				t.Fatalf("tracked private state committed: %v, %v", committed, err)
			}
		})
	}
}
