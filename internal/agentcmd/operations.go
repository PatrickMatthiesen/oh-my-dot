package agentcmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// AddSkills copies one or more skill directories into the managed oh-my-dot agent skills directory.
func AddSkills(repoPath, sourcePath, nameOverride string, force bool, recursive bool) ([]Skill, error) {
	sourceDir, err := normalizeSkillSource(sourcePath)
	if err != nil {
		return nil, err
	}

	sources, err := discoverSkillSources(sourceDir, recursive)
	if err != nil {
		return nil, err
	}

	if nameOverride != "" {
		if len(sources) != 1 {
			return nil, fmt.Errorf("--name can only be used when adding a single skill")
		}
		sources[0].Name = nameOverride
	}

	seenNames := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if err := validateSkillName(source.Name); err != nil {
			return nil, fmt.Errorf("invalid skill name %q: %w", source.Name, err)
		}
		if _, exists := seenNames[source.Name]; exists {
			return nil, fmt.Errorf("multiple discovered skills use the name %q; use --name with a single skill path or rename one of the source directories", source.Name)
		}
		seenNames[source.Name] = struct{}{}
	}

	added := make([]Skill, 0, len(sources))
	for _, source := range sources {
		targetDir := managedSkillDirectory(repoPath, source.Name)
		if err := copyDirectory(source.Path, targetDir, force); err != nil {
			return nil, fmt.Errorf("failed to add skill %s: %w", source.Name, err)
		}

		added = append(added, Skill{
			Name: source.Name,
			Path: targetDir,
		})
	}

	return added, nil
}

// RemoveSkill removes a skill from the managed oh-my-dot agent skills directory.
func RemoveSkill(repoPath, skillName string) error {
	if err := validateSkillName(skillName); err != nil {
		return fmt.Errorf("invalid skill name: %w", err)
	}

	skillDir := managedSkillDirectory(repoPath, skillName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill %q is not managed by oh-my-dot", skillName)
	} else if err != nil {
		return fmt.Errorf("cannot inspect managed skill: %w", err)
	}

	return removeDirectory(skillDir)
}

// ListSkills returns all managed agent skills.
func ListSkills(repoPath string) ([]Skill, error) {
	return listManagedSkills(repoPath)
}

// GetSkill returns a managed agent skill by name.
func GetSkill(repoPath, skillName string) (Skill, error) {
	if err := validateSkillName(skillName); err != nil {
		return Skill{}, fmt.Errorf("invalid skill name: %w", err)
	}

	skillDir := managedSkillDirectory(repoPath, skillName)
	if !sourceContainsSkill(skillDir) {
		return Skill{}, fmt.Errorf("skill %q is not managed by oh-my-dot", skillName)
	}

	return Skill{Name: skillName, Path: skillDir}, nil
}

// ApplySkill copies a managed skill into a user or project agent skills directory.
func ApplySkill(repoPath, skillName, scope, projectPath string, force bool) (string, error) {
	skill, err := GetSkill(repoPath, skillName)
	if err != nil {
		return "", err
	}

	targetRoot, err := targetSkillsDirectory(scope, projectPath)
	if err != nil {
		return "", err
	}

	targetDir := filepath.Join(targetRoot, skill.Name)
	if err := copyDirectory(skill.Path, targetDir, force); err != nil {
		return "", fmt.Errorf("failed to apply skill: %w", err)
	}

	return targetDir, nil
}

// UnapplySkill removes an applied skill from a user or project agent skills directory.
func UnapplySkill(skillName, scope, projectPath string) (string, error) {
	if err := validateSkillName(skillName); err != nil {
		return "", fmt.Errorf("invalid skill name: %w", err)
	}

	targetRoot, err := targetSkillsDirectory(scope, projectPath)
	if err != nil {
		return "", err
	}

	targetDir := filepath.Join(targetRoot, skillName)
	if err := ensurePathInside(targetRoot, targetDir); err != nil {
		return "", err
	}
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return "", fmt.Errorf("skill %q is not applied in %s scope", skillName, scope)
	} else if err != nil {
		return "", fmt.Errorf("cannot inspect applied skill: %w", err)
	}
	if !sourceContainsSkill(targetDir) {
		return "", fmt.Errorf("skill %q is not applied in %s scope: %s does not contain %s", skillName, scope, targetDir, skillReadmeName)
	}

	return targetDir, removeDirectory(targetDir)
}

func targetSkillsDirectory(scope, projectPath string) (string, error) {
	switch scope {
	case "user":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to find home directory: %w", err)
		}
		return filepath.Join(homeDir, ".codex", "skills"), nil
	case "project":
		if projectPath == "" {
			projectPath = "."
		}
		expanded, err := expandPath(projectPath)
		if err != nil {
			return "", err
		}
		projectAbs, err := filepath.Abs(expanded)
		if err != nil {
			return "", fmt.Errorf("failed to resolve project path: %w", err)
		}
		return filepath.Join(projectAbs, ".codex", "skills"), nil
	default:
		return "", fmt.Errorf("scope must be either user or project")
	}
}
