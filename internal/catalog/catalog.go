package catalog

// FeatureMetadata contains metadata about a feature from the catalog
type FeatureMetadata struct {
	Name            string   // Feature identifier (e.g., "git-prompt")
	Description     string   // Human-readable description
	Category        string   // Category (e.g., "prompt", "completion", "alias")
	DefaultStrategy string   // Default load strategy ("eager", "defer", "on-command")
	DefaultCommands []string // Default trigger commands for on-command features
	SupportedShells []string // Shells that support this feature
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
		DefaultStrategy: "defer",
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
