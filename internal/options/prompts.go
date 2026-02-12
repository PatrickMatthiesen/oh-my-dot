package options

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/validation"
)

// PromptForOptions collects user input for feature options interactively
func PromptForOptions(metadata catalog.FeatureMetadata) (map[string]any, error) {
	// If no options defined, return empty map
	if len(metadata.Options) == 0 {
		return map[string]any{}, nil
	}

	fileops.ColorPrintfn(fileops.Cyan, "\n⚙️  Feature Configuration\n")

	values := make(map[string]any)

	for _, opt := range metadata.Options {
		// For optional options, ask if user wants to configure
		if !opt.Required {
			question := fmt.Sprintf("Configure %s?", opt.DisplayName)
			if opt.Default != nil {
				question = fmt.Sprintf("%s (default: %v)", question, opt.Default)
			}

			configure, err := interactive.Confirm(question, false)
			if err != nil {
				return nil, fmt.Errorf("prompt cancelled: %w", err)
			}

			if !configure {
				// Use default value
				if opt.Default != nil {
					values[opt.Name] = opt.Default
				}
				continue
			}
		}

		// Prompt based on type
		value, err := promptForOption(opt)
		if err != nil {
			return nil, err
		}

		// Validate
		if err := validation.ValidateOption(opt, value); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Validation failed: %v", err)
			// Retry
			value, err = promptForOption(opt)
			if err != nil {
				return nil, err
			}
			// Validate again
			if err := validation.ValidateOption(opt, value); err != nil {
				return nil, fmt.Errorf("validation failed for %s: %w", opt.Name, err)
			}
		}

		values[opt.Name] = value
	}

	return values, nil
}

// promptForOption prompts for a single option based on its type
func promptForOption(opt catalog.OptionMetadata) (any, error) {
	// Build question with description
	question := opt.DisplayName
	if opt.Description != "" {
		question = fmt.Sprintf("%s - %s", question, opt.Description)
	}
	if opt.Required {
		question = fmt.Sprintf("%s (required)", question)
	}

	switch opt.Type {
	case catalog.OptionTypeString:
		return promptString(opt, question)
	case catalog.OptionTypeInt:
		return promptInt(opt, question)
	case catalog.OptionTypeBool:
		return promptBool(opt, question)
	case catalog.OptionTypeEnum:
		return promptEnum(opt, question)
	case catalog.OptionTypeFile, catalog.OptionTypePath:
		return promptPath(opt, question)
	default:
		return nil, fmt.Errorf("unsupported option type: %s", opt.Type)
	}
}

// promptString prompts for string input
func promptString(opt catalog.OptionMetadata, question string) (any, error) {
	defaultStr := ""
	if opt.Default != nil {
		defaultStr = fmt.Sprintf("%v", opt.Default)
	}

	value, err := interactive.PromptInput(question, defaultStr)
	if err != nil {
		return nil, err
	}

	// If empty and has default, use default
	if value == "" && opt.Default != nil {
		return opt.Default, nil
	}

	return value, nil
}

// promptInt prompts for integer input
func promptInt(opt catalog.OptionMetadata, question string) (any, error) {
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
		value, err := interactive.PromptInput(question, defaultStr)
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
func promptBool(opt catalog.OptionMetadata, question string) (any, error) {
	defaultBool := false
	if opt.Default != nil {
		if b, ok := opt.Default.(bool); ok {
			defaultBool = b
		}
	}

	value, err := interactive.Confirm(question, defaultBool)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// promptEnum prompts for enum selection
func promptEnum(opt catalog.OptionMetadata, question string) (any, error) {
	if len(opt.EnumValues) == 0 {
		return nil, fmt.Errorf("enum option has no valid values")
	}

	// Add description as header
	if opt.Description != "" {
		fileops.ColorPrintln(opt.Description, fileops.Reset)
	}

	// Use PromptSelect for single selection
	selectedIdx, err := interactive.PromptSelect(question, opt.EnumValues)
	if err != nil {
		return nil, err
	}

	return opt.EnumValues[selectedIdx], nil
}

// promptPath prompts for file/path input
func promptPath(opt catalog.OptionMetadata, question string) (any, error) {
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
		value, err := interactive.PromptInput(question, defaultStr)
		if err != nil {
			return nil, err
		}

		// If empty and has default, use default
		if value == "" && opt.Default != nil {
			return opt.Default, nil
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
