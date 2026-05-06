package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasFeatureTemplate(t *testing.T) {
	tests := []struct {
		name        string
		featureName string
		shellName   string
		want        bool
	}{
		{"ssh-agent bash (fallback to posix)", "ssh-agent", "bash", true},
		{"ssh-agent zsh (fallback to posix)", "ssh-agent", "zsh", true},
		{"ssh-agent fish", "ssh-agent", "fish", true},
		{"ssh-agent posix", "ssh-agent", "posix", true},
		{"ssh-agent sh (fallback to posix)", "ssh-agent", "sh", true},
		{"homebrew-path bash (fallback to posix)", "homebrew-path", "bash", true},
		{"homebrew-path zsh (fallback to posix)", "homebrew-path", "zsh", true},
		{"homebrew-path fish", "homebrew-path", "fish", true},
		{"homebrew-path posix", "homebrew-path", "posix", true},
		{"powershell-prompt powershell", "powershell-prompt", "powershell", true},
		{"powershell-aliases powershell", "powershell-aliases", "powershell", true},
		{"powershell-psreadline powershell", "powershell-psreadline", "powershell", true},
		{"posh-git powershell", "posh-git", "powershell", true},
		{"terminal-icons powershell", "terminal-icons", "powershell", true},
		{"dotnet-completion powershell", "dotnet-completion", "powershell", true},
		{"winget-completion powershell", "winget-completion", "powershell", true},
		{"winget-command-not-found powershell", "winget-command-not-found", "powershell", true},
		{"oh-my-posh bash", "oh-my-posh", "bash", true},
		{"oh-my-posh zsh", "oh-my-posh", "zsh", true},
		{"oh-my-posh fish", "oh-my-posh", "fish", true},
		{"oh-my-posh powershell", "oh-my-posh", "powershell", true},
		{"non-existent feature", "non-existent", "bash", false},
		{"ssh-agent powershell (no fallback)", "ssh-agent", "powershell", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasFeatureTemplate(tt.featureName, tt.shellName)
			if got != tt.want {
				t.Errorf("HasFeatureTemplate(%q, %q) = %v, want %v", tt.featureName, tt.shellName, got, tt.want)
			}
		})
	}
}

func TestGetFeatureTemplate(t *testing.T) {
	tests := []struct {
		name        string
		featureName string
		shellName   string
		wantError   bool
		contains    string
	}{
		{
			name:        "ssh-agent bash (fallback to posix)",
			featureName: "ssh-agent",
			shellName:   "bash",
			wantError:   false,
			contains:    "SSH_AUTH_SOCK",
		},
		{
			name:        "ssh-agent zsh (fallback to posix)",
			featureName: "ssh-agent",
			shellName:   "zsh",
			wantError:   false,
			contains:    "SSH_AUTH_SOCK",
		},
		{
			name:        "ssh-agent fish",
			featureName: "ssh-agent",
			shellName:   "fish",
			wantError:   false,
			contains:    "SSH_AUTH_SOCK",
		},
		{
			name:        "homebrew-path bash (fallback to posix)",
			featureName: "homebrew-path",
			shellName:   "bash",
			wantError:   false,
			contains:    "linuxbrew",
		},
		{
			name:        "homebrew-path zsh (fallback to posix)",
			featureName: "homebrew-path",
			shellName:   "zsh",
			wantError:   false,
			contains:    "linuxbrew",
		},
		{
			name:        "homebrew-path fish",
			featureName: "homebrew-path",
			shellName:   "fish",
			wantError:   false,
			contains:    "linuxbrew",
		},
		{
			name:        "non-existent feature",
			featureName: "non-existent",
			shellName:   "bash",
			wantError:   true,
			contains:    "",
		},
		{
			name:        "ssh-agent powershell (no fallback)",
			featureName: "ssh-agent",
			shellName:   "powershell",
			wantError:   true,
			contains:    "",
		},
		{
			name:        "powershell-prompt powershell",
			featureName: "powershell-prompt",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Get-GitBranch",
		},
		{
			name:        "powershell-aliases powershell",
			featureName: "powershell-aliases",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Set-Alias",
		},
		{
			name:        "powershell-aliases powershell head",
			featureName: "powershell-aliases",
			shellName:   "powershell",
			wantError:   false,
			contains:    "function head",
		},
		{
			name:        "powershell-aliases powershell tail",
			featureName: "powershell-aliases",
			shellName:   "powershell",
			wantError:   false,
			contains:    "function tail",
		},
		{
			name:        "powershell-aliases powershell less",
			featureName: "powershell-aliases",
			shellName:   "powershell",
			wantError:   false,
			contains:    "function less",
		},
		{
			name:        "powershell-aliases powershell find",
			featureName: "powershell-aliases",
			shellName:   "powershell",
			wantError:   false,
			contains:    "function find",
		},
		{
			name:        "powershell-aliases powershell touch",
			featureName: "powershell-aliases",
			shellName:   "powershell",
			wantError:   false,
			contains:    "function touch",
		},
		{
			name:        "powershell-psreadline powershell",
			featureName: "powershell-psreadline",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Set-PSReadLineKeyHandler",
		},
		{
			name:        "posh-git powershell",
			featureName: "posh-git",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Import-Module posh-git",
		},
		{
			name:        "terminal-icons powershell",
			featureName: "terminal-icons",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Import-Module Terminal-Icons",
		},
		{
			name:        "dotnet-completion powershell",
			featureName: "dotnet-completion",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Register-ArgumentCompleter",
		},
		{
			name:        "winget-completion powershell",
			featureName: "winget-completion",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Register-ArgumentCompleter",
		},
		{
			name:        "winget-command-not-found powershell",
			featureName: "winget-command-not-found",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Import-Module Microsoft.WinGet.CommandNotFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GetFeatureTemplate(tt.featureName, tt.shellName)
			if (err != nil) != tt.wantError {
				t.Errorf("GetFeatureTemplate(%q, %q) error = %v, wantError %v", tt.featureName, tt.shellName, err, tt.wantError)
				return
			}
			if !tt.wantError && content == "" {
				t.Errorf("GetFeatureTemplate(%q, %q) returned empty content", tt.featureName, tt.shellName)
			}
			if tt.contains != "" && !strings.Contains(content, tt.contains) {
				t.Errorf("GetFeatureTemplate(%q, %q) content does not contain %q", tt.featureName, tt.shellName, tt.contains)
			}
		})
	}
}

