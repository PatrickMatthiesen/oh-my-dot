package featurecmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func (state *commandState) runFeatureList(cmd *cobra.Command, args []string) error {
	if err := state.checkInspectionRepository(); err != nil {
		return err
	}
	records, err := featureRecords(state.repoPathProvider(), state.flagShell)
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		return writeFeatureJSON(cmd, featureInventory{SchemaVersion: 1, Features: records})
	}
	if len(records) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No shell features configured. Run '%s feature add' to browse.\n", state.aliasProvider())
		return nil
	}
	last := ""
	for _, record := range records {
		if record.Shell != last {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s:\n", record.Shell)
			last = record.Shell
		}
		status := "enabled"
		if record.Disabled {
			status = "disabled"
		}
		origin := record.Origin
		if record.Overridden {
			origin = "shared, overridden locally"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s, %s; %s)\n    %s\n", record.Name, status, record.Strategy, origin, record.Path)
		if len(record.Options) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "    options: %v\n", record.Options)
		}
	}
	return nil
}

type optionDescription struct {
	Name        string             `json:"name"`
	Type        catalog.OptionType `json:"type"`
	Description string             `json:"description"`
	Required    bool               `json:"required"`
	Default     any                `json:"default,omitempty"`
	Values      []string           `json:"values,omitempty"`
}

type requirementDescription struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Instruction string `json:"instruction"`
}

type featureDescription struct {
	Requirements    map[string][]requirementDescription `json:"requirements,omitempty"`
	Name            string                              `json:"name"`
	Description     string                              `json:"description"`
	Category        string                              `json:"category"`
	DefaultStrategy string                              `json:"defaultStrategy"`
	DefaultCommands []string                            `json:"defaultCommands,omitempty"`
	SupportedShells []string                            `json:"supportedShells"`
	Options         []optionDescription                 `json:"options"`
}

type featureInfoReport struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Catalog       *featureDescription `json:"catalog"`
	Installed     []featureRecord     `json:"installed"`
}

func (state *commandState) runFeatureInfo(cmd *cobra.Command, args []string) error {
	if viper.GetBool("initialized") {
		if err := state.checkInspectionRepository(); err != nil {
			return err
		}
	}
	records, err := configuredRecordsOrEmpty(state.repoPathProvider(), state.flagShell)
	if err != nil {
		return err
	}
	name := ""
	if len(args) > 0 {
		name = args[0]
	} else {
		if !interactive.ShouldPrompt(cmd, false) {
			return fmt.Errorf("feature name required; use feature info <feature> or run in a terminal to browse")
		}
		available := make(map[string]bool, len(catalog.Catalog)+len(records))
		for name := range catalog.Catalog {
			available[name] = true
		}
		for _, record := range records {
			available[record.Name] = true
		}
		names := make([]string, 0, len(available))
		for name := range available {
			names = append(names, name)
		}
		sort.Strings(names)
		index, err := selectLifecycleFeature("Show information about which feature?", names)
		if err != nil {
			return err
		}
		name = names[index]
	}
	installed := make([]featureRecord, 0)
	for _, record := range records {
		if record.Name == name {
			installed = append(installed, record)
		}
	}
	metadata, ok := catalog.GetFeature(name)
	if !ok {
		if len(installed) == 0 {
			return fmt.Errorf("feature %q is neither installed nor in the catalog", name)
		}
		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			return writeFeatureJSON(cmd, featureInfoReport{SchemaVersion: 1, Installed: installed})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n  Custom shell feature. Edit its implementation to change its behavior.\n", name)
		printEffectiveFeatureInfo(cmd, installed)
		return nil
	}
	description := featureDescription{Name: metadata.Name, Description: metadata.Description, Category: metadata.Category, DefaultStrategy: metadata.DefaultStrategy, DefaultCommands: metadata.DefaultCommands, SupportedShells: metadata.SupportedShells, Options: make([]optionDescription, 0, len(metadata.Options))}
	description.Requirements = make(map[string][]requirementDescription)
	for _, shellName := range metadata.SupportedShells {
		for _, requirement := range catalog.Requirements(name, shellName) {
			description.Requirements[shellName] = append(description.Requirements[shellName], requirementDescription{Kind: requirement.Kind, Name: requirement.Name, Instruction: requirement.Instruction})
		}
	}
	for _, opt := range metadata.Options {
		description.Options = append(description.Options, optionDescription{Name: opt.Name, Type: opt.Type, Description: opt.Description, Required: opt.Required, Default: opt.Default, Values: opt.EnumValues})
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		return writeFeatureJSON(cmd, featureInfoReport{SchemaVersion: 1, Catalog: &description, Installed: installed})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n  %s\n  Category: %s\n  Default strategy: %s\n  Supported shells: %s\n", metadata.Name, metadata.Description, metadata.Category, metadata.DefaultStrategy, strings.Join(metadata.SupportedShells, ", "))
	if len(description.Options) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nOptions (feature add or feature configure --option name=value):")
		for _, opt := range description.Options {
			required := "optional"
			if opt.Required {
				required = "required"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s, %s)", opt.Name, opt.Type, required)
			if opt.Default != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "; default: %v", opt.Default)
			}
			if len(opt.Values) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "; values: %s", strings.Join(opt.Values, ", "))
			}
			fmt.Fprintln(cmd.OutOrStdout())
			if opt.Description != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", opt.Description)
			}
		}
	}
	if len(description.Requirements) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nPrerequisites (configuration does not install these):")
		seen := map[string]bool{}
		for _, shellName := range metadata.SupportedShells {
			for _, requirement := range description.Requirements[shellName] {
				if !seen[requirement.Instruction] {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", requirement.Instruction)
					seen[requirement.Instruction] = true
				}
			}
		}
	}
	printEffectiveFeatureInfo(cmd, installed)
	return nil
}

func (state *commandState) checkInspectionRepository() error {
	path := state.repoPathProvider()
	repo, err := git.GetRepository(path)
	if err != nil {
		return fmt.Errorf("inspect configured repository %s: %w", path, err)
	}
	return repo.Close()
}

func printEffectiveFeatureInfo(cmd *cobra.Command, installed []featureRecord) {
	fmt.Fprintln(cmd.OutOrStdout(), "\nEffective configuration:")
	if len(installed) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  (not installed)")
	}
	for _, record := range installed {
		status := "enabled"
		if record.Disabled {
			status = "disabled"
		}
		origin := record.Origin
		if record.Overridden {
			origin = "shared, overridden locally"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s (%s; %s)\n", record.Shell, status, record.Strategy, origin)
		fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", record.Path)
		if len(record.Options) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "    options: %v\n", record.Options)
		}
	}
}
