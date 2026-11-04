package doctor

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

var lookupFeatureExecutable = exec.LookPath

func checkDirectoryStructure(ctx context) []result {
	var results []result

	shellDir := filepath.Join(ctx.repoPath, "omd-shells", ctx.shellName)

	if _, err := os.Stat(shellDir); os.IsNotExist(err) {
		return addResult(results, ctx, errorResult("Shell directory", fmt.Sprintf("Directory missing: %s", shellDir), false), nil)
	}

	results = addResult(results, ctx, okResult("Shell directory"), nil)

	featuresDir := filepath.Join(shellDir, "features")
	if _, err := os.Stat(featuresDir); os.IsNotExist(err) {
		results = addResult(
			results,
			ctx,
			errorResult("Features directory", fmt.Sprintf("Directory missing: %s", featuresDir), true),
			func() (string, error) {
				if err := os.MkdirAll(featuresDir, 0755); err != nil {
					return "", fmt.Errorf("create features directory: %w", err)
				}

				return fmt.Sprintf("Created %s", featuresDir), nil
			},
		)
	} else {
		results = addResult(results, ctx, okResult("Features directory"), nil)
	}

	libDir := filepath.Join(ctx.repoPath, "omd-shells", "lib")
	helpersFile := filepath.Join(libDir, "helpers.sh")

	if _, err := os.Stat(libDir); os.IsNotExist(err) {
		results = addResult(
			results,
			ctx,
			warningResult("Shared lib directory", fmt.Sprintf("Directory missing: %s (optional but recommended)", libDir), true),
			func() (string, error) {
				if err := os.MkdirAll(libDir, 0755); err != nil {
					return "", fmt.Errorf("create shared lib directory: %w", err)
				}

				return fmt.Sprintf("Created %s", libDir), nil
			},
		)
	} else {
		results = addResult(results, ctx, okResult("Shared lib directory"), nil)
	}

	if _, err := os.Stat(helpersFile); os.IsNotExist(err) {
		results = addResult(
			results,
			ctx,
			warningResult("Helpers file", fmt.Sprintf("File missing: %s (optional but recommended)", helpersFile), true),
			func() (string, error) {
				if err := os.MkdirAll(libDir, 0755); err != nil {
					return "", fmt.Errorf("create shared lib directory: %w", err)
				}

				if err := fileops.WriteTextFileLF(helpersFile, shell.HelpersFileContent, 0644); err != nil {
					return "", fmt.Errorf("create helpers file: %w", err)
				}

				return fmt.Sprintf("Created %s", helpersFile), nil
			},
		)
	} else {
		results = addResult(results, ctx, okResult("Helpers file"), nil)
	}

	return results
}

func checkManifest(ctx context) []result {
	var results []result

	manifestPath := shell.GetManifestPath(ctx.repoPath, ctx.shellName)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return addResult(results, ctx, errorResult("Manifest file", fmt.Sprintf("File missing: %s", manifestPath), false), nil)
	}

	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return addResult(results, ctx, errorResult("Manifest validity", fmt.Sprintf("Invalid manifest: %v", err), false), nil)
	}

	results = addResult(results, ctx, okResult("Manifest file"), nil)

	for _, feature := range m.Features {
		if err := feature.Validate(); err != nil {
			results = addResult(
				results,
				ctx,
				errorResult(fmt.Sprintf("Feature '%s' config", feature.Name), err.Error(), false),
				nil,
			)
		}
	}

	if len(m.Features) > 0 {
		results = addResult(results, ctx, okResult(fmt.Sprintf("Feature configs (%d)", len(m.Features))), nil)
	}

	return results
}

