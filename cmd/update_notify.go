package cmd

import (
	"sync"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// updateCheckTTL is the time-to-live ("Time To Live") for cached update-check results.
	updateCheckTTL               = 20 * time.Minute
	updateCheckResultWaitTimeout = 150 * time.Millisecond
	updateLastCheckedKey         = "update-check.last-checked-unix"
	updateHasUpdatesKey          = "update-check.remote-has-updates"
	updateStateKey               = "update-check.remote-sync-state"
)

type updateCheckResult struct {
	state     git.RemoteSyncState
	err       error
	checkedAt int64
}

type updateCheckRuntimeState struct {
	pendingChan     chan updateCheckResult
	immediateNotice bool
}

var (
	updateCheckStateMu sync.Mutex
	updateCheckState   updateCheckRuntimeState
)

func topLevelCommandName(cmd *cobra.Command) string {
	top := cmd
	for top.Parent() != nil && top.Parent() != cmd.Root() {
		top = top.Parent()
	}
	return top.Name()
}

func shouldCheckUpdatesForCommand(cmd *cobra.Command) bool {
	switch topLevelCommandName(cmd) {
	case "add", "remove", "apply", "push", "doctor":
		return true
	default:
		return false
	}
}

func shouldUseAsyncUpdateCheck(cmd *cobra.Command) bool {
	return shouldCheckUpdatesForCommand(cmd) && topLevelCommandName(cmd) != "push"
}

func printUpdateAvailableNotice(cmd *cobra.Command, state git.RemoteSyncState) {
	alias := cmd.Root().Name()

	switch state {
	case git.RemoteSyncRemoteAhead:
		fileops.ColorPrintfn(fileops.Yellow, "Remote updates are available. Run '%s pull' to update your local dotfiles repository.", alias)
	case git.RemoteSyncRemoteSignificantlyAhead:
		fileops.ColorPrintfn(fileops.Yellow, "Local repository is 100+ commits behind remote. Run '%s pull' to update your local dotfiles repository.", alias)
	case git.RemoteSyncDiverged:
		fileops.ColorPrintfn(fileops.Yellow, "Local and remote history diverged. Run '%s pull' and resolve conflicts if prompted.", alias)
	case git.RemoteSyncLocalAhead:
		fileops.ColorPrintfn(fileops.Cyan, "Local repository is ahead of remote. You can run '%s push' to publish local commits.", alias)
	}
}

func stateHasRemoteUpdates(state git.RemoteSyncState) bool {
	return state == git.RemoteSyncRemoteAhead || state == git.RemoteSyncRemoteSignificantlyAhead || state == git.RemoteSyncDiverged
}

func cacheUpdateCheckResult(res updateCheckResult) {
	viper.Set(updateLastCheckedKey, res.checkedAt)
	viper.Set(updateStateKey, string(res.state))
	viper.Set(updateHasUpdatesKey, stateHasRemoteUpdates(res.state))
	_ = viper.WriteConfig()
}

func StartAsyncUpdateCheck(cmd *cobra.Command) {
	if !shouldUseAsyncUpdateCheck(cmd) {
		return
	}

	repoPath := viper.GetString("repo-path")
	if repoPath == "" || !git.IsGitRepo(repoPath) {
		return
	}

	now := time.Now().Unix()
	lastChecked := viper.GetInt64(updateLastCheckedKey)
	if lastChecked > 0 {
		lastCheckTime := time.Unix(lastChecked, 0)
		if time.Since(lastCheckTime) < updateCheckTTL {
			state := git.RemoteSyncState(viper.GetString(updateStateKey))
			if state == "" && viper.GetBool(updateHasUpdatesKey) {
				state = git.RemoteSyncRemoteAhead
			}

			if stateHasRemoteUpdates(state) {
				updateCheckStateMu.Lock()
				updateCheckState.immediateNotice = true
				updateCheckStateMu.Unlock()
			}
			return
		}
	}

	ch := make(chan updateCheckResult, 1)

	updateCheckStateMu.Lock()
	updateCheckState.pendingChan = ch
	updateCheckState.immediateNotice = false
	updateCheckStateMu.Unlock()

	go func() {
		state, err := git.GetRemoteSyncState()
		ch <- updateCheckResult{
			state:     state,
			err:       err,
			checkedAt: now,
		}
	}()
}

func FinishAsyncUpdateCheck(cmd *cobra.Command) {
	if !shouldUseAsyncUpdateCheck(cmd) {
		return
	}

	updateCheckStateMu.Lock()
	state := updateCheckState
	updateCheckState = updateCheckRuntimeState{}
	updateCheckStateMu.Unlock()

	if state.immediateNotice {
		cachedState := git.RemoteSyncState(viper.GetString(updateStateKey))
		if cachedState == "" && viper.GetBool(updateHasUpdatesKey) {
			cachedState = git.RemoteSyncRemoteAhead
		}
		if stateHasRemoteUpdates(cachedState) {
			printUpdateAvailableNotice(cmd, cachedState)
		}
		return
	}

	if state.pendingChan == nil {
		return
	}

	select {
	case res := <-state.pendingChan:
		if res.err != nil {
			return
		}
		cacheUpdateCheckResult(res)
		if stateHasRemoteUpdates(res.state) {
			printUpdateAvailableNotice(cmd, res.state)
		}
	case <-time.After(updateCheckResultWaitTimeout):
		return
	}
}

func WarnIfRemoteUpdatesSync(cmd *cobra.Command) {
	if !shouldCheckUpdatesForCommand(cmd) || topLevelCommandName(cmd) != "push" {
		return
	}

	repoPath := viper.GetString("repo-path")
	if repoPath == "" || !git.IsGitRepo(repoPath) {
		return
	}

	state, err := git.GetRemoteSyncState()
	if err != nil {
		return
	}

	cacheUpdateCheckResult(updateCheckResult{
		state:     state,
		err:       nil,
		checkedAt: time.Now().Unix(),
	})

	if stateHasRemoteUpdates(state) || state == git.RemoteSyncLocalAhead {
		printUpdateAvailableNotice(cmd, state)
	}
}
