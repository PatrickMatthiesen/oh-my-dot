package featurecmd

import (
	"errors"
	"fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
)

func (state *commandState) localFeatureTargets(cmd *cobra.Command, args []string, refresh bool) ([]featureRecord, error) {
	if state.flagInteractive && len(args) > 0 {
		return nil, fmt.Errorf("interactive mode does not accept a feature name")
	}
	records, err := featureRecords(state.repoPathProvider(), state.flagShell)
	if err != nil {
		return nil, err
	}
	eligible := make([]featureRecord, 0)
	for _, record := range records {
		_, local, _, err := readFeatureManifest(state.repoPathProvider(), record.Shell)
		if err != nil {
			return nil, err
		}
		if local == nil {
			continue
		}
		if _, err := local.GetFeature(record.Name); err != nil {
			continue
		}
		if refresh && !catalog.HasFeatureTemplate(record.Name, record.Shell) {
			continue
		}
		if refresh && record.Origin != "local" {
			sharedPath, err := shell.GetFeatureFilePath(state.repoPathProvider(), record.Shell, record.Name)
			if err != nil {
				return nil, err
			}
			if record.Path == sharedPath {
				continue // A load-only override does not own a private implementation.
			}
		}
		eligible = append(eligible, record)
	}
	if len(args) > 0 {
		name := args[0]
		if err := manifest.ValidateFeatureName(name); err != nil {
			return nil, err
		}
		selected := make([]featureRecord, 0)
		for _, record := range eligible {
			if record.Name == name {
				selected = append(selected, record)
			}
		}
		if len(selected) == 0 {
			if refresh {
				return nil, fmt.Errorf("no private catalog template for %q in selected shells", name)
			}
			return nil, nil
		}
		if len(state.flagShell) > 0 || state.flagAll || len(selected) == 1 {
			return selected, nil
		}
		if !interactive.ShouldPrompt(cmd, false) {
			return nil, fmt.Errorf("private feature %q is installed in multiple shells; specify --shell or --all", name)
		}
		var names []string
		for _, record := range selected {
			names = append(names, record.Shell)
		}
		chosen, err := selectLifecycleShells("Change private "+name+" in which shells?", names, nil)
		if err != nil {
			return nil, err
		}
		filtered := make([]featureRecord, 0, len(chosen))
		for _, record := range selected {
			for _, shellName := range chosen {
				if record.Shell == shellName {
					filtered = append(filtered, record)
				}
			}
		}
		return filtered, nil
	}
	if refresh && state.flagAll {
		return eligible, nil
	}
	if !interactive.ShouldPrompt(cmd, false) {
		return nil, fmt.Errorf("feature name required; specify a feature and --shell (or --all), or run in a terminal to browse private features")
	}
	if len(eligible) == 0 {
		return eligible, nil
	}
	labels := make([]string, len(eligible))
	for index, record := range eligible {
		labels[index] = lifecycleRecordLabel(record)
	}
	index, err := selectLifecycleFeature("Choose a private feature:", labels)
	if err != nil {
		return nil, err
	}
	return []featureRecord{eligible[index]}, nil
}

func (state *commandState) runLocalFeatureUpdate(cmd *cobra.Command, args []string) error {
	records, err := state.localFeatureTargets(cmd, args, true)
	if err != nil {
		return err
	}
	plans := make([]*shell.FeatureConfigurationPlan, 0, len(records))
	for _, record := range records {
		plan, err := shell.PrepareLocalFeatureRefresh(state.repoPathProvider(), record.Shell, record.Name, record.Options, state.flagForce)
		if err != nil {
			return fmt.Errorf("prepare private %s in %s: %w", record.Name, record.Shell, err)
		}
		plans = append(plans, plan)
	}
	var failures []error
	completed := 0
	for index, plan := range plans {
		if !plan.Changed {
			continue
		}
		if err := plan.Apply(); err != nil {
			failures = append(failures, fmt.Errorf("refresh private %s in %s: %w", records[index].Name, records[index].Shell, err))
			continue
		}
		completed++
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Refreshed %d private template(s); no commit was created.\n", completed)
	if completed > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Open a new shell session to load the changes.")
	}
	return errors.Join(failures...)
}

func (state *commandState) runLocalFeatureRemove(cmd *cobra.Command, args []string) error {
	records, err := state.localFeatureTargets(cmd, args, false)
	if err != nil {
		return err
	}
	plans := make([]*shell.LocalFeatureRemovalPlan, 0, len(records))
	for _, record := range records {
		plan, err := shell.PrepareLocalFeatureRemoval(state.repoPathProvider(), record.Shell, record.Name)
		if err != nil {
			return fmt.Errorf("prepare private removal of %s in %s: %w", record.Name, record.Shell, err)
		}
		plans = append(plans, plan)
	}
	var failures []error
	completed := 0
	for index, plan := range plans {
		if !plan.Changed {
			continue
		}
		if err := plan.Apply(); err != nil {
			failures = append(failures, fmt.Errorf("remove private %s in %s: %w", records[index].Name, records[index].Shell, err))
			continue
		}
		completed++
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Removed %d private feature configuration(s); shared settings are unchanged and no commit was created.\n", completed)
	if completed > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Open a new shell session to load the changes.")
	}
	return errors.Join(failures...)
}
