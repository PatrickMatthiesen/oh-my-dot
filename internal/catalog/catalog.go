package catalog

// OptionType represents the type of a feature option
type OptionType string

const (
	OptionTypeString OptionType = "string"
	OptionTypeInt    OptionType = "int"
	OptionTypeBool   OptionType = "bool"
	OptionTypeEnum   OptionType = "enum"
	OptionTypeFile   OptionType = "file"
	OptionTypePath   OptionType = "path"
)

// OptionMetadata defines a configurable option for a feature
type OptionMetadata struct {
	Name        string      // Internal identifier (e.g., "theme_name")
	DisplayName string      // Human-readable label (e.g., "Theme Name")
	Description string      // Help text for the user
	Type        OptionType  // Data type
	Required    bool        // Whether this option is mandatory
	Default     any // Default value (used if user skips optional field)

	// Type-specific constraints
	EnumValues    []string // Valid values for enum type
	IntMin        *int     // Minimum value for int type
	IntMax        *int     // Maximum value for int type
	PathMustExist bool     // For file/path: must the path already exist?
	FileOnly      bool     // For path: restrict to files only (no directories)

	// Validation
	Validator func(any) error // Custom validation function
}

// FeatureMetadata contains metadata about a feature from the catalog
type FeatureMetadata struct {
	Name            string           // Feature identifier (e.g., "git-prompt")
	Description     string           // Human-readable description
	Category        string           // Category (e.g., "prompt", "completion", "alias")
	DefaultStrategy string           // Default load strategy ("eager", "defer", "on-command")
	DefaultCommands []string         // Default trigger commands for on-command features
	SupportedShells []string         // Shells that support this feature
	Options         []OptionMetadata // Configurable options for this feature
}

// Catalog is the global feature catalog
var Catalog = map[string]FeatureMetadata{
	"core-aliases": {
		Name:            "core-aliases",
		Description:     "Essential command aliases (ls, cd, git shortcuts)",
		Category:        "alias",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish", "posix"},
	},
	"git-prompt": {
		Name:            "git-prompt",
		Description:     "Git branch and status in shell prompt",
		Category:        "prompt",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"kubectl-completion": {
		Name:            "kubectl-completion",
		Description:     "Kubernetes CLI command completion",
		Category:        "completion",
		DefaultStrategy: "on-command",
		DefaultCommands: []string{"kubectl"},
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"docker-completion": {
		Name:            "docker-completion",
		Description:     "Docker CLI command completion",
		Category:        "completion",
		DefaultStrategy: "on-command",
		DefaultCommands: []string{"docker"},
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"nvm": {
		Name:            "nvm",
		Description:     "Node Version Manager integration",
		Category:        "tool",
		DefaultStrategy: "on-command",
		DefaultCommands: []string{"nvm", "node", "npm"},
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"terraform-completion": {
		Name:            "terraform-completion",
		Description:     "Terraform CLI command completion",
		Category:        "completion",
		DefaultStrategy: "on-command",
		DefaultCommands: []string{"terraform", "tf"},
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"aws-completion": {
		Name:            "aws-completion",
		Description:     "AWS CLI command completion",
		Category:        "completion",
		DefaultStrategy: "on-command",
		DefaultCommands: []string{"aws"},
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"gcloud-completion": {
		Name:            "gcloud-completion",
		Description:     "Google Cloud CLI command completion",
		Category:        "completion",
		DefaultStrategy: "on-command",
		DefaultCommands: []string{"gcloud"},
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"python-venv": {
		Name:            "python-venv",
		Description:     "Python virtual environment helpers",
		Category:        "tool",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"directory-shortcuts": {
		Name:            "directory-shortcuts",
		Description:     "Quick navigation to common directories",
		Category:        "alias",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish", "posix"},
	},
	"ssh-agent": {
		Name:            "ssh-agent",
		Description:     "Automatically start SSH agent and load keys",
		Category:        "tool",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish", "posix"},
	},
	"oh-my-dot-completion": {
		Name:            "oh-my-dot-completion",
		Description:     "Shell completions for oh-my-dot commands",
		Category:        "completion",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish"},
	},
	"homebrew-path": {
		Name:            "homebrew-path",
		Description:     "Sets up Homebrew PATH on Linux for package management",
		Category:        "environment",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish", "posix"},
	},
	"powershell-prompt": {
		Name:            "powershell-prompt",
		Description:     "Custom PowerShell prompt with git status",
		Category:        "prompt",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"powershell"},
	},
	"powershell-aliases": {
		Name:            "powershell-aliases",
		Description:     "Common PowerShell aliases and shortcuts",
		Category:        "alias",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"powershell"},
	},
	"posh-git": {
		Name:            "posh-git",
		Description:     "Git prompt and tab completion for PowerShell",
		Category:        "tool",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"powershell"},
	},
	"oh-my-posh": {
		Name:            "oh-my-posh",
		Description:     "Oh My Posh prompt engine with customizable themes",
		Category:        "prompt",
		DefaultStrategy: "eager",
		DefaultCommands: nil,
		SupportedShells: []string{"bash", "zsh", "fish", "powershell"},
		Options: []OptionMetadata{
			{
				Name:        "theme",
				DisplayName: "Theme",
				Description: "Oh My Posh theme to use",
				Type:        OptionTypeEnum,
				Required:    true,
				EnumValues: []string{
					"agnoster",
					"paradox",
					"powerlevel10k_rainbow",
					"robbyrussell",
					"jandedobbeleer",
					"atomic",
					"dracula",
					"pure",
				},
			},
			{
				Name:          "config_file",
				DisplayName:   "Config File",
				Description:   "Path to custom Oh My Posh configuration file (optional)",
				Type:          OptionTypeFile,
				Required:      false,
				PathMustExist: true,
				FileOnly:      true,
			},
			{
				Name:        "auto_upgrade",
				DisplayName: "Auto Upgrade",
				Description: "Automatically check for updates on shell start",
				Type:        OptionTypeBool,
				Required:    false,
				Default:     false,
			},
		},
	},
}

// GetFeature retrieves feature metadata from the catalog
func GetFeature(name string) (FeatureMetadata, bool) {
	metadata, ok := Catalog[name]
	return metadata, ok
}

// SupportsShell checks if a feature supports a specific shell
func (f *FeatureMetadata) SupportsShell(shell string) bool {
	for _, s := range f.SupportedShells {
		if s == shell {
			return true
		}
	}
	return false
}

// ListFeatures returns all features in the catalog
func ListFeatures() []FeatureMetadata {
	features := make([]FeatureMetadata, 0, len(Catalog))
	for _, metadata := range Catalog {
		features = append(features, metadata)
	}
	return features
}

// ListFeaturesByCategory returns features in a specific category
func ListFeaturesByCategory(category string) []FeatureMetadata {
	features := []FeatureMetadata{}
	for _, metadata := range Catalog {
		if metadata.Category == category {
			features = append(features, metadata)
		}
	}
	return features
}

// ListFeaturesForShell returns features that support a specific shell
func ListFeaturesForShell(shell string) []FeatureMetadata {
	features := []FeatureMetadata{}
	for _, metadata := range Catalog {
		if metadata.SupportsShell(shell) {
			features = append(features, metadata)
		}
	}
	return features
}