func checkFeatureFiles(ctx context) []result {
	var results []result
	merged, err := manifest.ParseManifestWithLocal(shell.GetManifestPath(ctx.repoPath, ctx.shellName), shell.GetLocalManifestPath(ctx.repoPath, ctx.shellName))
	if err != nil {
		return results
	}
	missingCount := 0
	for _, feature := range merged.Features {
		if feature.Name == "oh-my-posh" && !feature.Disabled {
			if _, err := lookupFeatureExecutable("oh-my-posh"); err != nil {
				results = addResult(results, ctx, warningResult("Oh My Posh executable", "The feature is configured, but oh-my-posh is not available on PATH. Install Oh My Posh using your package manager, then open a new shell. Configuring a feature does not install its prerequisite.", false), nil)
			}
		}
		featurePath, err := shell.GetFeatureFilePath(ctx.repoPath, ctx.shellName, feature.Name)
		if err != nil {
			continue
		}
		info, statErr := os.Lstat(featurePath)
		if statErr == nil {
			if !info.Mode().IsRegular() {
				results = addResult(results, ctx, errorResult(fmt.Sprintf("Feature file '%s'", feature.Name), fmt.Sprintf("Feature must be a regular file: %s", featurePath), false), nil)
				missingCount++
			}
			continue
		}
		if !os.IsNotExist(statErr) {
			results = addResult(results, ctx, errorResult(fmt.Sprintf("Feature file '%s'", feature.Name), fmt.Sprintf("Cannot inspect %s: %v", featurePath, statErr), false), nil)
			missingCount++
			continue
		}
		missingCount++
		featureName := feature.Name
		if !catalog.HasFeatureTemplate(featureName, ctx.shellName) {
			results = addResult(results, ctx, errorResult(fmt.Sprintf("Feature file '%s'", featureName), fmt.Sprintf("File missing: %s. Restore your custom implementation from Git history or a backup; no catalog template is available.", featurePath), false), nil)
			continue
		}
		featureOptions := feature.Options
		results = addResult(results, ctx, errorResult(fmt.Sprintf("Feature file '%s'", featureName), fmt.Sprintf("File missing: %s. Restore the catalog version with saved options; custom edits require Git history or a backup.", featurePath), true), func() (string, error) {
			source, err := catalog.GetFeatureTemplate(featureName, ctx.shellName)
			if err != nil {
				return "", fmt.Errorf("read feature template: %w", err)
			}
			content, err := catalog.RenderFeatureTemplate(source, featureName, ctx.shellName, featureOptions)
			if err != nil {
				return "", fmt.Errorf("render feature template: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(featurePath), 0755); err != nil {
				return "", fmt.Errorf("create feature directory: %w", err)
			}
			file, err := os.OpenFile(featurePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
			if err != nil {
				return "", fmt.Errorf("restore missing feature without replacing existing content: %w", err)
			}
			if _, err := file.WriteString(content); err != nil {
				file.Close()
				return "", fmt.Errorf("write feature template: %w", err)
			}
			if err := file.Close(); err != nil {
				return "", fmt.Errorf("close restored feature: %w", err)
			}
			return fmt.Sprintf("Restored catalog template %s with saved options. This restores the catalog version; customized content must come from Git history or a backup.", featurePath), nil
		})
	}
	if missingCount == 0 && len(merged.Features) > 0 {
		results = addResult(results, ctx, okResult(fmt.Sprintf("Feature files (%d)", len(merged.Features))), nil)
	}
	return results
}

func checkLineEndings(ctx context) []result {
	var results []result

	files, err := shellFrameworkFiles(ctx.repoPath, ctx.shellName)
	if err != nil {
		return addResult(results, ctx, warningResult("Line endings", fmt.Sprintf("Cannot scan files: %v", err), false), nil)
	}

	var crlfFiles []string
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return addResult(results, ctx, errorResult("Line endings", fmt.Sprintf("Cannot read %s: %v", path, err), false), nil)
		}

		if bytes.Contains(data, []byte("\r")) {
			crlfFiles = append(crlfFiles, path)
		}
	}

	if len(crlfFiles) == 0 {
		return addResult(results, ctx, okResult("Line endings"), nil)
	}

	return addResult(
		results,
		ctx,
		errorResult("Line endings", fmt.Sprintf("CRLF detected in %d file(s)", len(crlfFiles)), true),
		func() (string, error) {
			for _, path := range crlfFiles {
				data, err := os.ReadFile(path)
				if err != nil {
					return "", fmt.Errorf("read %s: %w", path, err)
				}

				if err := fileops.WriteTextFileLF(path, string(data), 0644); err != nil {
					return "", fmt.Errorf("rewrite %s: %w", path, err)
				}
			}

			return fmt.Sprintf("Rewrote %d file(s) with LF endings", len(crlfFiles)), nil
		},
	)
}

func shellFrameworkFiles(repoPath, shellName string) ([]string, error) {
	var files []string

	roots := []string{
		filepath.Join(repoPath, "omd-shells", "lib"),
		filepath.Join(repoPath, "omd-shells", shellName),
	}

	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", root, err)
		}

		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			files = append(files, path)
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk %s: %w", root, err)
		}
	}

	return files, nil
}
