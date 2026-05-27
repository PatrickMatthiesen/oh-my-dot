package agentcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

const (
	agentDirectoryName  = "omd-agents"
	skillsDirectoryName = "skills"
	skillReadmeName     = "SKILL.md"
)

var skillNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Skill describes a managed agent skill stored in the oh-my-dot repository.
type Skill struct {
	Name string
	Path string
}

func agentSkillsDirectory(repoPath string) string {
	return filepath.Join(repoPath, agentDirectoryName, skillsDirectoryName)
}

func managedSkillDirectory(repoPath, skillName string) string {
	return filepath.Join(agentSkillsDirectory(repoPath), skillName)
}

func skillReadmePath(skillDir string) string {
	return filepath.Join(skillDir, skillReadmeName)
}

func validateSkillName(skillName string) error {
	if !skillNamePattern.MatchString(skillName) {
		return fmt.Errorf("skill name must start with a letter or number and only contain letters, numbers, dots, dashes, or underscores")
	}
	return nil
}

func normalizeSkillSource(path string) (string, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(expanded)
	if err != nil {
		return "", fmt.Errorf("cannot inspect skill path: %w", err)
	}

	if info.IsDir() {
		return filepath.Abs(expanded)
	}

	if filepath.Base(expanded) == skillReadmeName {
		return filepath.Abs(filepath.Dir(expanded))
	}

	return "", fmt.Errorf("skill path must be a directory containing %s", skillReadmeName)
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	if path[0] != '~' {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %w", err)
	}

	if len(path) == 1 {
		return homeDir, nil
	}

	rest := path[1:]
	rest = filepath.Clean(rest)
	rest = trimLeadingPathSeparators(rest)
	return filepath.Join(homeDir, rest), nil
}

func trimLeadingPathSeparators(path string) string {
	for len(path) > 0 && (path[0] == filepath.Separator || path[0] == '/' || path[0] == '\\') {
		path = path[1:]
	}
	return path
}

func sourceContainsSkill(sourceDir string) bool {
	info, err := os.Stat(skillReadmePath(sourceDir))
	return err == nil && !info.IsDir()
}

func discoverSkillSources(sourceDir string) ([]Skill, error) {
	if sourceContainsSkill(sourceDir) {
		return []Skill{{
			Name: filepath.Base(sourceDir),
			Path: sourceDir,
		}}, nil
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read source directory: %w", err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		childDir := filepath.Join(sourceDir, entry.Name())
		if sourceContainsSkill(childDir) {
			skills = append(skills, Skill{
				Name: entry.Name(),
				Path: childDir,
			})
		}
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	if len(skills) == 0 {
		return nil, fmt.Errorf("no skills found; expected %s in %s or one of its child directories", skillReadmeName, sourceDir)
	}

	return skills, nil
}

func listManagedSkills(repoPath string) ([]Skill, error) {
	skillsDir := agentSkillsDirectory(repoPath)
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Skill{}, nil
		}
		return nil, fmt.Errorf("failed to read managed skills directory: %w", err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(skillsDir, entry.Name())
		if sourceContainsSkill(skillDir) {
			skills = append(skills, Skill{
				Name: entry.Name(),
				Path: skillDir,
			})
		}
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}
