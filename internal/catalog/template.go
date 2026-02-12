package catalog

import (
	"bytes"
	"embed"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed features/**/*
var featureTemplates embed.FS

// posixCompatibleShells lists shells that can fall back to posix implementation
var posixCompatibleShells = map[string]bool{
	"bash": true,
	"zsh":  true,
	"sh":   true,
}

// GetFeatureTemplate retrieves the template content for a feature and shell
// Falls back to posix.sh for POSIX-compatible shells if shell-specific template doesn't exist
func GetFeatureTemplate(featureName, shellName string) (string, error) {
	ext := GetShellExtension(shellName)
	templatePath := fmt.Sprintf("features/%s/%s%s", featureName, shellName, ext)

	content, err := featureTemplates.ReadFile(templatePath)
	if err != nil {
		// If shell-specific template not found and shell is POSIX-compatible, try posix fallback
		if posixCompatibleShells[shellName] {
			posixPath := fmt.Sprintf("features/%s/posix.sh", featureName)
			content, posixErr := featureTemplates.ReadFile(posixPath)
			if posixErr == nil {
				return string(content), nil
			}
		}
		return "", fmt.Errorf("template not found for feature %s and shell %s: %w", featureName, shellName, err)
	}

	return string(content), nil
}

// HasFeatureTemplate checks if a template exists for a feature and shell
// Returns true if either shell-specific or posix fallback exists
func HasFeatureTemplate(featureName, shellName string) bool {
	ext := GetShellExtension(shellName)
	templatePath := fmt.Sprintf("features/%s/%s%s", featureName, shellName, ext)

	_, err := featureTemplates.ReadFile(templatePath)
	if err == nil {
		return true
	}

	// Check for posix fallback if shell is POSIX-compatible
	if posixCompatibleShells[shellName] {
		posixPath := fmt.Sprintf("features/%s/posix.sh", featureName)
		_, posixErr := featureTemplates.ReadFile(posixPath)
		return posixErr == nil
	}

	return false
}

// WriteFeatureTemplate writes a feature template to the user's repository
func WriteFeatureTemplate(repoPath, shellName, featureName string, optionValues map[string]any) error {
	content, err := GetFeatureTemplate(featureName, shellName)
	if err != nil {
		return err
	}

	renderedContent, err := RenderFeatureTemplate(content, featureName, shellName, optionValues)
	if err != nil {
		return fmt.Errorf("failed to render feature template: %w", err)
	}

	ext := GetShellExtension(shellName)
	featurePath := filepath.Join(repoPath, "omd-shells", shellName, "features", featureName+ext)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(featurePath), 0755); err != nil {
		return fmt.Errorf("failed to create feature directory: %w", err)
	}

	// Write template
	if err := os.WriteFile(featurePath, []byte(renderedContent), 0644); err != nil {
		return fmt.Errorf("failed to write feature template: %w", err)
	}

	return nil
}

// RenderFeatureTemplate renders a feature template with shell/option context.
func RenderFeatureTemplate(content, featureName, shellName string, optionValues map[string]any) (string, error) {
	context := buildTemplateContext(featureName, shellName, optionValues)

	templateFuncs := template.FuncMap{
		"hasOption": func(key string) bool {
			_, ok := context.Options[key]
			return ok
		},
		"option": func(key string) any {
			return context.Options[key]
		},
	}

	tmpl, err := template.New(featureName).Funcs(templateFuncs).Option("missingkey=zero").Parse(content)
	if err != nil {
		return "", err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, context); err != nil {
		return "", err
	}

	return rendered.String(), nil
}

type featureTemplateContext struct {
	FeatureName       string
	ShellName         string
	Options           map[string]any
	Theme             string
	ThemeURL          string
	ConfigFile        string
	DefaultConfigPath string
	AutoUpgrade       bool
}

func buildTemplateContext(featureName, shellName string, optionValues map[string]any) featureTemplateContext {
	context := featureTemplateContext{
		FeatureName: featureName,
		ShellName:   shellName,
		Options:     map[string]any{},
	}

	maps.Copy(context.Options, optionValues)

	if featureName == "oh-my-posh" {
		context.Theme = getOptionString(optionValues, "theme", "jandedobbeleer")
		context.ThemeURL = fmt.Sprintf("https://raw.githubusercontent.com/JanDeDobbeleer/oh-my-posh/main/themes/%s.omp.json", context.Theme)
		context.ConfigFile = getOptionString(optionValues, "config_file", "")
		context.DefaultConfigPath = "$OMD_SHELL_ROOT/features/oh-my-posh.omp.json"
		context.AutoUpgrade = getOptionBool(optionValues, "auto_upgrade", false)
	}

	return context
}

func getOptionString(optionValues map[string]any, key, defaultValue string) string {
	rawValue, ok := optionValues[key]
	if !ok || rawValue == nil {
		return defaultValue
	}

	value, ok := rawValue.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	return value
}

func getOptionBool(optionValues map[string]any, key string, defaultValue bool) bool {
	rawValue, ok := optionValues[key]
	if !ok || rawValue == nil {
		return defaultValue
	}

	value, ok := rawValue.(bool)
	if !ok {
		return defaultValue
	}

	return value
}

// GetShellExtension returns the file extension for a given shell
func GetShellExtension(shellName string) string {
	switch shellName {
	case "fish":
		return ".fish"
	case "powershell":
		return ".ps1"
	default:
		return ".sh"
	}
}
