package cmd

import (
	"os"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const pushCompactConfigKey = "push-compact"

var confirmPushCompaction = func() (bool, error) {
	return interactive.Confirm("Compact repeated unpublished commits before pushing?", true)
}

func init() {
	rootCmd.AddCommand(pushCommand)
}

var pushCommand = &cobra.Command{
	Aliases:          []string{"p"},
	Use:              "push",
	Short:            "Push changes to the remote repository",
	Long:             `Push changes to the remote repository.`,
	TraverseChildren: true,
	GroupID:          "dotfiles",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Check write permissions on the repository
		if err := git.CheckRepoWritePermission(); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error: %s", err)
			os.Exit(1)
		}

		WarnIfRemoteUpdatesSync(cmd)

		// Check remote push permissions (exit on error since push requires access)
		git.CheckRemoteAccessWithHelp(true)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if state, err := git.GetRemoteSyncState(); err == nil && state == git.RemoteSyncLocalAhead {
			fileops.ColorPrintln("Detected local committed changes. Pushing...", fileops.Cyan)
		}

		if err := compactPushHistoryIfEnabled(cmd); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error compacting push history: %s", err)
			os.Exit(1)
		}

		pushed, err := git.PushRepo()
		if err != nil {
			if git.IsSSHAgentError(err) {
				git.DisplaySSHAgentError(true)
			} else {
				fileops.ColorPrintfn(fileops.Red, "Error pushing changes: %s", err)
				os.Exit(1)
			}
			return
		}

		if pushed {
			fileops.ColorPrintfn(fileops.Green, "Pushed changes to repository")
		} else {
			fileops.ColorPrintln("Repository already up to date", fileops.Green)
		}
	},
}

func compactPushHistoryIfEnabled(cmd *cobra.Command) error {
	mode := pushCompactMode()
	if mode == "off" {
		return nil
	}
	if mode != "auto" && mode != "ask" {
		fileops.ColorPrintfn(fileops.Yellow, "Warning: unknown push-compact mode %q; skipping compaction", mode)
		return nil
	}

	plan, err := git.PlanPushCompaction()
	if err != nil {
		return err
	}
	if plan.SkippedReason != "" {
		fileops.ColorPrintfn(fileops.Yellow, "Skipping commit compaction: %s", plan.SkippedReason)
		return nil
	}
	if !plan.HasCompaction() {
		return nil
	}

	printPushCompactionPlan(plan)

	if mode == "ask" {
		if interactive.GetMode(cmd) == interactive.ModeNonInteractive {
			fileops.ColorPrintln("Skipping commit compaction: push-compact is ask and prompts are disabled", fileops.Yellow)
			return nil
		}

		confirmed, err := confirmPushCompaction()
		if err != nil {
			return err
		}
		if !confirmed {
			fileops.ColorPrintln("Skipping commit compaction", fileops.Yellow)
			return nil
		}
	}

	_, err = git.CompactPushHistory(plan)
	if err != nil {
		return err
	}

	fileops.ColorPrintfn(fileops.Green, "Compacted %d unpublished commits into %d commits", plan.OriginalCommitCount, plan.CompactedCommitCount)
	return nil
}

func pushCompactMode() string {
	mode := strings.TrimSpace(strings.ToLower(viper.GetString(pushCompactConfigKey)))
	if mode == "" {
		return "auto"
	}
	return mode
}

func printPushCompactionPlan(plan git.PushCompactionPlan) {
	fileops.ColorPrintfn(fileops.Cyan, "Compacting repeated unpublished commits:")
	for _, group := range plan.Groups {
		fileops.ColorPrintfn(fileops.Cyan, "  %dx %s", group.Count, group.Subject)
	}
}