func TestPowerShellAliasesTailSupportsPipelineInput(t *testing.T) {
	pwshPath, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh is required for this test")
	}

	content, err := GetFeatureTemplate("powershell-aliases", "powershell")
	if err != nil {
		t.Fatalf("GetFeatureTemplate() error = %v", err)
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "powershell-aliases.ps1")
	inputPath := filepath.Join(tempDir, "input.txt")

	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile(scriptPath) error = %v", err)
	}

	inputContent := "one\ntwo\nthree\nfour\n"
	if err := os.WriteFile(inputPath, []byte(inputContent), 0644); err != nil {
		t.Fatalf("os.WriteFile(inputPath) error = %v", err)
	}

	t.Run("pipeline", func(t *testing.T) {
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, "Get-Content "+quotePowerShellPath(inputPath)+" | tail -n 2")
		if output != "three\nfour" {
			t.Fatalf("pipeline tail output = %q, want %q", output, "three\nfour")
		}
	})

	t.Run("path", func(t *testing.T) {
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, "tail -n 2 "+quotePowerShellPath(inputPath))
		if output != "three\nfour" {
			t.Fatalf("path tail output = %q, want %q", output, "three\nfour")
		}
	})
}

func TestPowerShellAliasesHeadSupportsPipelineInput(t *testing.T) {
	pwshPath, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh is required for this test")
	}

	content, err := GetFeatureTemplate("powershell-aliases", "powershell")
	if err != nil {
		t.Fatalf("GetFeatureTemplate() error = %v", err)
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "powershell-aliases.ps1")
	inputPath := filepath.Join(tempDir, "input.txt")

	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile(scriptPath) error = %v", err)
	}

	inputContent := "one\ntwo\nthree\nfour\n"
	if err := os.WriteFile(inputPath, []byte(inputContent), 0644); err != nil {
		t.Fatalf("os.WriteFile(inputPath) error = %v", err)
	}

	t.Run("pipeline", func(t *testing.T) {
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, "Get-Content "+quotePowerShellPath(inputPath)+" | head -n 2")
		if output != "one\ntwo" {
			t.Fatalf("pipeline head output = %q, want %q", output, "one\ntwo")
		}
	})

	t.Run("path", func(t *testing.T) {
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, "head -n 2 "+quotePowerShellPath(inputPath))
		if output != "one\ntwo" {
			t.Fatalf("path head output = %q, want %q", output, "one\ntwo")
		}
	})
}

