package interactive

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestFeatureNavigationKeys(t *testing.T) {
	for _, test := range []struct {
		key  rune
		want error
	}{{'b', ErrBack}, {'s', ErrSkip}, {'x', ErrClear}} {
		t.Run(test.want.Error(), func(t *testing.T) {
			key := tea.KeyPressMsg{Code: test.key, Mod: tea.ModCtrl}
			models := []tea.Model{
				textInputModel{navigation: promptNavigation{enabled: true}},
				confirmModel{navigation: promptNavigation{enabled: true}},
				selectModel{navigation: promptNavigation{enabled: true}},
			}
			for _, model := range models {
				result, command := model.Update(key)
				var err error
				switch result := result.(type) {
				case textInputModel:
					err = result.navigation.err
				case confirmModel:
					err = result.navigation.err
				case selectModel:
					err = result.navigation.err
				}
				if !errors.Is(err, test.want) || command == nil {
					t.Fatalf("model %T: err=%v command=%v", model, err, command)
				}
			}
			plain := promptNavigation{}
			if plain.handle(key) {
				t.Fatal("navigation active in an ordinary prompt")
			}
		})
	}
}

func TestPromptsWrapLongQuestionsToWindowWidth(t *testing.T) {
	const width = 40
	question := "Config File - Custom Oh My Posh config file (existing file, Enter to skip)"

	assertFitsWidth := func(t *testing.T, name, content string, maxWidth int) {
		t.Helper()
		lines := strings.Split(content, "\n")
		if len(lines) < 2 {
			t.Fatalf("%s did not wrap into multiple lines: %q", name, content)
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got > maxWidth {
				t.Fatalf("%s line %d exceeds width %d (%d cells): %q", name, i, maxWidth, got, line)
			}
		}
	}

	tiInput := textinput.New()
	tiInput.Focus()
	tiInput.SetWidth(50)
	tiInput.SetValue(`C:\Users\example\Documents\very-long-theme-config-file.omp.json`)
	ti := textInputModel{
		textInput:  tiInput,
		question:   question,
		navigation: promptNavigation{enabled: true},
	}
	res, _ := ti.Update(tea.WindowSizeMsg{Width: width, Height: 24})
	ti = res.(textInputModel)
	tiView := ti.View().Content
	assertFitsWidth(t, "textInputModel", tiView, width)
	if !strings.Contains(tiView, "Config File") || !strings.Contains(tiView, "skip)") {
		t.Fatalf("textInputModel lost question content: %q", tiView)
	}

	cm := confirmModel{
		question:   question,
		navigation: promptNavigation{enabled: true},
	}
	res, _ = cm.Update(tea.WindowSizeMsg{Width: width, Height: 24})
	cm = res.(confirmModel)
	cmView := cm.View().Content
	assertFitsWidth(t, "confirmModel", cmView, width)
	if !strings.Contains(cmView, "Config File") || !strings.Contains(cmView, "skip)") {
		t.Fatalf("confirmModel lost question content: %q", cmView)
	}

	ms := multiSelectModel{
		question: question,
		options:  []string{"a"},
		selected: map[int]bool{},
	}
	res, _ = ms.Update(tea.WindowSizeMsg{Width: width, Height: 24})
	ms = res.(multiSelectModel)
	msView := ms.View().Content
	assertFitsWidth(t, "multiSelectModel", msView, width)
	if !strings.Contains(msView, "Config File") || !strings.Contains(msView, "skip)") {
		t.Fatalf("multiSelectModel lost question content: %q", msView)
	}

	sm := newSelectModel(question, []string{"pure", "atomic"}, 0, true)
	res, _ = sm.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	sm = res.(selectModel)
	smView := sm.View().Content
	assertFitsWidth(t, "selectModel", smView, 60)
	if !strings.Contains(smView, "cancel operation") {
		t.Fatalf("selectModel lost navigation help content: %q", smView)
	}
}
