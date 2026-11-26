package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

const slowProfileEntryMS = 100

const (
	profileModeManaged = "managed"
	profileModeFull    = "full"
)

type profileDocument struct {
	SchemaVersion int             `json:"schemaVersion"`
	Results       []profileResult `json:"results"`
}

type profileResult struct {
	Shell  string                    `json:"shell"`
	Mode   string                    `json:"mode"`
	Status string                    `json:"status"`
	Report *shell.ShellProfileReport `json:"report,omitempty"`
	Error  string                    `json:"error,omitempty"`
}

func init() {
	rootCmd.AddCommand(newProfileCommand(
		shell.IsShellExecutableAvailable,
		func() string { return viper.GetString("repo-path") },
		func(ctx context.Context, name, path string) (shell.ShellProfileReport, error) {
			if name == "powershell" {
				return shell.ProfilePowerShellFeatures(ctx, path)
			}
			return shell.ProfileUnixShellFeatures(ctx, name, path)
		},
		func(ctx context.Context, name string) (shell.ShellProfileReport, error) {
			if name == "powershell" {
				return shell.ProfilePowerShell(ctx)
			}
			return shell.ProfileUnixShellStartup(ctx, name)
		},
	))
}

func newProfileCommand(
	available func(string) bool,
	repoPath func() string,
	profileManaged func(context.Context, string, string) (shell.ShellProfileReport, error),
	profileFull func(context.Context, string) (shell.ShellProfileReport, error),
) *cobra.Command {
	supported := []string{"bash", "zsh", "fish", "powershell", "posix"}
	var shellName string
	var asJSON bool
	var full bool
	command := &cobra.Command{
		Use:          "profile",
		Short:        "Measure managed shell feature load time",
		Long:         "Measure each enabled oh-my-dot managed feature in a separate process. By default, profile all supported installed shells. Use --full to measure shell startup.",
		Example:      "  oh-my-dot profile\n  oh-my-dot profile --shell powershell --json\n  oh-my-dot profile --full",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode := profileModeManaged
			if full {
				mode = profileModeFull
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			document := profileDocument{SchemaVersion: 1, Results: []profileResult{}}
			var failures []error
			names := supported
			explicit := cmd.Flags().Changed("shell")
			if explicit {
				name := strings.ToLower(shellName)
				if name == "sh" {
					name = "posix"
				}
				if slices.Contains(supported, name) {
					names = []string{name}
				} else {
					names = nil
					err := fmt.Errorf("startup profiling is unavailable for shell %q. Supported shells are %s", shellName, strings.Join(supported, ", "))
					document.Results = append(document.Results, profileResult{Shell: shellName, Mode: mode, Status: "unsupported", Error: err.Error()})
					failures = append(failures, err)
				}
			}
			readRepoPath := sync.OnceValue(repoPath)
			for _, name := range names {
				if ctx.Err() != nil {
					break
				}
				if !available(name) {
					if explicit {
						label := profileShellLabel(name)
						err := fmt.Errorf("%s executable not found. Install %s or add it to PATH", label, label)
						document.Results = append(document.Results, profileResult{Shell: name, Mode: mode, Status: "unavailable", Error: err.Error()})
						failures = append(failures, err)
					}
					continue
				}
				var report shell.ShellProfileReport
				var err error
				if full {
					report, err = profileFull(ctx, name)
				} else {
					report, err = profileManaged(ctx, name, readRepoPath())
				}
				result, err := newProfileResult(name, mode, report, err)
				document.Results = append(document.Results, result)
				if err != nil {
					failures = append(failures, err)
				}
			}
			profileErr := errors.Join(failures...)
			if err := ctx.Err(); err != nil {
				if !errors.Is(profileErr, err) {
					profileErr = errors.Join(profileErr, fmt.Errorf("shell profiling interrupted: %w", err))
				}
			} else if len(document.Results) == 0 {
				profileErr = fmt.Errorf("no supported shell executables found in PATH")
			}

			out := cmd.OutOrStdout()
			if asJSON {
				if err := json.NewEncoder(out).Encode(document); err != nil {
					return fmt.Errorf("write profile JSON: %w", err)
				}
			} else {
				for _, result := range document.Results {
					if result.Report != nil {
						if err := writeProfileText(out, result); err != nil {
							return fmt.Errorf("write profile report: %w", err)
						}
					}
				}
			}
			return profileErr
		},
	}
	command.Flags().StringVar(&shellName, "shell", "", "Profile one shell ("+strings.Join(supported, ", ")+")")
	command.Flags().BoolVar(&asJSON, "json", false, "Print a versioned JSON report")
	command.Flags().BoolVar(&full, "full", false, "Measure shell startup")
	return command
}

