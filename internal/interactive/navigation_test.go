package interactive

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFeatureNavigationKeys(t *testing.T) {
	for _, test := range []struct {
		key  rune
		want error
	}{{'b', ErrBack}, {'s', ErrSkip}} {
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
