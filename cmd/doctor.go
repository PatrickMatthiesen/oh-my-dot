package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/doctor"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Aliases: []string{"status"},
	Short:   "Check repository and shell framework health",
	Long: `Diagnose and validate repository and shell framework configuration.

Checks performed:
  - Git repository can be opened and commit identity is configured
  - Origin remote is configured and readable (warnings only)
  - Shell hooks are properly installed in profile files
  - Directory structure is correct
  - Manifest files are valid
  - Feature files exist
  - Local override security (permissions, ownership)
  - Init script syntax

Examples:
  oh-my-dot doctor              # Check repository and all shells
  oh-my-dot doctor --shell bash # Check repository and bash`,
	GroupID:      "dotfiles",
	SilenceUsage: true,
	RunE:         runDoctor,
}

var (
	flagDoctorShell []string
	flagFix         bool
)

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().Bool("json", false, "Print versioned JSON diagnostics without prompts")
	doctorCmd.Flags().Bool("offline", false, "Check local health without contacting the remote")
	doctorCmd.Flags().StringSliceVar(&flagDoctorShell, "shell", nil, "Check specific shell(s) only")
	doctorCmd.Flags().BoolVar(&flagFix, "fix", false, "Attempt to fix issues automatically")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	repoPath := viper.GetString("repo-path")
	alias := assumedAlias()

	shellsToCheck, err := doctor.ResolveShellsToCheck(repoPath, flagDoctorShell)
	if err != nil {
		return err
	}

	offline, _ := cmd.Flags().GetBool("offline")
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		report, checkErr := doctor.Inspect(repoPath, shellsToCheck, flagFix, offline)
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(report); err != nil {
			return fmt.Errorf("write doctor JSON: %w", err)
		}
		return checkErr
	}
	return doctor.RunWithOptions(repoPath, shellsToCheck, alias, flagFix, offline)
}
