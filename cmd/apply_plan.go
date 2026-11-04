package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/hooks"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/repopath"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

type applyAction struct {
	kind    string
	source  string
	target  string
	shell   string
	state   string
	warning string
	err     error
}

func inspectApplyLink(source, target string, allowOutsideHome bool) applyAction {
	action := applyAction{kind: "link", source: source, target: target, state: "create"}
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		action.err = fmt.Errorf("inspect source %s: %w", source, err)
		return action
	}
	if !sourceInfo.Mode().IsRegular() {
		action.err = fmt.Errorf("source %s must be a regular file", source)
		return action
	}
	targetInfo, err := os.Lstat(target)
	if err == nil {
		linkedInfo, statErr := os.Stat(target)
		if statErr == nil && os.SameFile(sourceInfo, linkedInfo) {
			action.state = "already correct"
			return action
		}
		action.err = fmt.Errorf("target %s already exists and is not the managed file; move it aside or add it explicitly before applying", target)
		if targetInfo.Mode()&os.ModeSymlink != 0 {
			action.err = fmt.Errorf("target %s is an unrelated or broken symbolic link; move it aside before applying", target)
		}
		return action
	}
	if !os.IsNotExist(err) {
		action.err = fmt.Errorf("inspect target %s: %w", target, err)
		return action
	}
	parent, err := os.Stat(filepath.Dir(target))
	if err != nil {
		action.err = fmt.Errorf("target directory for %s is unavailable: %w", target, err)
		return action
	}
	if !parent.IsDir() {
		action.err = fmt.Errorf("target parent %s is not a directory", filepath.Dir(target))
		return action
	}
	warnings, err := applyTargetWarnings(target)
	if err != nil {
		action.err = err
		return action
	}
	if !allowOutsideHome && containsApplyWarning(warnings, "target is outside your home directory") {
		action.err = fmt.Errorf("refusing to link %s outside your home directory; use --allow-outside-home to allow this target", target)
		return action
	}
	action.warning = strings.Join(warnings, "; ")
	return action
}

func inspectApplyHook(shellName, source, target string) applyAction {
	return inspectPreparedApplyHook(shellName, source, target, false)
}

func inspectPreparedApplyHook(shellName, source, target string, prepared bool) applyAction {
	action := applyAction{kind: "hook", shell: shellName, source: source, target: target, state: "create"}
	if !prepared {
		info, err := os.Stat(source)
		if err != nil {
			action.err = fmt.Errorf("inspect %s init script: %w", shellName, err)
			return action
		}
		if !info.Mode().IsRegular() {
			action.err = fmt.Errorf("%s init script is not a regular file", shellName)
			return action
		}
	}
	exists, err := hooks.HasHook(target)
	if err != nil {
		action.err = fmt.Errorf("inspect %s profile: %w", shellName, err)
		return action
	}
	if exists {
		matches, err := hooks.HookMatches(target, hooks.GenerateHook(shellName, source))
		if err != nil {
			action.err = err
			return action
		}
		if matches {
			action.state = "already correct"
		} else {
			action.state = "update"
		}
		return action
	}
	// InsertHook creates missing parent directories. Validate existing ancestors
	// now so preview and execution reject the same obvious path conflicts.
	for parent := filepath.Dir(target); ; parent = filepath.Dir(parent) {
		info, err := os.Stat(parent)
		if err == nil {
			if !info.IsDir() {
				action.err = fmt.Errorf("profile parent %s is not a directory", parent)
			}
			break
		}
		if !os.IsNotExist(err) {
			action.err = fmt.Errorf("inspect profile directory %s: %w", parent, err)
			break
		}
		if filepath.Dir(parent) == parent {
			action.err = fmt.Errorf("profile parent %s is unavailable", parent)
			break
		}
	}
	return action
}

