package interactive

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestConfirmNavigationAndSubmit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		initial bool
		keys    []rune
		want    bool
	}{
		{"left selects visible Yes", false, []rune{tea.KeyLeft}, true},
		{"right selects visible No", true, []rune{tea.KeyRight}, false},
		{"repeated left stays Yes", false, []rune{tea.KeyLeft, tea.KeyLeft}, true},
		{"left then right selects No", false, []rune{tea.KeyLeft, tea.KeyRight}, false},
		{"h selects Yes", false, []rune{'h'}, true},
		{"l selects No", true, []rune{'l'}, false},
		{"default remains No", false, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := confirmModel{question: "Delete local file?", selected: tc.initial}
			for _, code := range tc.keys {
				updated, command := m.Update(tea.KeyPressMsg{Code: code})
				m = updated.(confirmModel)
				if command != nil {
					t.Fatal("navigation must not submit")
				}
			}
			if m.selected != tc.want {
				t.Fatalf("selected=%v, want %v", m.selected, tc.want)
			}
			label := "> [No]"
			other := "> [Yes]"
			if tc.want {
				label, other = other, label
			}
			if view := m.View().Content; !strings.Contains(view, label) || strings.Contains(view, other) {
				t.Fatalf("wrong visible selection: %q", view)
			}
			updated, command := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			result := updated.(confirmModel)
			if result.selected != tc.want || result.cancelled || command == nil {
				t.Fatalf("incorrect submitted model: %+v", result)
			}
			if _, ok := command().(tea.QuitMsg); !ok {
				t.Fatal("Enter did not finish prompt")
			}
		})
	}
}

func TestConfirmShortcutsAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		key          rune
		want, cancel bool
	}{
		{'y', true, false}, {'Y', true, false}, {'n', false, false}, {'N', false, false}, {tea.KeyEscape, false, true},
	} {
		t.Run(string(tc.key), func(t *testing.T) {
			updated, command := (confirmModel{}).Update(tea.KeyPressMsg{Code: tc.key})
			m := updated.(confirmModel)
			if m.selected != tc.want || m.cancelled != tc.cancel || command == nil {
				t.Fatalf("unexpected result: %+v", m)
			}
		})
	}
}

func TestConfirmIgnoresKeyRelease(t *testing.T) {
	updated, command := (confirmModel{selected: true}).Update(tea.KeyReleaseMsg{Code: 'n'})
	if !updated.(confirmModel).selected || command != nil {
		t.Fatal("release event changed or submitted confirmation")
	}
}
