package doctor

import "github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"

func registeredChecks() []check {
	return []check{
		{run: checkDirectoryStructure},
		{run: checkManifest},
		{run: checkLocalOverride},
		{run: checkFeatureFiles},
		{run: checkModulePrerequisites},
		{run: checkLineEndings},
		{run: checkProfileHooks},
		{run: checkInitScriptSyntax},
	}
}

func runChecks(repoPath string, shellsToCheck []string, fix bool) []result {
	return runChecksWithOptions(repoPath, shellsToCheck, fix, false, false)
}

func runChecksWithOptions(repoPath string, shellsToCheck []string, fix, quiet, offline bool) []result {
	ctx := context{repoPath: repoPath, fix: fix, quiet: quiet, offline: offline}
	allResults := checkRepository(ctx)
	allResults = append(allResults, checkLinkings(ctx)...)

	for _, shellName := range shellsToCheck {
		if !quiet {
			fileops.ColorPrintfn(fileops.Cyan, "\nChecking %s shell...", shellName)
		}

		ctx := context{
			repoPath:  repoPath,
			shellName: shellName,
			fix:       fix,
			quiet:     quiet,
			offline:   offline,
		}

		for _, check := range registeredChecks() {
			allResults = append(allResults, check.run(ctx)...)
		}
	}

	return allResults
}
