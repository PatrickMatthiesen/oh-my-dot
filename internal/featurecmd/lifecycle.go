package featurecmd

import (
	"errors"
	"fmt"
	"sort"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
)

var selectLifecycleFeature = interactive.PromptSelect
var selectLifecycleShells = interactive.MultiSelect

func lifecycleRecordLabel(record featureRecord) string {
	return fmt.Sprintf("%s (%s)", record.Name, record.Shell)
}

func (state *commandState) loadSelection(cmd *cobra.Command, args []string, disabled bool) (string, []string, error) {
	records, err := featureRecords(state.repoPathProvider(), state.flagShell)
	if err != nil {
		return "", nil, err
	}
	if len(args) == 0 {
		if !interactive.ShouldPrompt(cmd, false) {
			return "", nil, fmt.Errorf("feature name required; specify a feature and --shell (or --all), or run in a terminal to select")
		}
		var eligible []featureRecord
		var labels []string
		for _, record := range records {
			if record.Disabled != disabled {
				eligible = append(eligible, record)
				labels = append(labels, lifecycleRecordLabel(record))
			}
		}
		if len(eligible) == 0 {
			return "", nil, nil
		}
		action := "enable"
		if disabled {
			action = "disable"
		}
		selected, err := selectLifecycleFeature("Which feature should be "+action+"d?", labels)
		if err != nil {
			return "", nil, err
		}
		return eligible[selected].Name, []string{eligible[selected].Shell}, nil
	}
	name := args[0]
	var installed []string
	for _, record := range records {
		if record.Name == name {
			installed = append(installed, record.Shell)
		}
	}
	if len(installed) == 0 {
		return "", nil, fmt.Errorf("feature %q is not installed in selected shells", name)
	}
	if len(state.flagShell) > 0 {
		for _, selected := range state.flagShell {
			found := false
			for _, candidate := range installed {
				if selected == candidate {
					found = true
				}
			}
			if !found {
				return "", nil, fmt.Errorf("feature %q is not installed in %s", name, selected)
			}
		}
		return name, installed, nil
	}
	if state.flagAll || len(installed) == 1 {
		return name, installed, nil
	}
	if current, err := shell.DetectCurrentShell(); err == nil {
		for _, candidate := range installed {
			if candidate == current {
				return name, []string{current}, nil
			}
		}
	}
	if !interactive.ShouldPrompt(cmd, false) {
		return "", nil, fmt.Errorf("feature %q is installed in multiple shells; specify --shell or --all", name)
	}
	selected, err := selectLifecycleShells("Change "+name+" in which shells?", installed, nil)
	return name, selected, err
}

func (state *commandState) runFeatureEnable(cmd *cobra.Command, args []string) error {
	return state.runFeatureLoad(cmd, args, false)
}
func (state *commandState) runFeatureDisable(cmd *cobra.Command, args []string) error {
	return state.runFeatureLoad(cmd, args, true)
}