func TestPowerShellAliasesLessSupportsPipelineInput(t *testing.T) {
	pwshPath, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh is required for this test")
	}

	content, err := GetFeatureTemplate("powershell-aliases", "powershell")
	if err != nil {
		t.Fatalf("GetFeatureTemplate() error = %v", err)
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "powershell-aliases.ps1")
	inputPath := filepath.Join(tempDir, "input.txt")

	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile(scriptPath) error = %v", err)
	}

	inputContent := "one\ntwo\nthree\nfour\n"
	if err := os.WriteFile(inputPath, []byte(inputContent), 0644); err != nil {
		t.Fatalf("os.WriteFile(inputPath) error = %v", err)
	}

	t.Run("pipeline", func(t *testing.T) {
		command := strings.Join([]string{
			"function Out-Host {",
			"param([switch]$Paging, [Parameter(ValueFromPipeline = $true)]$InputObject)",
			"process { $InputObject }",
			"}",
			"Get-Content " + quotePowerShellPath(inputPath) + " | less",
		}, " ")
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, command)
		if output != "one\ntwo\nthree\nfour" {
			t.Fatalf("pipeline less output = %q, want %q", output, "one\ntwo\nthree\nfour")
		}
	})

	t.Run("path", func(t *testing.T) {
		command := strings.Join([]string{
			"function Out-Host {",
			"param([switch]$Paging, [Parameter(ValueFromPipeline = $true)]$InputObject)",
			"process { $InputObject }",
			"}",
			"less " + quotePowerShellPath(inputPath),
		}, " ")
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, command)
		if output != "one\ntwo\nthree\nfour" {
			t.Fatalf("path less output = %q, want %q", output, "one\ntwo\nthree\nfour")
		}
	})
}

func TestPowerShellAliasesLessIgnoresUserStoppedPager(t *testing.T) {
	pwshPath, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh is required for this test")
	}

	content, err := GetFeatureTemplate("powershell-aliases", "powershell")
	if err != nil {
		t.Fatalf("GetFeatureTemplate() error = %v", err)
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "powershell-aliases.ps1")
	inputPath := filepath.Join(tempDir, "input.txt")

	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile(scriptPath) error = %v", err)
	}

	if err := os.WriteFile(inputPath, []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile(inputPath) error = %v", err)
	}

	t.Run("pipeline", func(t *testing.T) {
		command := strings.Join([]string{
			"function Out-Host { [CmdletBinding()] param([switch]$Paging, [Parameter(ValueFromPipeline = $true)]$InputObject) process { $exception = [System.Management.Automation.RuntimeException]::new('The command was stopped by the user.'); $errorRecord = [System.Management.Automation.ErrorRecord]::new($exception, 'OperationStopped', [System.Management.Automation.ErrorCategory]::OperationStopped, $null); $PSCmdlet.ThrowTerminatingError($errorRecord) } }",
			"Get-Content " + quotePowerShellPath(inputPath) + " | less",
			"'completed'",
		}, "; ")
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, command)
		if output != "completed" {
			t.Fatalf("pipeline less output = %q, want %q", output, "completed")
		}
	})

	t.Run("path", func(t *testing.T) {
		command := strings.Join([]string{
			"function Out-Host { [CmdletBinding()] param([switch]$Paging, [Parameter(ValueFromPipeline = $true)]$InputObject) process { $exception = [System.Management.Automation.RuntimeException]::new('The command was stopped by the user.'); $errorRecord = [System.Management.Automation.ErrorRecord]::new($exception, 'OperationStopped', [System.Management.Automation.ErrorCategory]::OperationStopped, $null); $PSCmdlet.ThrowTerminatingError($errorRecord) } }",
			"less " + quotePowerShellPath(inputPath),
			"'completed'",
		}, "; ")
		output := runPowerShellAliasCommand(t, pwshPath, scriptPath, command)
		if output != "completed" {
			t.Fatalf("path less output = %q, want %q", output, "completed")
		}
	})
}

