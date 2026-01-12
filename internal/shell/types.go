package shell

// ShellConfig represents the configuration for a specific shell
type ShellConfig struct {
	Name        string // Shell name (e.g., "bash", "zsh", "fish", "powershell", "posix")
	ProfilePath string // Path to the shell's profile file (e.g., "~/.bashrc")
	Extension   string // File extension for feature files (e.g., ".sh", ".zsh", ".fish")
	InitScript  string // Name of the init script (e.g., "init.sh", "init.zsh")
}

// SupportedShells returns a map of all supported shells with their configurations
func SupportedShells() map[string]ShellConfig {
	return map[string]ShellConfig{
		"bash": {
			Name:        "bash",
			ProfilePath: "~/.bashrc",
			Extension:   ".sh",
			InitScript:  "init.sh",
		},
		"zsh": {
			Name:        "zsh",
			ProfilePath: "~/.zshrc",
			Extension:   ".zsh",
			InitScript:  "init.zsh",
		},
		"fish": {
			Name:        "fish",
			ProfilePath: "~/.config/fish/config.fish",
			Extension:   ".fish",
			InitScript:  "init.fish",
		},
		"powershell": {
			Name:        "powershell",
			ProfilePath: "$PROFILE",
			Extension:   ".ps1",
			InitScript:  "init.ps1",
		},
		"posix": {
			Name:        "posix",
			ProfilePath: "~/.profile",
			Extension:   ".sh",
			InitScript:  "init.sh",
		},
	}
}

// GetShellConfig returns the configuration for a specific shell
func GetShellConfig(shellName string) (ShellConfig, bool) {
	config, ok := SupportedShells()[shellName]
	return config, ok
}

// IsShellSupported checks if a shell is supported
func IsShellSupported(shellName string) bool {
	_, ok := SupportedShells()[shellName]
	return ok
}
