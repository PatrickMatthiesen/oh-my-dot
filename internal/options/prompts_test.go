package options

import (
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
)

func TestResolveOptionsForNonInteractive(t *testing.T) {
	metadata := catalog.FeatureMetadata{
		Name: "test-feature",
		Options: []catalog.OptionMetadata{
			{
				Name:     "required_with_default",
				Type:     catalog.OptionTypeString,
				Required: true,
				Default:  "value",
			},
			{
				Name:     "optional_with_default",
				Type:     catalog.OptionTypeBool,
				Required: false,
				Default:  true,
			},
			{
				Name:     "optional_without_default",
				Type:     catalog.OptionTypeString,
				Required: false,
			},
		},
	}

	values, err := ResolveOptionsForNonInteractive(metadata)
	if err != nil {
		t.Fatalf("ResolveOptionsForNonInteractive() error = %v", err)
	}

	if values["required_with_default"] != "value" {
		t.Fatalf("required_with_default = %v, want value", values["required_with_default"])
	}

	if values["optional_with_default"] != true {
		t.Fatalf("optional_with_default = %v, want true", values["optional_with_default"])
	}

	if _, exists := values["optional_without_default"]; exists {
		t.Fatalf("optional_without_default should not be set when no default exists")
	}
}

func TestResolveOptionsForNonInteractiveRequiredWithoutDefault(t *testing.T) {
	metadata := catalog.FeatureMetadata{
		Name: "test-feature",
		Options: []catalog.OptionMetadata{
			{
				Name:     "required_without_default",
				Type:     catalog.OptionTypeString,
				Required: true,
			},
		},
	}

	_, err := ResolveOptionsForNonInteractive(metadata)
	if err == nil {
		t.Fatal("expected error for required option without default, got nil")
	}
}

func TestResolveOptionsForNonInteractiveInvalidDefault(t *testing.T) {
	metadata := catalog.FeatureMetadata{
		Name: "test-feature",
		Options: []catalog.OptionMetadata{
			{
				Name:       "theme",
				Type:       catalog.OptionTypeEnum,
				Required:   true,
				Default:    "invalid",
				EnumValues: []string{"light", "dark"},
			},
		},
	}

	_, err := ResolveOptionsForNonInteractive(metadata)
	if err == nil {
		t.Fatal("expected error for invalid default value, got nil")
	}
}

func TestParseOptionOverrides(t *testing.T) {
	metadata := catalog.FeatureMetadata{
		Name: "test-feature",
		Options: []catalog.OptionMetadata{
			{Name: "theme", Type: catalog.OptionTypeEnum, EnumValues: []string{"light", "dark"}},
			{Name: "enabled", Type: catalog.OptionTypeBool},
			{Name: "timeout", Type: catalog.OptionTypeInt},
		},
	}

	overrides, err := ParseOptionOverrides(metadata, []string{"theme=dark", "enabled=true", "timeout=30"})
	if err != nil {
		t.Fatalf("ParseOptionOverrides() error = %v", err)
	}

	if overrides["theme"] != "dark" {
		t.Fatalf("theme = %v, want dark", overrides["theme"])
	}

	if overrides["enabled"] != true {
		t.Fatalf("enabled = %v, want true", overrides["enabled"])
	}

	if overrides["timeout"] != 30 {
		t.Fatalf("timeout = %v, want 30", overrides["timeout"])
	}
}

func TestParseOptionOverridesUnknownOption(t *testing.T) {
	metadata := catalog.FeatureMetadata{
		Name: "test-feature",
		Options: []catalog.OptionMetadata{
			{Name: "theme", Type: catalog.OptionTypeString},
		},
	}

	_, err := ParseOptionOverrides(metadata, []string{"unknown=value"})
	if err == nil {
		t.Fatal("expected unknown option error, got nil")
	}
}

func TestResolveOptionsForNonInteractiveWithOverrides(t *testing.T) {
	metadata := catalog.FeatureMetadata{
		Name: "test-feature",
		Options: []catalog.OptionMetadata{
			{Name: "required_theme", Type: catalog.OptionTypeString, Required: true},
			{Name: "optional_timeout", Type: catalog.OptionTypeInt, Required: false, Default: 10},
		},
	}

	overrides := map[string]any{"required_theme": "dark", "optional_timeout": 42}
	values, err := ResolveOptionsForNonInteractiveWithOverrides(metadata, overrides)
	if err != nil {
		t.Fatalf("ResolveOptionsForNonInteractiveWithOverrides() error = %v", err)
	}

	if values["required_theme"] != "dark" {
		t.Fatalf("required_theme = %v, want dark", values["required_theme"])
	}

	if values["optional_timeout"] != 42 {
		t.Fatalf("optional_timeout = %v, want 42", values["optional_timeout"])
	}
}
