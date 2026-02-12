package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/spf13/viper"
)

// shellMetacharacters are characters that have special meaning in shells
var shellMetacharacters = []string{
	"$", "`", "(", ")", "{", "}", "[", "]", "|", "&", ";", "<", ">", "\\", "\"", "'", "\n", "\r", "\t", "*", "?",
}

// suspiciousPatterns are patterns that might indicate injection attempts
var suspiciousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\$\([^)]*\)`), // Command substitution $(...)
	regexp.MustCompile("`[^`]*`"),     // Command substitution `...`
	regexp.MustCompile(`[;&|]`),       // Command separators
	regexp.MustCompile(`\.\./`),       // Path traversal
}

// ValidateOption validates a user-provided value against option metadata
func ValidateOption(opt catalog.OptionMetadata, value any) error {
	// Check required
	if opt.Required && value == nil {
		return fmt.Errorf("required option '%s' cannot be empty", opt.DisplayName)
	}

	// If optional and nil, return default
	if !opt.Required && value == nil {
		return nil
	}

	// Type-specific validation
	switch opt.Type {
	case catalog.OptionTypeString:
		return validateString(opt, value)
	case catalog.OptionTypeInt:
		return validateInt(opt, value)
	case catalog.OptionTypeBool:
		return validateBool(opt, value)
	case catalog.OptionTypeEnum:
		return validateEnum(opt, value)
	case catalog.OptionTypeFile:
		return validateFile(opt, value)
	case catalog.OptionTypePath:
		return validatePath(opt, value)
	default:
		return fmt.Errorf("unsupported option type: %s", opt.Type)
	}
}

// validateString validates string input with security checks
func validateString(opt catalog.OptionMetadata, value any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}

	// Check length
	const maxLength = 500
	if len(str) > maxLength {
		return fmt.Errorf("string too long (max %d characters)", maxLength)
	}

	// Check for null bytes
	if strings.ContainsRune(str, '\x00') {
		return fmt.Errorf("string contains null byte")
	}

	// Check for suspicious patterns
	for _, pattern := range suspiciousPatterns {
		if pattern.MatchString(str) {
			return fmt.Errorf("string contains potentially dangerous pattern")
		}
	}

	// Run custom validator if provided
	if opt.Validator != nil {
		return opt.Validator(value)
	}

	return nil
}

// validateInt validates integer input with range checks
func validateInt(opt catalog.OptionMetadata, value any) error {
	var intVal int64

	// Handle various numeric types
	switch v := value.(type) {
	case int:
		intVal = int64(v)
	case int64:
		intVal = v
	case float64:
		intVal = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer: %w", err)
		}
		intVal = parsed
	default:
		return fmt.Errorf("expected integer, got %T", value)
	}

	// Check min
	if opt.IntMin != nil && intVal < int64(*opt.IntMin) {
		return fmt.Errorf("value %d is below minimum %d", intVal, *opt.IntMin)
	}

	// Check max
	if opt.IntMax != nil && intVal > int64(*opt.IntMax) {
		return fmt.Errorf("value %d is above maximum %d", intVal, *opt.IntMax)
	}

	// Run custom validator if provided
	if opt.Validator != nil {
		return opt.Validator(value)
	}

	return nil
}

// validateBool validates boolean input
func validateBool(opt catalog.OptionMetadata, value any) error {
	switch v := value.(type) {
	case bool:
		_ = v
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		validValues := []string{"true", "false", "1", "0", "yes", "no", "y", "n"}
		for _, valid := range validValues {
			if lower == valid {
				if opt.Validator != nil {
					return opt.Validator(value)
				}
				return nil
			}
		}
		return fmt.Errorf("invalid boolean value: %s (expected: true/false, 1/0, yes/no, y/n)", v)
	default:
		return fmt.Errorf("expected boolean, got %T", value)
	}

	if opt.Validator != nil {
		return opt.Validator(value)
	}

	return nil
}

// validateEnum validates enum input against allowed values
func validateEnum(opt catalog.OptionMetadata, value any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}

	if len(opt.EnumValues) == 0 {
		return fmt.Errorf("enum option has no valid values defined")
	}

	for _, validValue := range opt.EnumValues {
		if str == validValue {
			return nil
		}
	}

	return fmt.Errorf("invalid value '%s', must be one of: %v", str, opt.EnumValues)
}

