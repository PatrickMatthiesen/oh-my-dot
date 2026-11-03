package options

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
)

func TestOptionalBoolHasOneQuestionAndBlankPathIsSkipped(t *testing.T) {
	metadata := catalog.FeatureMetadata{Options: []catalog.OptionMetadata{
		{Name: "config", DisplayName: "Config File", Type: catalog.OptionTypeFile, PathMustExist: true, FileOnly: true},
		{Name: "upgrade", DisplayName: "Auto Upgrade", Type: catalog.OptionTypeBool, Default: false},
	}}
	confirms, inputs := 0, 0
	values, err := promptForOptions(metadata, nil, optionPrompter{
		input: func(question, defaultValue string) (string, error) {
			inputs++
			if !strings.Contains(question, "skip") {
				t.Fatal("missing blank-input hint")
			}
			return "", nil
		},
		confirm: func(question string, defaultValue bool) (bool, error) {
			confirms++
			if strings.Contains(question, "Configure") {
				t.Fatal("unexpected configure gate")
			}
			return true, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if confirms != 1 || inputs != 1 || values["upgrade"] != true {
		t.Fatalf("questions=%d,%d values=%v", confirms, inputs, values)
	}
	if _, exists := values["config"]; exists {
		t.Fatal("blank optional path was stored")
	}
}

func TestBackPreservesAnswersAndOverrides(t *testing.T) {
	metadata := catalog.FeatureMetadata{Options: []catalog.OptionMetadata{
		{Name: "fixed", Type: catalog.OptionTypeString},
		{Name: "theme", Type: catalog.OptionTypeEnum, Required: true, EnumValues: []string{"a", "b"}},
		{Name: "path", Type: catalog.OptionTypeFile},
	}}
	selects, inputs := 0, 0
	values, err := promptForOptions(metadata, map[string]any{"fixed": "override"}, optionPrompter{
		selectOption: func(_ string, _ []string, selected int) (int, error) {
			selects++
			if selects == 2 && selected != 1 {
				t.Fatalf("previous theme lost: selected=%d", selected)
			}
			return 1, nil
		},
		input: func(_, _ string) (string, error) {
			inputs++
			if inputs == 1 {
				return "", interactive.ErrBack
			}
			return "", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if selects != 2 || inputs != 2 || values["theme"] != "b" || values["fixed"] != "override" {
		t.Fatalf("values=%v selects=%d inputs=%d", values, selects, inputs)
	}
}

func TestInvalidPathRetriesOnlyCurrentOption(t *testing.T) {
	metadata := catalog.FeatureMetadata{Options: []catalog.OptionMetadata{
		{Name: "theme", Type: catalog.OptionTypeEnum, Required: true, EnumValues: []string{"a"}},
		{Name: "path", Type: catalog.OptionTypeFile, PathMustExist: true, FileOnly: true},
	}}
	selects, inputs := 0, 0
	values, err := promptForOptions(metadata, nil, optionPrompter{
		selectOption: func(string, []string, int) (int, error) { selects++; return 0, nil },
		input: func(string, string) (string, error) {
			inputs++
			if inputs == 1 {
				return t.TempDir() + "/missing", nil
			}
			return "", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if selects != 1 || inputs != 2 || values["theme"] != "a" {
		t.Fatalf("values=%v selects=%d inputs=%d", values, selects, inputs)
	}
}

func TestWizardPreservesSkipAndCancelErrors(t *testing.T) {
	for _, sentinel := range []error{interactive.ErrSkip, interactive.ErrCancelled} {
		t.Run(sentinel.Error(), func(t *testing.T) {
			_, err := promptForOptions(catalog.FeatureMetadata{Options: []catalog.OptionMetadata{{Name: "enabled", Type: catalog.OptionTypeBool}}}, nil, optionPrompter{
				confirm: func(string, bool) (bool, error) { return false, fmt.Errorf("wrapped: %w", sentinel) },
			})
			if !errors.Is(err, sentinel) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
