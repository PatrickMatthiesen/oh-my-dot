package catalog

// Requirement describes a runtime prerequisite; configuring a feature never
// grants permission to install it or execute a package manager.
type Requirement struct {
	Kind        string // "command" or "module"
	Name        string // Executable or PowerShell module name
	Instruction string // Explicit user-controlled setup guidance
}

// Requirements returns prerequisites for the feature and target shell.
func Requirements(feature, shell string) []Requirement {
	switch feature {
	case "oh-my-posh":
		return []Requirement{{"command", "oh-my-posh", "Install Oh My Posh using https://ohmyposh.dev/docs/installation, then open a new shell."}}
	case "posh-git":
		if shell == "powershell" {
			return []Requirement{{"module", "posh-git", "Run Install-Module posh-git -Scope CurrentUser, then open a new PowerShell session."}}
		}
	case "terminal-icons":
		if shell == "powershell" {
			return []Requirement{{"module", "Terminal-Icons", "Run Install-Module Terminal-Icons -Scope CurrentUser, then open a new PowerShell session."}}
		}
	}
	return nil
}
