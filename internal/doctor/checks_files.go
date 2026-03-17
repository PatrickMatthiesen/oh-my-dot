package doctor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

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

				if err := os.WriteFile(helpersFile, []byte(shell.HelpersFileContent), 0644); err != nil {
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

	manifestPath := shell.GetManifestPath(ctx.repoPath, ctx.shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return results
	}

	missingCount := 0
	for _, feature := range m.Features {
		featurePath, err := shell.GetFeatureFilePath(ctx.repoPath, ctx.shellName, feature.Name)
		if err != nil {
			continue
		}

		if _, err := os.Stat(featurePath); os.IsNotExist(err) {
			missingCount++
			featureName := feature.Name
			results = addResult(
				results,
				ctx,
				errorResult(fmt.Sprintf("Feature file '%s'", featureName), fmt.Sprintf("File missing: %s", featurePath), true),
				func() (string, error) {
					content := []byte("# Feature: " + featureName + "\n")
					if err := os.WriteFile(featurePath, content, 0644); err != nil {
						return "", fmt.Errorf("create feature file %q: %w", featureName, err)
					}

					return fmt.Sprintf("Created %s", featurePath), nil
				},
			)
		}
	}

	if missingCount == 0 && len(m.Features) > 0 {
		results = addResult(results, ctx, okResult(fmt.Sprintf("Feature files (%d)", len(m.Features))), nil)
	}

	return results
}
