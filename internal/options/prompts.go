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

func hasNonEmptyDefault(val any) bool {
	return val != nil && fmt.Sprintf("%v", val) != ""
}

// formatOptionQuestion formats a concise single-line prompt for an option,
// combining requirement, constraint, and blank-input hints into one parenthetical.
func formatOptionQuestion(opt catalog.OptionMetadata) string {
	question := opt.DisplayName
	if opt.Description != "" {
		question = fmt.Sprintf("%s - %s", question, opt.Description)
	}

	var hints []string
	if opt.Required {
		hints = append(hints, "required")
	}

	switch opt.Type {
	case catalog.OptionTypeInt:
		if opt.IntMin != nil {
			hints = append(hints, fmt.Sprintf("min: %d", *opt.IntMin))
		}
		if opt.IntMax != nil {
			hints = append(hints, fmt.Sprintf("max: %d", *opt.IntMax))
		}
	case catalog.OptionTypeFile, catalog.OptionTypePath:
		switch {
		case opt.PathMustExist && opt.FileOnly:
			hints = append(hints, "existing file")
		case opt.PathMustExist:
			hints = append(hints, "must exist")
		case opt.FileOnly:
			hints = append(hints, "file only")
		}
	}

	if !opt.Required && (opt.Type == catalog.OptionTypeFile || opt.Type == catalog.OptionTypePath || opt.Type == catalog.OptionTypeString) {
		if hasNonEmptyDefault(opt.Default) {
			hints = append(hints, "Enter to keep")
		} else {
			hints = append(hints, "Enter to skip")
		}
	}

	if len(hints) > 0 {
		question = fmt.Sprintf("%s (%s)", question, strings.Join(hints, ", "))
	}

	return question
}

// promptForOption prompts for a single option based on its type
func (p optionPrompter) promptForOption(opt catalog.OptionMetadata) (any, error) {
	question := formatOptionQuestion(opt)

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

	// If empty and has a non-empty default, use default
	if value == "" && hasNonEmptyDefault(opt.Default) {
		return opt.Default, nil
	}

	if value == "" && !opt.Required {
		return nil, nil
	}
	return value, nil
}

// promptInt prompts for integer input
func (p optionPrompter) promptInt(opt catalog.OptionMetadata, question string) (any, error) {
	defaultStr := ""
	if opt.Default != nil {
		defaultStr = fmt.Sprintf("%v", opt.Default)
	}

	for {
		value, err := p.input(question, defaultStr)
		if err != nil {
			return nil, err
		}

		// If empty and has a non-empty default, use default
		if value == "" && hasNonEmptyDefault(opt.Default) {
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

		// If empty and has a non-empty default, use default
		if value == "" && hasNonEmptyDefault(opt.Default) {
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
