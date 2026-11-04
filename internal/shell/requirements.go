package shell

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

var lookupRequirementCommand = exec.LookPath
var listPowerShellModules = func() (map[string]bool, error) {
	executable := ""
	for _, candidate := range powershellExecutableCandidates() {
		if path, err := lookupRequirementCommand(candidate); err == nil {
			executable = path
			break
		}
	}
	if executable == "" {
		return nil, fmt.Errorf("PowerShell executable is not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Fixed module names only; this inspects availability without importing modules
	// or loading profiles. No install command runs during inspection or activation.
	output, err := exec.CommandContext(ctx, executable, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "Get-Module -ListAvailable -Name posh-git,Terminal-Icons | Select-Object -ExpandProperty Name -Unique").Output()
	if err != nil {
		return nil, fmt.Errorf("inspect PowerShell modules: %w", err)
	}
	available := map[string]bool{}
	for _, name := range strings.Fields(string(output)) {
		available[strings.ToLower(name)] = true
	}
	return available, nil
}

// FeatureRequirementWarnings reports missing prerequisites for enabled features.
// It never installs dependencies, sources profiles or executes feature scripts.
func FeatureRequirementWarnings(repoPath, shellName string) ([]string, error) {
	merged, err := manifest.ParseManifestWithLocal(GetManifestPath(repoPath, shellName), GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return nil, fmt.Errorf("read %s prerequisites: %w", shellName, err)
	}
	features := make([]manifest.FeatureConfig, 0, len(merged.Features))
	for _, feature := range merged.Features {
		features = append(features, feature.FeatureConfig)
	}
	return CheckRequirementsForFeatures(shellName, features)
}

// CheckRequirementsForFeatures checks selected runtime prerequisites without
// importing modules, loading features, or installing dependencies.
func CheckRequirementsForFeatures(shellName string, features []manifest.FeatureConfig) ([]string, error) {
	var warnings []string
	var modules map[string]bool
	var moduleErr error
	checkedModules := false
	for _, feature := range features {
		if feature.Disabled {
			continue
		}
		for _, requirement := range catalog.Requirements(feature.Name, shellName) {
			if requirement.Kind == "command" {
				if _, err := lookupRequirementCommand(requirement.Name); err != nil {
					warnings = append(warnings, fmt.Sprintf("%s is configured but %s is not installed. %s", feature.Name, requirement.Name, requirement.Instruction))
				}
			} else {
				if !checkedModules {
					modules, moduleErr = listPowerShellModules()
					checkedModules = true
				}
				if moduleErr != nil {
					warnings = append(warnings, fmt.Sprintf("Cannot verify %s prerequisite: %v. %s", feature.Name, moduleErr, requirement.Instruction))
				} else if !modules[strings.ToLower(requirement.Name)] {
					warnings = append(warnings, fmt.Sprintf("%s is configured but module %s is not installed. %s", feature.Name, requirement.Name, requirement.Instruction))
				}
			}
		}
	}
	sort.Strings(warnings)
	return warnings, nil
}
