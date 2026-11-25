package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

const slowProfileEntryMS = 100

type profileDocument struct {
	SchemaVersion int             `json:"schemaVersion"`
	Results       []profileResult `json:"results"`
}

type profileResult struct {
	Shell  string                         `json:"shell"`
	Status string                         `json:"status"`
	Report *shell.PowerShellProfileReport `json:"report,omitempty"`
	Error  string                         `json:"error,omitempty"`
}

func init() {
	rootCmd.AddCommand(newProfileCommand(shell.IsShellExecutableAvailable, shell.ProfilePowerShell))
}

func newProfileCommand(
	available func(string) bool,
	profilePowerShell func(context.Context) (shell.PowerShellProfileReport, error),
) *cobra.Command {
	var shellName string
	var asJSON bool
	command := &cobra.Command{
		Use:          "profile",
		Short:        "Measure shell profile startup time",
		Long:         "Measure profile startup for all installed shells with a profiler. Currently supports PowerShell.",
		Example:      "  oh-my-dot profile\n  oh-my-dot profile --shell powershell --json",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result := profileResult{Shell: "powershell"}
			var profileErr error
			switch {
			case cmd.Flags().Changed("shell") && !strings.EqualFold(shellName, "powershell"):
				result.Shell = shellName
				result.Status = "unsupported"
				profileErr = fmt.Errorf("startup profiling is unavailable for shell %q. The supported shell is powershell", shellName)
			case !available("powershell"):
				result.Status = "unavailable"
				profileErr = fmt.Errorf("PowerShell executable not found. Install PowerShell or add it to PATH")
			default:
				report, err := profilePowerShell(cmd.Context())
				result.Status = "ok"
				if err != nil {
					profileErr = fmt.Errorf("profile PowerShell: %w", err)
					result.Status = "failed"
				}
				// The runtime can return timings collected before a failure.
				if err == nil || len(report.Profiles) != 0 || report.Version != "" {
					result.Report = &report
					if err != nil {
						result.Status = "partial"
					}
				}
			}
			if profileErr != nil {
				result.Error = profileErr.Error()
			}

			out := cmd.OutOrStdout()
			if asJSON {
				document := profileDocument{SchemaVersion: 1, Results: []profileResult{result}}
				if err := json.NewEncoder(out).Encode(document); err != nil {
					return fmt.Errorf("write profile JSON: %w", err)
				}
			} else if result.Report != nil {
				if err := writeProfileText(out, result); err != nil {
					return fmt.Errorf("write profile report: %w", err)
				}
			}
			return profileErr
		},
	}
	command.Flags().StringVar(&shellName, "shell", "", "Profile one shell (currently powershell)")
	command.Flags().BoolVar(&asJSON, "json", false, "Print a versioned JSON report")
	return command
}

func writeProfileText(out io.Writer, result profileResult) error {
	report := result.Report
	var text strings.Builder
	fmt.Fprintf(&text, "PowerShell %s (%s, %s)\n", report.Version, report.Edition, report.Host)
	fmt.Fprintf(&text, "Executable: %s\n", report.Executable)
	if result.Status == "partial" {
		fmt.Fprintln(&text, "Partial result. Timings cover the collected samples.")
	}
	fmt.Fprintf(&text, "Total profile time: %.2f ms\n", report.TotalMS)
	if len(report.Profiles) == 0 {
		fmt.Fprintln(&text, "No profile file timings were recorded.")
	}
	for _, profile := range report.Profiles {
		fmt.Fprintf(&text, "  %.2f ms  %s\n", profile.DurationMS, profile.Path)
		// Features arrive slowest first and their timings may overlap.
		if profile.DurationMS >= slowProfileEntryMS && (len(profile.Features) == 0 || profile.Features[0].DurationMS < slowProfileEntryMS) {
			fmt.Fprintln(&text, "    Inspect imports and commands in this profile.")
		}
		for _, feature := range profile.Features {
			fmt.Fprintf(&text, "    %.2f ms  %s (%s)  %s\n", feature.DurationMS, feature.Name, feature.Strategy, feature.Path)
			if feature.DurationMS >= slowProfileEntryMS {
				if strings.EqualFold(feature.Name, "oh-my-posh") {
					fmt.Fprintln(&text, "      Inspect the Oh My Posh theme and cache configuration. Check whether auto_upgrade runs during startup.")
				} else {
					fmt.Fprintln(&text, "      Inspect imports and external commands in this feature.")
					fmt.Fprintln(&text, "      If you do not use this feature, consider disabling it.")
				}
			}
		}
	}
	// The runtime normally explains missing attribution in its warnings.
	if !report.AttributionAvailable && len(report.Warnings) == 0 {
		fmt.Fprintln(&text, "Managed feature timings are unavailable for this run.")
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(&text, "Warning: %s\n", warning)
	}
	_, err := io.WriteString(out, text.String())
	return err
}
