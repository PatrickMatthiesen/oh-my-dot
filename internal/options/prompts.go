package options

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/validation"
)

// ParseOptionOverrides parses and validates --option key=value pairs against metadata.
func ParseOptionOverrides(metadata catalog.FeatureMetadata, rawOptions []string) (map[string]any, error) {
	overrides := make(map[string]any)
	if len(rawOptions) == 0 {
		return overrides, nil
	}

	optionByName := make(map[string]catalog.OptionMetadata, len(metadata.Options))
	for _, opt := range metadata.Options {
		optionByName[opt.Name] = opt
	}

	for _, raw := range rawOptions {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid --option format '%s' (expected key=value)", raw)
		}

		key := strings.TrimSpace(parts[0])
		valueStr := parts[1]

		opt, exists := optionByName[key]
		if !exists {
			return nil, fmt.Errorf("unknown option '%s' for feature '%s'", key, metadata.Name)
		}

		parsedValue, err := parseOptionValue(opt, valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid value for option '%s': %w", key, err)
		}

		if err := validation.ValidateOption(opt, parsedValue); err != nil {
			return nil, fmt.Errorf("invalid value for option '%s': %w", key, err)
		}

		overrides[key] = parsedValue
	}

	return overrides, nil
}

// PromptForOptions collects user input for feature options interactively
func PromptForOptions(metadata catalog.FeatureMetadata) (map[string]any, error) {
	return PromptForOptionsWithOverrides(metadata, nil)
}

// PromptForOptionsWithOverrides collects user input for feature options interactively,
// using any provided overrides as fixed values.
func PromptForOptionsWithOverrides(metadata catalog.FeatureMetadata, overrides map[string]any) (map[string]any, error) {
	return promptForOptions(metadata, overrides, optionPrompter{
		input:        interactive.PromptInputWithNavigation,
		confirm:      interactive.ConfirmWithNavigation,
		selectOption: interactive.PromptSelectWithNavigation,
	})
}

type optionPrompter struct {
	input        func(string, string) (string, error)
	confirm      func(string, bool) (bool, error)
	selectOption func(string, []string, int) (int, error)
}

func promptForOptions(metadata catalog.FeatureMetadata, overrides map[string]any, prompter optionPrompter) (map[string]any, error) {
	values := make(map[string]any)
	for key, value := range overrides {
		values[key] = value
	}
	var editable []catalog.OptionMetadata
	for _, opt := range metadata.Options {
		if _, fixed := overrides[opt.Name]; !fixed {
			editable = append(editable, opt)
		}
	}
	for index := 0; index < len(editable); {
		opt := editable[index]
		if previous, answered := values[opt.Name]; answered {
			opt.Default = previous
		}
		value, err := prompter.promptOption(opt)
		if errors.Is(err, interactive.ErrBack) {
			if index > 0 {
				index--
			}
			continue
		}
		if errors.Is(err, interactive.ErrClear) {
			if opt.Required {
				fileops.ColorPrintfn(fileops.Yellow, "%s is required and cannot be cleared.", opt.DisplayName)
				continue
			}
			delete(values, opt.Name)
			index++
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("configure %s: %w", opt.DisplayName, err)
		}
		if err := validation.ValidateOption(opt, value); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Validation failed: %v. Please try again.", err)
			continue
		}
		if value == nil {
			delete(values, opt.Name)
		} else {
			values[opt.Name] = value
		}
		index++
	}
	return values, nil
}

func (p optionPrompter) promptOption(opt catalog.OptionMetadata) (any, error) {
	// Booleans are already a yes/no question. Text and paths can be left blank.
	if !opt.Required && (opt.Type == catalog.OptionTypeInt || opt.Type == catalog.OptionTypeEnum) {
		configure, err := p.confirm(fmt.Sprintf("Configure %s?", opt.DisplayName), false)
		if err != nil {
			return nil, err
		}
		if !configure {
			return opt.Default, nil
		}
	}
	return p.promptForOption(opt)
}

// ResolveOptionsForNonInteractive resolves feature options without prompting.
// Required options must have valid defaults in non-interactive mode.
func ResolveOptionsForNonInteractive(metadata catalog.FeatureMetadata) (map[string]any, error) {
	return ResolveOptionsForNonInteractiveWithOverrides(metadata, nil)
}

// ResolveOptionsForNonInteractiveWithOverrides resolves feature options without prompting.
// Required options must have valid defaults or explicit overrides in non-interactive mode.
func ResolveOptionsForNonInteractiveWithOverrides(metadata catalog.FeatureMetadata, overrides map[string]any) (map[string]any, error) {
	values := make(map[string]any)
	for key, value := range overrides {
		values[key] = value
	}

	for _, opt := range metadata.Options {
		if _, hasOverride := values[opt.Name]; hasOverride {
			continue
		}

		value, hasValue, err := resolveOptionDefault(opt)
		if err != nil {
			return nil, err
		}

		if hasValue {
			values[opt.Name] = value
		}
	}

	return values, nil
}

func parseOptionValue(opt catalog.OptionMetadata, valueStr string) (any, error) {
	switch opt.Type {
	case catalog.OptionTypeString, catalog.OptionTypeEnum:
		return valueStr, nil
	case catalog.OptionTypeInt:
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			return nil, fmt.Errorf("expected integer")
		}
		return value, nil
	case catalog.OptionTypeBool:
		value, err := validation.ParseBool(valueStr)
		if err != nil {
			return nil, err
		}
		return value, nil
	case catalog.OptionTypeFile, catalog.OptionTypePath:
		return validation.ExpandPath(valueStr)
	default:
		return nil, fmt.Errorf("unsupported option type: %s", opt.Type)
	}
}

