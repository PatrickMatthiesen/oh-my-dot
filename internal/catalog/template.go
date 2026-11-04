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

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
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
	if err := fileops.WriteTextFileLF(featurePath, renderedContent, 0644); err != nil {
		return fmt.Errorf("failed to write feature template: %w", err)
	}

	return nil
}

// RenderFeatureTemplate renders a feature template with shell/option context.
func RenderFeatureTemplate(content, featureName, shellName string, optionValues map[string]any) (string, error) {
	context := buildTemplateContext(featureName, shellName, optionValues)
	if featureName == "oh-my-posh" && strings.IndexFunc(context.Theme, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-')
	}) >= 0 {
		return "", fmt.Errorf("theme must contain only letters, numbers, underscores and hyphens; use config_file for a custom path")
	}

	templateFuncs := template.FuncMap{
		"literal": func(value string) string { return shellLiteral(shellName, value) },
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

// shellLiteral quotes data without allowing shell interpolation. Generated
// expressions such as OMD_SHELL_ROOT remain separate from user-provided paths.
func shellLiteral(shellName, value string) string {
	switch shellName {
	case "powershell":
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	case "fish":
		return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(value) + "'"
	default:
		return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
	}
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
		context.DefaultConfigPath = fmt.Sprintf("$OMD_SHELL_ROOT/.local/cache/oh-my-posh.%s.omp.json", context.Theme)
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
