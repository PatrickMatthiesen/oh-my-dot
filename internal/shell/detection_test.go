package shell

import (
	"os"
	"runtime"
	"testing"
)

func TestDetectCurrentShell(t *testing.T) {
	// Save original environment
	originalShell := os.Getenv("SHELL")
	original0 := os.Getenv("0")
	originalPSModulePath := os.Getenv("PSModulePath")
	
	// Restore environment after test
	defer func() {
		os.Setenv("SHELL", originalShell)
		os.Setenv("0", original0)
		os.Setenv("PSModulePath", originalPSModulePath)
	}()

	tests := []struct {
		name           string
		setupEnv       func()
		expectedShell  string
		shouldError    bool
		skipOnOS       string // Skip test on specific OS
	}{
		{
			name: "detect bash from SHELL",
			setupEnv: func() {
				os.Setenv("SHELL", "/bin/bash")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "bash",
			shouldError:   false,
		},
		{
			name: "detect zsh from SHELL",
			setupEnv: func() {
				os.Setenv("SHELL", "/usr/local/bin/zsh")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "zsh",
			shouldError:   false,
		},
		{
			name: "detect fish from SHELL",
			setupEnv: func() {
				os.Setenv("SHELL", "/usr/bin/fish")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "fish",
			shouldError:   false,
		},
		{
			name: "detect PowerShell from PSModulePath on Windows",
			setupEnv: func() {
				os.Setenv("SHELL", "")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "C:\\Program Files\\PowerShell\\Modules")
			},
			expectedShell: "powershell",
			shouldError:   false,
			skipOnOS:      "linux", // Only relevant on Windows
		},
		{
			name: "detect from $0 when SHELL not set",
			setupEnv: func() {
				os.Setenv("SHELL", "")
				os.Setenv("0", "/bin/bash")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "bash",
			shouldError:   false,
		},
		{
			name: "normalize pwsh to powershell",
			setupEnv: func() {
				os.Setenv("SHELL", "/usr/bin/pwsh")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "powershell",
			shouldError:   false,
		},
		{
			name: "normalize sh to posix",
			setupEnv: func() {
				os.Setenv("SHELL", "/bin/sh")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "posix",
			shouldError:   false,
		},
		{
			name: "error when no shell detected",
			setupEnv: func() {
				os.Setenv("SHELL", "")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "",
			shouldError:   true,
		},
		{
			name: "ignore unsupported shell",
			setupEnv: func() {
				os.Setenv("SHELL", "/bin/unsupported-shell")
				os.Setenv("0", "")
				os.Setenv("PSModulePath", "")
			},
			expectedShell: "",
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if specified
			if tt.skipOnOS != "" && runtime.GOOS == tt.skipOnOS {
				t.Skipf("Skipping test on %s", runtime.GOOS)
			}

			// Setup test environment
			tt.setupEnv()

			// Run detection
			shell, err := DetectCurrentShell()

			// Verify result
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if shell != tt.expectedShell {
					t.Errorf("Expected shell %q, got %q", tt.expectedShell, shell)
				}
			}
		})
	}
}

func TestNormalizeShellName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/bin/bash", "bash"},
		{"/usr/local/bin/zsh", "zsh"},
		{"bash.exe", "bash"},
		{"powershell.exe", "powershell"},
		{"pwsh", "powershell"},
		{"pwsh.exe", "powershell"},
		{"sh", "posix"},
		{"dash", "posix"},
		{"fish", "fish"},
		{"BASH", "bash"}, // Test case insensitivity
		{"ZSH.exe", "zsh"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeShellName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeShellName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