// validateFile validates file path input with security checks
func validateFile(opt catalog.OptionMetadata, value any) error {
	pathStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string path, got %T", value)
	}

	// Expand home directory
	if strings.HasPrefix(pathStr, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand home directory: %w", err)
		}
		pathStr = filepath.Join(homeDir, pathStr[1:])
	}

	if hasPathTraversal(pathStr) {
		return fmt.Errorf("path traversal detected")
	}

	// Clean the path first (resolves .., ., removes duplicate separators)
	cleanPath := filepath.Clean(pathStr)

	// Now safe to convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	effectivePath := absPath

	// Verify path exists if required
	if opt.PathMustExist {
		info, err := os.Lstat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", absPath)
			}
			return fmt.Errorf("cannot access path: %w", err)
		}

		// Check if file is a symlink and resolve it
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(absPath)
			if err != nil {
				return fmt.Errorf("broken symlink: %w", err)
			}

			// Validate the symlink target
			targetInfo, err := os.Stat(target)
			if err != nil {
				return fmt.Errorf("symlink target invalid: %w", err)
			}

			if opt.FileOnly && targetInfo.IsDir() {
				return fmt.Errorf("symlink target is a directory, expected a file: %s", target)
			}

			effectivePath = target
		} else if opt.FileOnly && info.IsDir() {
			return fmt.Errorf("path is a directory, expected a file: %s", absPath)
		}
	}

	// Security check: optionally restrict paths to home directory
	if viper.GetBool("restrict-paths-to-home") {
		if err := validatePathWithinHome(effectivePath); err != nil {
			return err
		}
	}

	// Run custom validator if provided
	if opt.Validator != nil {
		return opt.Validator(value)
	}

	return nil
}

// validatePath validates path input (similar to file but allows directories)
func validatePath(opt catalog.OptionMetadata, value any) error {
	pathStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string path, got %T", value)
	}

	// Expand home directory
	if strings.HasPrefix(pathStr, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand home directory: %w", err)
		}
		pathStr = filepath.Join(homeDir, pathStr[1:])
	}

	if hasPathTraversal(pathStr) {
		return fmt.Errorf("path traversal detected")
	}

	// Clean the path first (resolves .., ., removes duplicate separators)
	cleanPath := filepath.Clean(pathStr)

	// Now safe to convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	effectivePath := absPath

	// Verify path exists if required
	if opt.PathMustExist {
		info, err := os.Lstat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("path does not exist: %s", absPath)
			}
			return fmt.Errorf("cannot access path: %w", err)
		}

		// Check if path is a symlink and resolve it
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(absPath)
			if err != nil {
				return fmt.Errorf("broken symlink: %w", err)
			}

			// Validate the symlink target
			_, err = os.Stat(target)
			if err != nil {
				return fmt.Errorf("symlink target invalid: %w", err)
			}

			effectivePath = target
		}
	}

	// Security check: optionally restrict paths to home directory
	if viper.GetBool("restrict-paths-to-home") {
		if err := validatePathWithinHome(effectivePath); err != nil {
			return err
		}
	}

	// Run custom validator if provided
	if opt.Validator != nil {
		return opt.Validator(value)
	}

	return nil
}

// SanitizeString removes or escapes dangerous characters from string input
func SanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Escape backslash first to prevent double-escaping
	input = strings.ReplaceAll(input, "\\", "\\\\")

	// Escape other shell metacharacters (excluding backslash which was already handled)
	metacharsToEscape := []string{
		"$", "`", "(", ")", "{", "}", "[", "]", "|", "&", ";", "<", ">", "\"", "'", "\n", "\r", "\t", "*", "?",
	}
	for _, char := range metacharsToEscape {
		input = strings.ReplaceAll(input, char, "\\"+char)
	}

	return input
}

// ParseBool converts various boolean representations to bool
func ParseBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "true", "1", "yes", "y":
			return true, nil
		case "false", "0", "no", "n":
			return false, nil
		default:
			return false, fmt.Errorf("invalid boolean value: %s", v)
		}
	default:
		return false, fmt.Errorf("cannot convert %T to boolean", value)
	}
}

// ExpandPath expands ~ to home directory and resolves to absolute path
func ExpandPath(pathStr string) (string, error) {
	// Expand home directory
	if strings.HasPrefix(pathStr, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand home directory: %w", err)
		}
		pathStr = filepath.Join(homeDir, pathStr[1:])
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(pathStr)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}

func hasPathTraversal(pathStr string) bool {
	normalized := strings.ReplaceAll(pathStr, "\\", "/")
	segments := strings.Split(normalized, "/")

	for _, segment := range segments {
		if segment == ".." {
			return true
		}
	}

	return false
}

func validatePathWithinHome(path string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine user home directory: %w", err)
	}

	if !isWithinBasePath(path, homeDir) {
		return fmt.Errorf("path must be within home directory (restrict-paths-to-home is enabled): %s", path)
	}

	return nil
}

func isWithinBasePath(path, base string) bool {
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(base)

	rel, err := filepath.Rel(cleanBase, cleanPath)
	if err != nil {
		return false
	}

	if rel == "." {
		return true
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
