package catalog

import (
	"slices"
	"strings"
	"testing"
)

func TestOhMyPoshThemeCatalog(t *testing.T) {
	metadata, _ := GetFeature("oh-my-posh")
	option := metadata.Options[0]
	if len(option.EnumValues) != 125 || !slices.IsSorted(option.EnumValues) {
		t.Fatalf("expected the 125 sorted themes from the pinned upstream snapshot, got %d", len(option.EnumValues))
	}
	for _, name := range []string{"1_shell", "agnoster", "amro", "atomic", "atomicBit", "dracula", "jandedobbeleer", "paradox", "powerlevel10k_rainbow", "pure", "robbyrussell", "agnoster.minimal", "devious-diamonds", "glowsticks", "M365Princess", "jandedobbeleer-accessible"} {
		if !slices.Contains(option.EnumValues, name) {
			t.Errorf("missing theme %q", name)
		}
	}
	if !strings.Contains(option.Description, "https://ohmyposh.dev/docs/themes/") {
		t.Fatal("theme picker does not link to the official gallery")
	}
}

func TestOhMyPoshThemeFilenames(t *testing.T) {
	for _, test := range []struct {
		theme string
		file  string
	}{
		{"agnoster.minimal", "agnoster.minimal.omp.json"},
		{"devious-diamonds", "devious-diamonds.omp.yaml"},
		{"glowsticks", "glowsticks.omp.yaml"},
		{"M365Princess", "M365Princess.omp.json"},
		{"atomicBit", "atomicBit.omp.json"},
		{"pure", "pure.omp.json"},
	} {
		for _, shellName := range []string{"bash", "zsh", "fish", "powershell"} {
			t.Run(test.theme+"/"+shellName, func(t *testing.T) {
				template, err := GetFeatureTemplate("oh-my-posh", shellName)
				if err != nil {
					t.Fatal(err)
				}
				script, err := RenderFeatureTemplate(template, "oh-my-posh", shellName, map[string]any{"theme": test.theme})
				if err != nil {
					t.Fatal(err)
				}
				for _, expected := range []string{"https://raw.githubusercontent.com/JanDeDobbeleer/oh-my-posh/main/themes/" + test.file, "$OMD_SHELL_ROOT/.local/cache/oh-my-posh." + test.file} {
					if !strings.Contains(script, expected) {
						t.Errorf("rendered script missing %q", expected)
					}
				}
			})
		}
	}
}

func TestOhMyPoshThemeNameSafety(t *testing.T) {
	for _, name := range []string{"../pure", "..", "pure/other", `pure\other`, "pure;echo bad", "$(touch bad)", "https://example.com/theme", "pure\nother", "pure..other"} {
		t.Run(name, func(t *testing.T) {
			if _, err := RenderFeatureTemplate("{{ .ThemeURL }}", "oh-my-posh", "bash", map[string]any{"theme": name}); err == nil {
				t.Fatal("unsafe theme name was accepted")
			}
		})
	}
}