func planApply(repoPath string, noShell, allowOutsideHome bool) ([]applyAction, error) {
	links, err := symlink.GetLinkings()
	if err != nil {
		return nil, fmt.Errorf("retrieve linkings: %w", err)
	}
	if err := symlink.ValidateLinkings(repoPath, links); err != nil {
		return nil, fmt.Errorf("validate linkings: %w", err)
	}
	keys := make([]string, 0, len(links))
	for key := range links {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	actions := make([]applyAction, 0, len(keys))
	for _, key := range keys {
		source, err := repopath.Resolve(repoPath, key)
		if err != nil {
			return nil, fmt.Errorf("resolve repository file: %w", err)
		}
		target, err := fileops.ExpandPath(links[key])
		if err != nil {
			return nil, fmt.Errorf("expand target %s: %w", links[key], err)
		}
		actions = append(actions, inspectApplyLink(source, target, allowOutsideHome))
	}
	if noShell {
		return actions, nil
	}
	shells, err := shell.ListShellsWithFeatures(repoPath)
	if err != nil {
		return nil, fmt.Errorf("list shells: %w", err)
	}
	sort.Strings(shells)
	for _, shellName := range shells {
		config, ok := shell.GetShellConfig(shellName)
		if !ok || !shell.IsShellExecutableAvailable(shellName) {
			actions = append(actions, applyAction{kind: "hook", shell: shellName, state: "unavailable", warning: "shell executable is not installed"})
			continue
		}
		if err := shell.ValidateRuntimeScripts(repoPath, shellName); err != nil {
			actions = append(actions, applyAction{kind: "runtime", shell: shellName, err: fmt.Errorf("prepare %s runtime: %w", shellName, err)})
			continue
		}
		actions = append(actions, applyAction{kind: "runtime", shell: shellName, state: "update"})
		// PowerShell discovery uses -NoProfile and reads only its profile path. It
		// never sources the user's profile or any configured feature.
		target, err := shell.ResolveProfilePath(config)
		if err != nil {
			actions = append(actions, applyAction{kind: "hook", shell: shellName, err: fmt.Errorf("resolve %s profile: %w", shellName, err)})
			continue
		}
		source, err := shell.GetInitScriptPath(repoPath, shellName)
		if err != nil {
			actions = append(actions, applyAction{kind: "hook", shell: shellName, target: target, err: err})
			continue
		}
		actions = append(actions, inspectPreparedApplyHook(shellName, source, target, true))
	}
	for _, shellName := range shells {
		warnings, err := shell.FeatureRequirementWarnings(repoPath, shellName)
		if err != nil {
			return nil, err
		}
		for _, warning := range warnings {
			actions = append(actions, applyAction{kind: "prerequisite", shell: shellName, state: "not installed", warning: warning})
		}
	}
	return actions, nil
}

func runApply(cmd *cobra.Command, _ []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noShell, _ := cmd.Flags().GetBool("no-shell")
	allowOutsideHome, _ := cmd.Flags().GetBool("allow-outside-home")
	actions, err := planApply(viper.GetString("repo-path"), noShell, allowOutsideHome)
	if err != nil {
		return err
	}
	var failures []error
	for _, action := range actions {
		name := action.target
		if name == "" {
			name = action.shell
		}
		if action.err != nil {
			failures = append(failures, action.err)
			fmt.Fprintf(cmd.OutOrStdout(), "Conflict: %s\n", action.err)
			continue
		}
		if action.warning != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s: %s\n", name, action.warning)
		}
		if dryRun || action.state != "create" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s (%s)\n", action.kind, name, action.state)
		}
	}
	// Known conflicts are reported together before making any changes.
	if len(failures) > 0 {
		return fmt.Errorf("apply has %d conflict(s): %w", len(failures), errors.Join(failures...))
	}
	if dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Preview complete. No managed files were changed.")
		return nil
	}
	addedHooks := 0
	failedRuntime := map[string]bool{}
	for _, action := range actions {
		if action.state != "create" && action.state != "update" {
			continue
		}
		if action.kind == "runtime" {
			if err := shell.RegenerateInitScript(viper.GetString("repo-path"), action.shell); err != nil {
				failedRuntime[action.shell] = true
				failures = append(failures, fmt.Errorf("regenerate %s runtime: %w", action.shell, err))
			}
			continue
		}
		if failedRuntime[action.shell] {
			continue
		}
		if action.kind == "link" {
			current := inspectApplyLink(action.source, action.target, allowOutsideHome)
			if current.err != nil {
				failures = append(failures, current.err)
				continue
			}
			if current.state == "already correct" {
				continue
			}
			if err := createDotfileLink(action.source, action.target); err != nil {
				failures = append(failures, fmt.Errorf("link %s: %w", action.target, err))
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Linked %s\n", action.target)
		} else {
			current := inspectApplyHook(action.shell, action.source, action.target)
			if current.err != nil {
				failures = append(failures, current.err)
				continue
			}
			if current.state == "already correct" {
				continue
			}
			if current.state == "update" {
				if err := hooks.ReplaceHook(action.target, hooks.GenerateHook(action.shell, action.source)); err != nil {
					failures = append(failures, err)
					continue
				}
				addedHooks++
				fmt.Fprintf(cmd.OutOrStdout(), "Hook updated for %s: %s\n", action.shell, action.target)
				continue
			}
			added, err := hooks.InsertHook(action.target, hooks.GenerateHook(action.shell, action.source))
			if err != nil {
				failures = append(failures, fmt.Errorf("install %s hook: %w", action.shell, err))
				continue
			}
			if added {
				addedHooks++
				fmt.Fprintf(cmd.OutOrStdout(), "Hook added for %s: %s\n", action.shell, action.target)
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("apply completed with %d failure(s); successful changes were kept: %w", len(failures), errors.Join(failures...))
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Done!")
	if addedHooks > 0 || hasAvailableHook(actions) {
		missingPrerequisites := false
		for _, action := range actions {
			if action.kind == "prerequisite" {
				missingPrerequisites = true
				break
			}
		}
		if missingPrerequisites {
			fmt.Fprintln(cmd.OutOrStdout(), "Install the listed prerequisites, then open a new shell session. Apply does not install dependencies.")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Open a new shell session to load the configured features.")
		}
	}
	return nil
}

func hasAvailableHook(actions []applyAction) bool {
	for _, action := range actions {
		if action.kind == "hook" && action.err == nil && action.state != "unavailable" {
			return true
		}
	}
	return false
}
