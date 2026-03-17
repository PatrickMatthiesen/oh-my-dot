package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/hooks"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func checkLocalOverride(ctx context) []result {
	var results []result

	localManifestPath := shell.GetLocalManifestPath(ctx.repoPath, ctx.shellName)
	if _, err := os.Stat(localManifestPath); os.IsNotExist(err) {
		return results
	}

	if err := manifest.ValidateLocalManifest(localManifestPath); err != nil {
		return addResult(results, ctx, errorResult("Local override security", fmt.Sprintf("Unsafe: %v", err), false), nil)
	}

	if _, err := manifest.ParseManifest(localManifestPath); err != nil {
		return addResult(results, ctx, errorResult("Local override validity", fmt.Sprintf("Invalid: %v", err), false), nil)
	}

	return addResult(results, ctx, okResult("Local override"), nil)
}

func checkProfileHooks(ctx context) []result {
	var results []result

	shellConfig, ok := shell.GetShellConfig(ctx.shellName)
	if !ok {
		return addResult(results, ctx, warningResult("Profile hook", "Shell not supported for hook checking", false), nil)
	}

	profilePath, err := shell.ResolveProfilePath(shellConfig)
	if err != nil {
		return addResult(results, ctx, warningResult("Profile hook", fmt.Sprintf("Cannot resolve profile: %v", err), false), nil)
	}

	hasHook, err := hooks.HasHook(profilePath)
	if err != nil {
		return addResult(results, ctx, errorResult("Profile hook", fmt.Sprintf("Cannot check hook: %v", err), false), nil)
	}

	if !hasHook {
		return addResult(
			results,
			ctx,
			errorResult("Profile hook", fmt.Sprintf("Hook missing in %s", profilePath), true),
			func() (string, error) {
				initScriptPath, err := shell.GetInitScriptPath(ctx.repoPath, ctx.shellName)
				if err != nil {
					return "", fmt.Errorf("resolve init script path: %w", err)
				}

				hookContent := hooks.GenerateHook(ctx.shellName, initScriptPath)
				added, err := hooks.InsertHook(profilePath, hookContent)
				if err != nil {
					return "", fmt.Errorf("insert hook: %w", err)
				}
				if !added {
					return "", fmt.Errorf("hook was not inserted")
				}

				return fmt.Sprintf("Added hook to %s", profilePath), nil
			},
		)
	}

	return addResult(results, ctx, okResult("Profile hook"), nil)
}

func checkInitScriptSyntax(ctx context) []result {
	var results []result

	initScriptPath, err := shell.GetInitScriptPath(ctx.repoPath, ctx.shellName)
	if err != nil {
		return results
	}

	if _, err := os.Stat(initScriptPath); os.IsNotExist(err) {
		return addResult(
			results,
			ctx,
			errorResult("Init script", fmt.Sprintf("File missing: %s", initScriptPath), true),
			func() (string, error) {
				if err := shell.RegenerateInitScript(ctx.repoPath, ctx.shellName); err != nil {
					return "", fmt.Errorf("regenerate init script: %w", err)
				}

				return fmt.Sprintf("Generated %s", initScriptPath), nil
			},
		)
	}

	results = addResult(results, ctx, okResult("Init script"), nil)

	cmd := initScriptSyntaxCommand(ctx.shellName, initScriptPath)
	if cmd == nil {
		return results
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		syntaxMessage := fmt.Sprintf("Syntax error: %s", strings.TrimSpace(string(output)))
		return addResult(
			results,
			ctx,
			errorResult("Init script syntax", syntaxMessage, true),
			func() (string, error) {
				if err := shell.RegenerateInitScript(ctx.repoPath, ctx.shellName); err != nil {
					return "", fmt.Errorf("regenerate init script: %w", err)
				}

				return fmt.Sprintf("Regenerated %s", initScriptPath), nil
			},
		)
	}

	return addResult(results, ctx, okResult("Init script syntax"), nil)
}

func initScriptSyntaxCommand(shellName, initScriptPath string) *exec.Cmd {
	switch shellName {
	case "bash":
		return exec.Command("bash", "-n", initScriptPath)
	case "zsh":
		return exec.Command("zsh", "-n", initScriptPath)
	case "fish":
		return exec.Command("fish", "-n", initScriptPath)
	case "posix":
		return exec.Command("sh", "-n", initScriptPath)
	default:
		return nil
	}
}
