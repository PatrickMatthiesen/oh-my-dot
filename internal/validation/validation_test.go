package validation

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/spf13/viper"
)

func TestValidateString(t *testing.T) {
	tests := []struct {
		name      string
		opt       catalog.OptionMetadata
		value     any
		wantError bool
	}{
		{
			name:      "valid string",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "hello world",
			wantError: false,
		},
		{
			name:      "empty string",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "",
			wantError: false,
		},
		{
			name:      "string with null byte",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "hello\x00world",
			wantError: true,
		},
		{
			name:      "command substitution",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "$(rm -rf /)",
			wantError: true,
		},
		{
			name:      "backtick command substitution",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "`whoami`",
			wantError: true,
		},
		{
			name:      "pipe separator",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "cat file | grep pattern",
			wantError: true,
		},
		{
			name:      "semicolon separator",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "echo hello; rm file",
			wantError: true,
		},
		{
			name:      "path traversal",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     "../../../etc/passwd",
			wantError: true,
		},
		{
			name:      "not a string",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeString},
			value:     123,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateString(tt.opt, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateString() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateInt(t *testing.T) {
	min := 0
	max := 100

	tests := []struct {
		name      string
		opt       catalog.OptionMetadata
		value     any
		wantError bool
	}{
		{
			name:      "valid int",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeInt},
			value:     50,
			wantError: false,
		},
		{
			name:      "valid int64",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeInt},
			value:     int64(50),
			wantError: false,
		},
		{
			name:      "valid string int",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeInt},
			value:     "50",
			wantError: false,
		},
		{
			name:      "below minimum",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeInt, IntMin: &min},
			value:     -10,
			wantError: true,
		},
		{
			name:      "above maximum",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeInt, IntMax: &max},
			value:     150,
			wantError: true,
		},
		{
			name:      "invalid string",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeInt},
			value:     "not a number",
			wantError: true,
		},
		{
			name:      "wrong type",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeInt},
			value:     true,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInt(tt.opt, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateInt() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateBool(t *testing.T) {
	tests := []struct {
		name      string
		opt       catalog.OptionMetadata
		value     any
		wantError bool
	}{
		{
			name:      "true bool",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     true,
			wantError: false,
		},
		{
			name:      "false bool",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     false,
			wantError: false,
		},
		{
			name:      "string true",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     "true",
			wantError: false,
		},
		{
			name:      "string false",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     "false",
			wantError: false,
		},
		{
			name:      "string 1",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     "1",
			wantError: false,
		},
		{
			name:      "string 0",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     "0",
			wantError: false,
		},
		{
			name:      "string yes",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     "yes",
			wantError: false,
		},
		{
			name:      "string no",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     "no",
			wantError: false,
		},
		{
			name:      "invalid string",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     "maybe",
			wantError: true,
		},
		{
			name:      "wrong type",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeBool},
			value:     123,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBool(tt.opt, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateBool() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	validValues := []string{"option1", "option2", "option3"}

	tests := []struct {
		name      string
		opt       catalog.OptionMetadata
		value     any
		wantError bool
	}{
		{
			name:      "valid enum value",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeEnum, EnumValues: validValues},
			value:     "option1",
			wantError: false,
		},
		{
			name:      "invalid enum value",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeEnum, EnumValues: validValues},
			value:     "invalid",
			wantError: true,
		},
		{
			name:      "empty enum values",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeEnum, EnumValues: []string{}},
			value:     "option1",
			wantError: true,
		},
		{
			name:      "wrong type",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeEnum, EnumValues: validValues},
			value:     123,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnum(tt.opt, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateEnum() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name      string
		opt       catalog.OptionMetadata
		value     any
		wantError bool
	}{
		{
			name:      "valid file path without existence check",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     filepath.Join(tmpDir, "nonexistent.txt"),
			wantError: false,
		},
		{
			name:      "valid existing file",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: true, FileOnly: true},
			value:     testFile,
			wantError: false,
		},
		{
			name:      "directory when file expected",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: true, FileOnly: true},
			value:     testDir,
			wantError: true,
		},
		{
			name:      "nonexistent file when must exist",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: true},
			value:     filepath.Join(tmpDir, "nonexistent.txt"),
			wantError: true,
		},
		{
			name:      "wrong type",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile},
			value:     123,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFile(tt.opt, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateFile() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no dangerous characters",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "with dollar sign",
			input:    "test$var",
			expected: "test" + `\` + "$var",
		},
		{
			name:     "with backtick",
			input:    "echo `whoami`",
			expected: "echo " + `\` + "`whoami" + `\` + "`",
		},
		{
			name:     "with null byte",
			input:    "hello\x00world",
			expected: "helloworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		want      bool
		wantError bool
	}{
		{
			name:      "bool true",
			value:     true,
			want:      true,
			wantError: false,
		},
		{
			name:      "bool false",
			value:     false,
			want:      false,
			wantError: false,
		},
		{
			name:      "string true",
			value:     "true",
			want:      true,
			wantError: false,
		},
		{
			name:      "string yes",
			value:     "yes",
			want:      true,
			wantError: false,
		},
		{
			name:      "string 1",
			value:     "1",
			want:      true,
			wantError: false,
		},
		{
			name:      "string false",
			value:     "false",
			want:      false,
			wantError: false,
		},
		{
			name:      "string no",
			value:     "no",
			want:      false,
			wantError: false,
		},
		{
			name:      "string 0",
			value:     "0",
			want:      false,
			wantError: false,
		},
		{
			name:      "invalid string",
			value:     "maybe",
			want:      false,
			wantError: true,
		},
		{
			name:      "invalid type",
			value:     123,
			want:      false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBool(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseBool() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("ParseBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "tilde expansion",
			input:     "~/test.txt",
			wantError: false,
		},
		{
			name:      "absolute path",
			input:     filepath.Join(homeDir, "test.txt"),
			wantError: false,
		},
		{
			name:      "relative path",
			input:     "test.txt",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ExpandPath() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && !filepath.IsAbs(result) {
				t.Errorf("ExpandPath() did not return absolute path: %v", result)
			}
		})
	}
}

func TestValidateFilePathTraversal(t *testing.T) {
	// Save original config value and restore after test
	originalValue := viper.GetBool("restrict-paths-to-home")
	defer viper.Set("restrict-paths-to-home", originalValue)

	// Disable home directory restriction for these tests
	viper.Set("restrict-paths-to-home", false)

	// Create a temporary test directory inside home for valid test cases
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		opt       catalog.OptionMetadata
		value     any
		wantError bool
		errorMsg  string
	}{
		{
			name:      "path traversal with ../../../",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     "../../../etc/passwd",
			wantError: true,
			errorMsg:  "path traversal detected",
		},
		{
			name:      "path traversal with ..",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     "..",
			wantError: true,
			errorMsg:  "path traversal detected",
		},
		{
			name:      "path traversal with ../ at start",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     "../config.txt",
			wantError: true,
			errorMsg:  "path traversal detected",
		},
		{
			name:      "path traversal in middle of path",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     "some/path/../../../escape",
			wantError: true,
			errorMsg:  "path traversal detected",
		},
		{
			name:      "legitimate relative path",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     filepath.Join(tmpDir, "config.txt"),
			wantError: false,
		},
		{
			name:      "legitimate absolute path",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     testFile,
			wantError: false,
		},
		{
			name:      "tilde expansion",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     "~/test.txt",
			wantError: false,
		},
	}

	// Windows-specific test case
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name      string
			opt       catalog.OptionMetadata
			value     any
			wantError bool
			errorMsg  string
		}{
			name:      "windows relative path traversal",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false},
			value:     "..\\..\\..\\Windows\\System32",
			wantError: true,
			errorMsg:  "path traversal detected",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFile(tt.opt, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateFile() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && tt.errorMsg != "" {
				if err.Error() != tt.errorMsg {
					t.Errorf("validateFile() error message = %q, want %q", err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

func TestValidatePathTraversal(t *testing.T) {
	// Save original config value and restore after test
	originalValue := viper.GetBool("restrict-paths-to-home")
	defer viper.Set("restrict-paths-to-home", originalValue)

	// Disable home directory restriction for these tests
	viper.Set("restrict-paths-to-home", false)

	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		opt       catalog.OptionMetadata
		value     any
		wantError bool
		errorMsg  string
	}{
		{
			name:      "path traversal with ../../../",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypePath, PathMustExist: false},
			value:     "../../../etc",
			wantError: true,
			errorMsg:  "path traversal detected",
		},
		{
			name:      "path traversal with ..",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypePath, PathMustExist: false},
			value:     "..",
			wantError: true,
			errorMsg:  "path traversal detected",
		},
		{
			name:      "legitimate directory path",
			opt:       catalog.OptionMetadata{Type: catalog.OptionTypePath, PathMustExist: false},
			value:     tmpDir,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.opt, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validatePath() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && tt.errorMsg != "" {
				if err.Error() != tt.errorMsg {
					t.Errorf("validatePath() error message = %q, want %q", err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

func TestRestrictPathsToHome(t *testing.T) {
	// Save original config value and restore after test
	originalValue := viper.GetBool("restrict-paths-to-home")
	defer viper.Set("restrict-paths-to-home", originalValue)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	// Create a test file within home directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name               string
		restrictToHome     bool
		value              string
		wantError          bool
		expectedErrorMatch string
	}{
		{
			name:           "unrestricted mode - allows home path",
			restrictToHome: false,
			value:          filepath.Join(homeDir, "test.txt"),
			wantError:      false,
		},
		{
			name:           "unrestricted mode - allows system path",
			restrictToHome: false,
			value:          testFile, // This might be outside home in temp
			wantError:      false,
		},
		{
			name:           "restricted mode - allows home path",
			restrictToHome: true,
			value:          filepath.Join(homeDir, "test.txt"),
			wantError:      false,
		},
		{
			name:               "restricted mode - blocks system path",
			restrictToHome:     true,
			value:              "/etc/profile", // Unix path outside home
			wantError:          true,
			expectedErrorMatch: "restrict-paths-to-home is enabled",
		},
		{
			name:           "tilde expansion works in both modes",
			restrictToHome: false,
			value:          "~/test.txt",
			wantError:      false,
		},
	}

	// Add Windows-specific test
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name               string
			restrictToHome     bool
			value              string
			wantError          bool
			expectedErrorMatch string
		}{
			name:               "restricted mode - blocks Windows system path",
			restrictToHome:     true,
			value:              "C:\\Windows\\System32\\config.txt",
			wantError:          true,
			expectedErrorMatch: "restrict-paths-to-home is enabled",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("restrict-paths-to-home", tt.restrictToHome)

			opt := catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false}
			err := validateFile(opt, tt.value)

			if (err != nil) != tt.wantError {
				t.Errorf("validateFile() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err != nil && tt.expectedErrorMatch != "" {
				if !contains(err.Error(), tt.expectedErrorMatch) {
					t.Errorf("validateFile() error = %q, expected to contain %q", err.Error(), tt.expectedErrorMatch)
				}
			}
		})
	}
}

func TestRestrictPathsToHomeForValidatePath(t *testing.T) {
	// Save original config value and restore after test
	originalValue := viper.GetBool("restrict-paths-to-home")
	defer viper.Set("restrict-paths-to-home", originalValue)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name               string
		restrictToHome     bool
		value              string
		wantError          bool
		expectedErrorMatch string
	}{
		{
			name:           "unrestricted mode - allows home path",
			restrictToHome: false,
			value:          homeDir,
			wantError:      false,
		},
		{
			name:           "restricted mode - allows home path",
			restrictToHome: true,
			value:          filepath.Join(homeDir, "Documents"),
			wantError:      false,
		},
		{
			name:               "restricted mode - blocks system path",
			restrictToHome:     true,
			value:              "/usr/local", // Unix path outside home
			wantError:          true,
			expectedErrorMatch: "restrict-paths-to-home is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("restrict-paths-to-home", tt.restrictToHome)

			opt := catalog.OptionMetadata{Type: catalog.OptionTypePath, PathMustExist: false}
			err := validatePath(opt, tt.value)

			if (err != nil) != tt.wantError {
				t.Errorf("validatePath() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err != nil && tt.expectedErrorMatch != "" {
				if !contains(err.Error(), tt.expectedErrorMatch) {
					t.Errorf("validatePath() error = %q, expected to contain %q", err.Error(), tt.expectedErrorMatch)
				}
			}
		})
	}
}

func TestRestrictPathsToHomeBlocksPrefixBypass(t *testing.T) {
	originalValue := viper.GetBool("restrict-paths-to-home")
	defer viper.Set("restrict-paths-to-home", originalValue)

	viper.Set("restrict-paths-to-home", true)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	bypassPath := homeDir + "_outside"

	opt := catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: false}
	err = validateFile(opt, bypassPath)
	if err == nil {
		t.Fatal("expected prefix bypass path to be rejected")
	}

	if !contains(err.Error(), "restrict-paths-to-home is enabled") {
		t.Fatalf("expected restrict-paths-to-home error, got: %v", err)
	}
}

func TestRestrictPathsToHomeBlocksSymlinkTargetOutsideHome(t *testing.T) {
	originalValue := viper.GetBool("restrict-paths-to-home")
	defer viper.Set("restrict-paths-to-home", originalValue)

	viper.Set("restrict-paths-to-home", true)

	var targetPath string
	if runtime.GOOS == "windows" {
		targetPath = `C:\Windows\System32\drivers\etc\hosts`
	} else {
		targetPath = "/etc/hosts"
	}

	if _, err := os.Stat(targetPath); err != nil {
		t.Skipf("target path for symlink test not available: %v", err)
	}

	linkPath := filepath.Join(t.TempDir(), "outside-link")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlink creation not supported in this environment: %v", err)
	}

	opt := catalog.OptionMetadata{Type: catalog.OptionTypeFile, PathMustExist: true, FileOnly: true}
	err := validateFile(opt, linkPath)
	if err == nil {
		t.Fatal("expected symlink target outside home to be rejected")
	}

	if !contains(err.Error(), "restrict-paths-to-home is enabled") {
		t.Fatalf("expected restrict-paths-to-home error, got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
