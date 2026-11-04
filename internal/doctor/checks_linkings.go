package doctor

import (
	"fmt"
	"os"
	"sort"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/repopath"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

func checkLinkings(ctx context) []result {
	var results []result
	links, err := symlink.LoadLinkings(ctx.repoPath)
	if err != nil {
		return addResult(results, ctx, errorResult("Dotfile registrations", err.Error(), false), nil)
	}
	if err := symlink.ValidateLinkings(ctx.repoPath, links); err != nil {
		return addResult(results, ctx, errorResult("Dotfile registrations", err.Error(), false), nil)
	}
	results = addResult(results, ctx, okResult("Dotfile registrations"), nil)
	keys := make([]string, 0, len(links))
	for key := range links {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		source, err := repopath.Resolve(ctx.repoPath, key)
		if err != nil {
			results = addResult(results, ctx, errorResult("Dotfile "+key, err.Error(), false), nil)
			continue
		}
		sourceInfo, err := os.Stat(source)
		if err != nil || !sourceInfo.Mode().IsRegular() {
			results = addResult(results, ctx, errorResult("Dotfile "+key, "Stored source missing or not a regular file; restore it from Git or a backup.", false), nil)
			continue
		}
		target, err := fileops.ExpandPath(links[key])
		if err != nil {
			results = addResult(results, ctx, errorResult("Dotfile "+key, err.Error(), false), nil)
			continue
		}
		targetInfo, err := os.Stat(target)
		if os.IsNotExist(err) {
			if _, linkErr := os.Lstat(target); linkErr == nil {
				results = addResult(results, ctx, errorResult("Dotfile "+key, fmt.Sprintf("Broken link at %s; inspect it before apply.", target), false), nil)
			} else {
				results = addResult(results, ctx, warningResult("Dotfile "+key, "Not applied; run apply --dry-run, then apply.", false), nil)
			}
			continue
		}
		if err != nil {
			results = addResult(results, ctx, errorResult("Dotfile "+key, err.Error(), false), nil)
			continue
		}
		if !os.SameFile(sourceInfo, targetInfo) {
			results = addResult(results, ctx, warningResult("Dotfile "+key, fmt.Sprintf("Local file %s differs from the managed source. Inspect both; use add --update to capture intentional local edits, or move a conflict aside before apply.", target), false), nil)
			continue
		}
		results = addResult(results, ctx, okResult("Dotfile "+key), nil)
	}
	return results
}
