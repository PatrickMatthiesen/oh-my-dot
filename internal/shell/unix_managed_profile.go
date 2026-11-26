package shell

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

// ProfileUnixShellFeatures sources each enabled feature in a fresh,
// noninteractive shell. Individual failures are recorded on the timings.
func ProfileUnixShellFeatures(ctx context.Context, shellName, repoPath string) (ShellProfileReport, error) {
	ctx, cancel := context.WithTimeout(ctx, shellProfileTimeout)
	defer cancel()

	report := ShellProfileReport{Shell: shellName, Profiles: []ProfileTiming{}, Warnings: []string{}}
	var shellArgs []string
	switch shellName {
	case "bash":
		shellArgs = []string{"--noprofile", "--norc", "-c"}
	case "zsh":
		shellArgs = []string{"-f", "-c"}
	case "fish":
		shellArgs = []string{"--no-config", "-c"}
	case "posix":
		shellArgs = []string{"-c"}
	default:
		return report, fmt.Errorf("unsupported Unix shell: %s", shellName)
	}
	if strings.TrimSpace(repoPath) == "" {
		return report, fmt.Errorf("oh-my-dot repository path is not configured")
	}
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return report, fmt.Errorf("resolve oh-my-dot repository path: %w", err)
	}
	if err := unixManagedProfileContextError(ctx, shellName); err != nil {
		return report, err
	}

	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if errors.Is(err, fs.ErrNotExist) {
		return report, unixManagedProfileContextError(ctx, shellName)
	}
	if err != nil {
		return report, fmt.Errorf("load shared %s manifest: %w", shellName, err)
	}
	local, err := strictLocalManifest(GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return report, err
	}
	shellRoot := GetShellDirectory(repoPath, shellName)
	for _, feature := range manifest.MergeManifests(base, local).Features {
		if ctx.Err() != nil {
			break
		}
		if feature.Disabled {
			continue
		}
		if report.Executable == "" {
			report.Executable, err = resolveUnixProfileExecutable(shellName)
			if err != nil {
				return report, err
			}
		}
		timing := ManagedFeatureTiming{FeatureTiming: FeatureTiming{Name: feature.Name, Strategy: feature.Strategy}}
		if timing.Strategy == "" {
			timing.Strategy = "eager"
		}
		timing.Path, err = effectiveFeatureFilePath(repoPath, shellName, base, feature)
		if err == nil {
			err = measureUnixManagedFeature(ctx, report.Executable, shellArgs, shellName, shellRoot, &timing)
		}
		if err != nil {
			timing.Error = err.Error()
		}
		report.ManagedFeatures = append(report.ManagedFeatures, timing)
		report.TotalMS += timing.DurationMS
	}
	sort.SliceStable(report.ManagedFeatures, func(i, j int) bool {
		return report.ManagedFeatures[i].DurationMS > report.ManagedFeatures[j].DurationMS
	})
	return report, unixManagedProfileContextError(ctx, shellName)
}

func unixManagedProfileContextError(ctx context.Context, shellName string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s feature profiling interrupted: %w", shellName, err)
	}
	return nil
}

func measureUnixManagedFeature(ctx context.Context, executable string, shellArgs []string, shellName, shellRoot string, timing *ManagedFeatureTiming) error {
	info, err := os.Stat(timing.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("feature script not found: %w", err)
	}
	if err != nil {
		return fmt.Errorf("inspect feature script: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("feature script is not a regular file")
	}

	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return fmt.Errorf("resolve feature timing directory: %w", err)
	}
	dir, err := os.MkdirTemp(tempRoot, "omd-unix-profile-")
	if err != nil {
		return fmt.Errorf("create temporary feature timing directory: %w", err)
	}
	defer os.RemoveAll(dir)
	startPath, endPath := filepath.Join(dir, "start"), filepath.Join(dir, "end")
	script := unixManagedFeatureScript(shellName, shellRoot, timing.Path, startPath, endPath)

	cmd := exec.Command(executable, append(shellArgs, script)...)
	// Suppress the inherited startup entry points used by bash and POSIX sh.
	cmd.Env = append(managedShellProfileEnv(), "BASH_ENV=", "ENV=")
	// Keep the streams nil so feature output cannot interfere with the timing
	// and descendants cannot hold Go pipes open.
	runErr := runShellProfileCommand(ctx, cmd)
	if runErr != nil {
		runErr = fmt.Errorf("run %s feature: %w", shellName, runErr)
	}

	start, startErr := os.Stat(startPath)
	end, endErr := os.Stat(endPath)
	if startErr == nil && endErr == nil {
		// The shell creates these files with built-ins immediately around the
		// source operation. The file system's timestamp resolution sets the
		// precision, so very fast features can measure as zero.
		timing.DurationMS = float64(max(0, end.ModTime().Sub(start.ModTime()))) / float64(time.Millisecond)
		return runErr
	}
	if runErr != nil {
		return runErr
	}
	if startErr != nil && !errors.Is(startErr, fs.ErrNotExist) {
		return fmt.Errorf("read feature start marker: %w", startErr)
	}
	if endErr != nil && !errors.Is(endErr, fs.ErrNotExist) {
		return fmt.Errorf("read feature end marker: %w", endErr)
	}
	return fmt.Errorf("%s exited before the feature timing finished", shellName)
}

func unixManagedFeatureScript(shellName, shellRoot, featurePath, startPath, endPath string) string {
	if shellName == "fish" {
		return fmt.Sprintf(`set -g OMD_SHELL_ROOT %s
builtin printf '' > %s; or exit $status
source %s
builtin set -l __omd_profile_status $status
builtin printf '' > %s; or exit $status
exit $__omd_profile_status
`, fishProfileLiteral(shellRoot), fishProfileLiteral(startPath), fishProfileLiteral(featurePath), fishProfileLiteral(endPath))
	}

	source := ". " + shellSingleQuote(filepath.ToSlash(featurePath))
	prepare := ""
	if shellName == "bash" || shellName == "zsh" {
		// Their generated loaders source features inside functions, where
		// feature-level local declarations are valid.
		prepare = "__omd_profile_source() {\n  " + source + "\n}\n"
		source = "__omd_profile_source"
	}
	return fmt.Sprintf(`OMD_SHELL_ROOT=%s
if [ -r "$OMD_SHELL_ROOT/../lib/helpers.sh" ]; then
  . "$OMD_SHELL_ROOT/../lib/helpers.sh"
fi
%s: > %s || exit $?
%s
__omd_profile_status=$?
: > %s || exit $?
exit "$__omd_profile_status"
`, shellSingleQuote(filepath.ToSlash(shellRoot)), prepare, shellSingleQuote(filepath.ToSlash(startPath)),
		source, shellSingleQuote(filepath.ToSlash(endPath)))
}

func fishProfileLiteral(path string) string {
	return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(filepath.ToSlash(path)) + "'"
}