func TestPowerShellAliasesLessPreservesNonUserPagerErrors(t *testing.T) {
	pwshPath, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh is required for this test")
	}

	content, err := GetFeatureTemplate("powershell-aliases", "powershell")
	if err != nil {
		t.Fatalf("GetFeatureTemplate() error = %v", err)
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "powershell-aliases.ps1")
	inputPath := filepath.Join(tempDir, "input.txt")

	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile(scriptPath) error = %v", err)
	}

	if err := os.WriteFile(inputPath, []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile(inputPath) error = %v", err)
	}

	command := strings.Join([]string{
		"function Out-Host { param([switch]$Paging, [Parameter(ValueFromPipeline = $true)]$InputObject) process { throw 'pager exploded' } }",
		"Get-Content " + quotePowerShellPath(inputPath) + " | less",
	}, "; ")

	_, err = runPowerShellAliasCommandWithError(pwshPath, scriptPath, command)
	if err == nil {
		t.Fatal("expected less to preserve non-user pager error")
	}
	if !strings.Contains(err.Error(), "pager exploded") {
		t.Fatalf("expected pager error in output, got %v", err)
	}
}

func runPowerShellAliasCommand(t *testing.T, pwshPath, scriptPath, command string) string {
	t.Helper()

	output, err := runPowerShellAliasCommandWithError(pwshPath, scriptPath, command)
	if err != nil {
		t.Fatalf("pwsh command failed: %v", err)
	}

	return output
}

func runPowerShellAliasCommandWithError(pwshPath, scriptPath, command string) (string, error) {
	powershellCommand := "& { . " + quotePowerShellPath(scriptPath) + "; " + command + " }"
	cmd := exec.Command(pwshPath, "-NoProfile", "-Command", powershellCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, output)
	}

	return normalizePowerShellOutput(string(output)), nil
}

func quotePowerShellPath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "''") + "'"
}

func normalizePowerShellOutput(output string) string {
	normalized := strings.ReplaceAll(output, "\r\n", "\n")
	return strings.TrimSpace(normalized)
}

func TestGetShellExtension(t *testing.T) {
	tests := []struct {
		shellName string
		want      string
	}{
		{"bash", ".sh"},
		{"zsh", ".sh"},
		{"posix", ".sh"},
		{"fish", ".fish"},
		{"powershell", ".ps1"},
		{"unknown", ".sh"},
	}

	for _, tt := range tests {
		t.Run(tt.shellName, func(t *testing.T) {
			got := GetShellExtension(tt.shellName)
			if got != tt.want {
				t.Errorf("GetShellExtension(%q) = %q, want %q", tt.shellName, got, tt.want)
			}
		})
	}
}

func TestRenderFeatureTemplate(t *testing.T) {
	t.Run("renders with oh-my-posh defaults", func(t *testing.T) {
		content := `url={{ .ThemeURL }} config={{ .ConfigFile }} fallback={{ .DefaultConfigPath }}`

		rendered, err := RenderFeatureTemplate(content, "oh-my-posh", "bash", nil)
		if err != nil {
			t.Fatalf("RenderFeatureTemplate() error = %v", err)
		}

		if !strings.Contains(rendered, "jandedobbeleer.omp.json") {
			t.Fatalf("expected default theme URL in rendered content, got %q", rendered)
		}
		if !strings.Contains(rendered, "$OMD_SHELL_ROOT/features/oh-my-posh.omp.json") {
			t.Fatalf("expected default config path in rendered content, got %q", rendered)
		}
	})

	t.Run("renders with option overrides", func(t *testing.T) {
		content := `url={{ .ThemeURL }} config={{ .ConfigFile }} auto={{ .AutoUpgrade }} custom={{ option "theme" }}`
		options := map[string]any{
			"theme":        "catppuccin",
			"config_file":  "/tmp/custom.omp.json",
			"auto_upgrade": true,
		}

		rendered, err := RenderFeatureTemplate(content, "oh-my-posh", "bash", options)
		if err != nil {
			t.Fatalf("RenderFeatureTemplate() error = %v", err)
		}

		if !strings.Contains(rendered, "catppuccin.omp.json") {
			t.Fatalf("expected overridden theme URL in rendered content, got %q", rendered)
		}
		if !strings.Contains(rendered, "/tmp/custom.omp.json") {
			t.Fatalf("expected overridden config file in rendered content, got %q", rendered)
		}
		if !strings.Contains(rendered, "auto=true") {
			t.Fatalf("expected auto upgrade boolean in rendered content, got %q", rendered)
		}
	})
}
