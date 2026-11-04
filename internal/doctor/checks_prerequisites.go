package doctor

import (
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func checkModulePrerequisites(ctx context) []result {
	merged, err := manifest.ParseManifestWithLocal(shell.GetManifestPath(ctx.repoPath, ctx.shellName), shell.GetLocalManifestPath(ctx.repoPath, ctx.shellName))
	if err != nil {
		return nil
	} // The manifest diagnostic owns this error.
	var features []manifest.FeatureConfig
	for _, feature := range merged.Features {
		if feature.Name == "posh-git" || feature.Name == "terminal-icons" {
			features = append(features, feature.FeatureConfig)
		}
	}
	warnings, err := shell.CheckRequirementsForFeatures(ctx.shellName, features)
	var results []result
	if err != nil {
		return addResult(results, ctx, warningResult("Feature prerequisites", err.Error(), false), nil)
	}
	for _, warning := range warnings {
		results = addResult(results, ctx, warningResult("Feature prerequisite", warning, false), nil)
	}
	return results
}
