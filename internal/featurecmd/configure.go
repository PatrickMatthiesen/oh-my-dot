package featurecmd

import (
	"errors"
	"fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/options"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/validation"
	"github.com/spf13/cobra"
)

var promptExistingFeatureOptions = options.PromptForOptionsWithOverrides

func (state *commandState) runFeatureConfigure(cmd *cobra.Command, args []string) error {
	if err := interactive.ValidateMode(cmd); err != nil {
		return err
	}
	name, records, err := state.configureSelection(cmd, args)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No configuration changes selected.")
		return nil
	}
	metadata, ok := catalog.GetFeature(name)
	if !ok || len(metadata.Options) == 0 {
		return fmt.Errorf("feature %q has no catalog options to configure", name)
	}
	overrides, err := options.ParseOptionOverrides(metadata, state.flagOption)
	if err != nil {
		return err
	}
	reset := make(map[string]catalog.OptionMetadata)
	for _, key := range state.flagResetOption {
		found := false
		for _, opt := range metadata.Options {
			if opt.Name == key {
				if opt.Required && opt.Default == nil && !state.flagLocal {
					return fmt.Errorf("required option %q has no default and cannot be reset", key)
				}
				reset[key] = opt
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown reset option %q for %s", key, name)
		}
		if _, set := overrides[key]; set {
			return fmt.Errorf("option %q cannot be changed and reset together", key)
		}
	}
	interactiveEdit := len(state.flagOption) == 0 && len(state.flagResetOption) == 0
	if interactiveEdit && !interactive.ShouldPrompt(cmd, false) {
		return fmt.Errorf("no changes specified; pass --option key=value or --reset-option name, or run in a terminal to edit current values")
	}
	plans := make([]*shell.FeatureConfigurationPlan, 0, len(records))
	changed := false
	for _, record := range records {
		saved := record.Options
		if !state.flagLocal {
			base, _, _, err := readFeatureManifest(state.repoPathProvider(), record.Shell)
			if err != nil {
				return err
			}
			shared, err := base.GetFeature(name)
			if err != nil {
				return fmt.Errorf("%s is local-only in %s; use --local", name, record.Shell)
			}
			saved = shared.Options
		}
		current := make(map[string]any, len(saved))
		for key, value := range saved {
			current[key] = value
		}
		var values map[string]any
		if interactiveEdit {
			promptMetadata := metadata
			promptMetadata.Options = append([]catalog.OptionMetadata(nil), metadata.Options...)
			for index, opt := range promptMetadata.Options {
				if value, exists := current[opt.Name]; exists {
					promptMetadata.Options[index].Default = value
				}
			}
			values, err = promptExistingFeatureOptions(promptMetadata, nil)
			if err != nil {
				return fmt.Errorf("configure %s in %s: %w", name, record.Shell, err)
			}
		} else {
			values = current
			for key, value := range overrides {
				values[key] = value
			}
			for key, opt := range reset {
				if state.flagLocal {
					base, _, _, err := readFeatureManifest(state.repoPathProvider(), record.Shell)
					if err != nil {
						return err
					}
					shared, _ := base.GetFeature(name)
					if shared != nil {
						if value, exists := shared.Options[key]; exists {
							values[key] = value
							continue
						}
					}
				}
				if opt.Required && opt.Default == nil {
					return fmt.Errorf("required option %q has no shared value or catalog default to inherit", key)
				}
				if opt.Default == nil {
					delete(values, key)
				} else {
					values[key] = opt.Default
				}
			}
		}
		for _, opt := range metadata.Options {
			value, exists := values[opt.Name]
			if !exists && opt.Default != nil {
				value = opt.Default
			}
			if err := validation.ValidateOption(opt, value); err != nil {
				return fmt.Errorf("invalid %s option %s: %w", record.Shell, opt.Name, err)
			}
		}
		prepare := shell.PrepareFeatureConfiguration
		if state.flagLocal {
			prepare = shell.PrepareLocalFeatureConfiguration
		}
		plan, err := prepare(state.repoPathProvider(), record.Shell, name, values, state.flagForce)
		if err != nil {
			return fmt.Errorf("prepare %s configuration: %w", record.Shell, err)
		}
		plans = append(plans, plan)
		changed = changed || plan.Changed
	}
	if !changed {
		fmt.Fprintf(cmd.OutOrStdout(), "%s already has the requested options.\n", name)
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
			failures = append(failures, fmt.Errorf("configure %s: %w", records[index].Shell, err))
			continue
		}
		completed++
	}
	if completed > 0 {
		if state.flagLocal {
			fmt.Fprintln(cmd.OutOrStdout(), "Device-local options updated; no commit was created.")
		} else {
			if err := autoCommitShellFeatureChanges("Configure shell feature: " + name); err != nil {
				failures = append(failures, err)
			}
		}
		for _, record := range records {
			showFeatureRequirements(cmd, state.repoPathProvider(), record.Shell)
			if !state.flagLocal && record.Overridden {
				fmt.Fprintf(cmd.OutOrStdout(), "Shared settings for %s changed. A local override is still active; feature info shows the effective settings.\n", record.Shell)
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Configured %s in %d shell(s). Open a new shell session to load the changes.\n", name, completed)
	}
	return errors.Join(failures...)
}