func newProfileResult(name, mode string, report shell.ShellProfileReport, err error) (profileResult, error) {
	if mode == profileModeManaged && err == nil {
		for _, feature := range report.ManagedFeatures {
			if feature.Error != "" {
				err = fmt.Errorf("one or more managed features could not be profiled")
				break
			}
		}
	}
	result := profileResult{Shell: name, Mode: mode, Status: "ok"}
	if err == nil {
		result.Report = &report
		return result, nil
	}
	err = fmt.Errorf("profile %s: %w", profileShellLabel(name), err)
	result.Status = "failed"
	result.Error = err.Error()
	// The runtime can return timings collected before a failure.
	if report.TotalMS != 0 || len(report.Profiles) != 0 || len(report.ManagedFeatures) != 0 || report.Version != "" {
		result.Report = &report
		result.Status = "partial"
	}
	return result, err
}

func profileShellLabel(name string) string {
	switch name {
	case "powershell":
		return "PowerShell"
	case "posix":
		return "POSIX sh"
	default:
		return name
	}
}

func writeProfileText(out io.Writer, result profileResult) error {
	report := result.Report
	var text strings.Builder
	label := profileShellLabel(result.Shell)
	if result.Shell == "powershell" && (result.Mode == profileModeFull || report.Version != "") {
		fmt.Fprintf(&text, "PowerShell %s (%s, %s)\n", report.Version, report.Edition, report.Host)
	} else if result.Mode == profileModeManaged {
		fmt.Fprintf(&text, "%s managed feature profiling\n", label)
	} else {
		fmt.Fprintf(&text, "%s startup profiling\n", label)
	}
	if result.Mode == profileModeFull || report.Executable != "" {
		fmt.Fprintf(&text, "Executable: %s\n", report.Executable)
	}
	if result.Status == "partial" {
		fmt.Fprintln(&text, "Partial result. Timings cover the collected samples.")
	}
	if result.Mode == profileModeManaged {
		writeManagedProfileTimings(&text, report, label)
	} else if result.Shell == "powershell" {
		writeFullProfileTimings(&text, report)
	} else {
		fmt.Fprintf(&text, "Total startup time (including process launch): %.2f ms\n", report.TotalMS)
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(&text, "Warning: %s\n", warning)
	}
	_, err := io.WriteString(out, text.String())
	return err
}

func writeManagedProfileTimings(text *strings.Builder, report *shell.ShellProfileReport, label string) {
	fmt.Fprintf(text, "Total managed feature time: %.2f ms\n", report.TotalMS)
	if len(report.ManagedFeatures) == 0 {
		fmt.Fprintf(text, "No enabled managed %s features to profile.\n", label)
	}
	for _, feature := range report.ManagedFeatures {
		writeProfileFeatureTiming(text, feature.FeatureTiming, "  ")
		if feature.Error != "" {
			fmt.Fprintf(text, "    Error: %s\n", feature.Error)
		} else {
			writeProfileFeatureAdvice(text, feature.FeatureTiming, "    ", false)
		}
	}
}

func writeFullProfileTimings(text *strings.Builder, report *shell.ShellProfileReport) {
	fmt.Fprintf(text, "Total profile time: %.2f ms\n", report.TotalMS)
	if len(report.Profiles) == 0 {
		fmt.Fprintln(text, "No profile file timings were recorded.")
	}
	for _, profile := range report.Profiles {
		fmt.Fprintf(text, "  %.2f ms  %s\n", profile.DurationMS, profile.Path)
		// Features arrive slowest first and their timings may overlap.
		if profile.DurationMS >= slowProfileEntryMS && (len(profile.Features) == 0 || profile.Features[0].DurationMS < slowProfileEntryMS) {
			fmt.Fprintln(text, "    Inspect imports and commands in this profile.")
		}
		for _, feature := range profile.Features {
			writeProfileFeatureTiming(text, feature, "    ")
			writeProfileFeatureAdvice(text, feature, "      ", true)
		}
	}
	// The runtime normally explains missing attribution in its warnings.
	if !report.AttributionAvailable && len(report.Warnings) == 0 {
		fmt.Fprintln(text, "Managed feature timings are unavailable for this run.")
	}
}

func writeProfileFeatureTiming(text *strings.Builder, feature shell.FeatureTiming, indent string) {
	fmt.Fprintf(text, "%s%.2f ms  %s (%s)  %s\n", indent, feature.DurationMS, feature.Name, feature.Strategy, feature.Path)
}

func writeProfileFeatureAdvice(text *strings.Builder, feature shell.FeatureTiming, indent string, full bool) {
	if feature.DurationMS < slowProfileEntryMS {
		return
	}
	if strings.EqualFold(feature.Name, "oh-my-posh") {
		fmt.Fprintln(text, indent+"Inspect the Oh My Posh theme and cache configuration. Check whether auto_upgrade runs during startup.")
		return
	}
	fmt.Fprintln(text, indent+"Inspect imports and external commands in this feature.")
	if full && feature.Strategy == "eager" {
		fmt.Fprintln(text, indent+"Consider defer if this feature can wait until the shell is idle.")
	}
	fmt.Fprintln(text, indent+"If you do not use this feature, consider disabling it.")
}
