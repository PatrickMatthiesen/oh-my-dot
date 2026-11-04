package shell

import (
	"os"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func TestRequirementInspectionWithoutInstallation(t *testing.T) {
	oldLookup, oldModules := lookupRequirementCommand, listPowerShellModules
	t.Cleanup(func() { lookupRequirementCommand, listPowerShellModules = oldLookup, oldModules })
	lookupRequirementCommand = func(string) (string, error) { return "", os.ErrNotExist }
	listPowerShellModules = func() (map[string]bool, error) { return map[string]bool{"posh-git": true}, nil }
	features := []manifest.FeatureConfig{{Name: "oh-my-posh"}, {Name: "posh-git"}, {Name: "terminal-icons"}}
	warnings, err := CheckRequirementsForFeatures("powershell", features)
	if err != nil || len(warnings) != 2 {
		t.Fatalf("prerequisites %v %v", warnings, err)
	}
	if !strings.Contains(strings.Join(warnings, "\n"), "Install-Module Terminal-Icons") {
		t.Fatal("missing actionable setup")
	}
	for i := range features {
		features[i].Disabled = true
	}
	warnings, err = CheckRequirementsForFeatures("powershell", features)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("disabled prerequisite %v %v", warnings, err)
	}
}