func (state *commandState) runFeatureLoad(cmd *cobra.Command, args []string, disabled bool) error {
	if err := interactive.ValidateMode(cmd); err != nil {
		return err
	}
	name, shells, err := state.loadSelection(cmd, args, disabled)
	if err != nil {
		return err
	}
	if len(shells) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No matching feature changes selected.")
		return nil
	}
	plans := make([]*shell.FeatureLoadPlan, 0, len(shells))
	changed := false
	for _, shellName := range shells {
		var triggers []string
		if len(state.flagOnCommand) > 0 {
			triggers = state.flagOnCommand
		}
		plan, err := shell.PrepareFeatureLoadConfiguration(state.repoPathProvider(), shellName, name, disabled, state.flagStrategy, triggers, state.flagLocal)
		if err != nil {
			return fmt.Errorf("prepare %s: %w", shellName, err)
		}
		plans = append(plans, plan)
		changed = changed || plan.Changed
	}
	if !changed {
		fmt.Fprintf(cmd.OutOrStdout(), "%s already has the requested load settings.\n", name)
		return nil
	}
	if !state.flagLocal {
		if err := preflightFeatureMutation(state.repoPathProvider()); err != nil {
			return err
		}
	}
	var failures []error
	completed := 0
	for index, plan := range plans {
		if !plan.Changed {
			continue
		}
		if err := plan.Apply(); err != nil {
			failures = append(failures, fmt.Errorf("change %s in %s: %w", name, shells[index], err))
			continue
		}
		completed++
	}
	if completed > 0 && !state.flagLocal {
		action := "Enable"
		if disabled {
			action = "Disable"
		}
		if err := autoCommitShellFeatureChanges(action + " shell feature: " + name); err != nil {
			failures = append(failures, err)
		}
	}
	if completed > 0 {
		if state.flagLocal {
			fmt.Fprintln(cmd.OutOrStdout(), "Device-local load settings updated; no commit was created.")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s in %d shell(s).\n", name, completed)
			if records, err := featureRecords(state.repoPathProvider(), shells); err == nil {
				for _, record := range records {
					if record.Name == name && record.Overridden && record.Disabled != disabled {
						fmt.Fprintf(cmd.OutOrStdout(), "A local override still controls %s in %s. Use --local to change this device, or feature remove %s --shell %s --local to inherit shared settings.\n", name, record.Shell, name, record.Shell)
					}
				}
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Open a new shell session to load the changed features.")
	}
	return errors.Join(failures...)
}

func (state *commandState) configureSelection(cmd *cobra.Command, args []string) (string, []featureRecord, error) {
	records, err := featureRecords(state.repoPathProvider(), state.flagShell)
	if err != nil {
		return "", nil, err
	}
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		if !interactive.ShouldPrompt(cmd, false) {
			return "", nil, fmt.Errorf("feature name required; use feature configure <feature> --shell <shell> --option key=value")
		}
		names := make(map[string]bool)
		for _, record := range records {
			if metadata, ok := catalog.GetFeature(record.Name); ok && len(metadata.Options) > 0 {
				names[record.Name] = true
			}
		}
		var choices []string
		for name := range names {
			choices = append(choices, name)
		}
		sort.Strings(choices)
		if len(choices) == 0 {
			return "", nil, fmt.Errorf("no installed catalog features have configurable options")
		}
		index, err := selectLifecycleFeature("Configure which installed feature?", choices)
		if err != nil {
			return "", nil, err
		}
		name = choices[index]
	}
	selected := make([]featureRecord, 0)
	for _, record := range records {
		if record.Name == name {
			selected = append(selected, record)
		}
	}
	if len(selected) == 0 {
		return "", nil, fmt.Errorf("feature %q is not installed in selected shells", name)
	}
	if len(selected) > 1 && len(state.flagShell) == 0 {
		if !interactive.ShouldPrompt(cmd, false) {
			return "", nil, fmt.Errorf("feature %q is installed in multiple shells; specify --shell", name)
		}
		labels := make([]string, len(selected))
		for index, record := range selected {
			labels[index] = record.Shell
		}
		chosen, err := selectLifecycleShells("Configure "+name+" in which shells?", labels, nil)
		if err != nil {
			return "", nil, err
		}
		filtered := selected[:0]
		for _, record := range selected {
			for _, shellName := range chosen {
				if record.Shell == shellName {
					filtered = append(filtered, record)
				}
			}
		}
		selected = filtered
	}
	if len(state.flagShell) > 0 {
		for _, shellName := range state.flagShell {
			found := false
			for _, record := range selected {
				found = found || record.Shell == shellName
			}
			if !found {
				return "", nil, fmt.Errorf("feature %q is not installed in %s", name, shellName)
			}
		}
	}
	return name, selected, nil
}

func showFeatureRequirements(cmd *cobra.Command, repoPath, shellName string) {
	warnings, err := shell.FeatureRequirementWarnings(repoPath, shellName)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %v\n", err)
		return
	}
	for _, warning := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", warning)
	}
}