func resolveOptionDefault(opt catalog.OptionMetadata) (any, bool, error) {
	if opt.Default == nil {
		if opt.Required {
			return nil, false, fmt.Errorf("required option '%s' has no default and cannot be resolved in non-interactive mode", opt.DisplayName)
		}

		return nil, false, nil
	}

	if err := validation.ValidateOption(opt, opt.Default); err != nil {
		return nil, false, fmt.Errorf("invalid default value for option '%s': %w", opt.DisplayName, err)
	}

	return opt.Default, true, nil
}

// promptForOption prompts for a single option based on its type
func (p optionPrompter) promptForOption(opt catalog.OptionMetadata) (any, error) {
	// Build question with description
	question := opt.DisplayName
	if opt.Description != "" {
		question = fmt.Sprintf("%s - %s", question, opt.Description)
	}
	if opt.Required {
		question = fmt.Sprintf("%s (required)", question)
	} else if opt.Type == catalog.OptionTypeFile || opt.Type == catalog.OptionTypePath || opt.Type == catalog.OptionTypeString {
		question += " (Enter to keep default or skip)"
	}

	switch opt.Type {
	case catalog.OptionTypeString:
		return p.promptString(opt, question)
	case catalog.OptionTypeInt:
		return p.promptInt(opt, question)
	case catalog.OptionTypeBool:
		return p.promptBool(opt, question)
	case catalog.OptionTypeEnum:
		return p.promptEnum(opt, question)
	case catalog.OptionTypeFile, catalog.OptionTypePath:
		return p.promptPath(opt, question)
	default:
		return nil, fmt.Errorf("unsupported option type: %s", opt.Type)
	}
}

// promptString prompts for string input
func (p optionPrompter) promptString(opt catalog.OptionMetadata, question string) (any, error) {
	defaultStr := ""
	if opt.Default != nil {
		defaultStr = fmt.Sprintf("%v", opt.Default)
	}

	value, err := p.input(question, defaultStr)
	if err != nil {
		return nil, err
	}

	// If empty and has default, use default
	if value == "" && opt.Default != nil {
		return opt.Default, nil
	}

	if value == "" && !opt.Required {
		return nil, nil
	}
	return value, nil
}

// promptInt prompts for integer input
func (p optionPrompter) promptInt(opt catalog.OptionMetadata, question string) (any, error) {
	// Build constraints string
	var constraints []string
	if opt.IntMin != nil {
		constraints = append(constraints, fmt.Sprintf("min: %d", *opt.IntMin))
	}
	if opt.IntMax != nil {
		constraints = append(constraints, fmt.Sprintf("max: %d", *opt.IntMax))
	}
	if len(constraints) > 0 {
		question = fmt.Sprintf("%s (%s)", question, strings.Join(constraints, ", "))
	}

	defaultStr := ""
	if opt.Default != nil {
		defaultStr = fmt.Sprintf("%v", opt.Default)
	}

	for {
		value, err := p.input(question, defaultStr)
		if err != nil {
			return nil, err
		}

		// If empty and has default, use default
		if value == "" && opt.Default != nil {
			return opt.Default, nil
		}

		// Parse as integer
		intVal, err := strconv.Atoi(value)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Invalid integer. Please try again.")
			continue
		}

		return intVal, nil
	}
}

// promptBool prompts for boolean input
func (p optionPrompter) promptBool(opt catalog.OptionMetadata, question string) (any, error) {
	defaultBool := false
	if opt.Default != nil {
		if b, ok := opt.Default.(bool); ok {
			defaultBool = b
		}
	}

	value, err := p.confirm(question, defaultBool)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// promptEnum prompts for enum selection
func (p optionPrompter) promptEnum(opt catalog.OptionMetadata, question string) (any, error) {
	if len(opt.EnumValues) == 0 {
		return nil, fmt.Errorf("enum option has no valid values")
	}

	// Use PromptSelect for single selection
	selected := 0
	for i, value := range opt.EnumValues {
		if value == opt.Default {
			selected = i
			break
		}
	}
	selectedIdx, err := p.selectOption(question, opt.EnumValues, selected)
	if err != nil {
		return nil, err
	}

	return opt.EnumValues[selectedIdx], nil
}

// promptPath prompts for file/path input
func (p optionPrompter) promptPath(opt catalog.OptionMetadata, question string) (any, error) {
	// Add path requirements to question
	var hints []string
	if opt.PathMustExist {
		hints = append(hints, "must exist")
	}
	if opt.FileOnly {
		hints = append(hints, "file only")
	}
	if len(hints) > 0 {
		question = fmt.Sprintf("%s (%s)", question, strings.Join(hints, ", "))
	}

	defaultStr := ""
	if opt.Default != nil {
		defaultStr = fmt.Sprintf("%v", opt.Default)
	}

	// For now, use text input
	// Future enhancement: use file picker for better UX
	for {
		value, err := p.input(question, defaultStr)
		if err != nil {
			return nil, err
		}

		// If empty and has default, use default
		if value == "" && opt.Default != nil {
			return opt.Default, nil
		}

		if strings.TrimSpace(value) == "" && !opt.Required {
			return nil, nil
		}

		// Expand and validate path
		expandedPath, err := validation.ExpandPath(value)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Invalid path: %v. Please try again.", err)
			continue
		}

		return expandedPath, nil
	}
}

// HasRequiredOptions checks if a feature has any required options
func HasRequiredOptions(metadata catalog.FeatureMetadata) bool {
	for _, opt := range metadata.Options {
		if opt.Required {
			return true
		}
	}
	return false
}
