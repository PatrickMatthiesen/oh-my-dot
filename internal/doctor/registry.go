package doctor

import "github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"

func registeredChecks() []check {
	return []check{
		{run: checkDirectoryStructure},
		{run: checkManifest},
		{run: checkLocalOverride},
		{run: checkFeatureFiles},
		{run: checkLineEndings},
		{run: checkProfileHooks},
		{run: checkInitScriptSyntax},
	}
}

func runChecks(repoPath string, shellsToCheck []string, fix bool) []result {
	var allResults []result

	for _, shellName := range shellsToCheck {
		fileops.ColorPrintfn(fileops.Cyan, "\nChecking %s shell...", shellName)

		ctx := context{
			repoPath:  repoPath,
			shellName: shellName,
			fix:       fix,
		}

		for _, check := range registeredChecks() {
			allResults = append(allResults, check.run(ctx)...)
		}
	}

	return allResults
}
