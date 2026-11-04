package interactive

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
)

func themeSelectionModel(t *testing.T) (selectModel, int) {
	t.Helper()
	metadata, _ := catalog.GetFeature("oh-my-posh")
	names := metadata.Options[0].EnumValues
	index := slices.Index(names, "agnoster.minimal")
	model := newSelectModel("Theme", names, index, true)
	if !model.list.FilteringEnabled() || model.list.SelectedItem().(item).index != index {
		t.Fatal("theme list must allow search and preserve the current selection")
	}
	return model, index
}

func TestFilteredSelectionPreservesOriginalIndex(t *testing.T) {
	model, index := themeSelectionModel(t)
	result, _ := model.Update(tea.KeyPressMsg{Code: '/'})
	model = result.(selectModel)
	if model.list.FilterState() != list.Filtering {
		t.Fatal("slash did not start filtering")
	}
	// SetFilterText computes matches synchronously, avoiding the asynchronous
	// terminal event loop while exercising the real list filtering algorithm.
	model.list.SetFilterText("agnoster.minimal")
	model.list.SetFilterState(list.Filtering)
	result, command := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = result.(selectModel)
	if model.list.FilterState() != list.FilterApplied || model.cancelled {
		t.Fatal("Enter should apply the search before selecting a theme")
	}
	if command != nil {
		if _, quits := command().(tea.QuitMsg); quits {
			t.Fatal("applying a search closed the picker")
		}
	}
	result, command = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = result.(selectModel)
	if command == nil || model.list.SelectedItem().(item).index != index {
		t.Fatal("filtered selection lost the original theme index")
	}
	if _, quits := command().(tea.QuitMsg); !quits {
		t.Fatal("Enter did not submit the filtered selection")
	}
}

func TestSearchEscapeAndNavigation(t *testing.T) {
	for _, state := range []list.FilterState{list.Filtering, list.FilterApplied} {
		t.Run(state.String(), func(t *testing.T) {
			model, _ := themeSelectionModel(t)
			model.list.SetFilterText("minimal")
			model.list.SetFilterState(state)
			result, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			cleared := result.(selectModel)
			if cleared.cancelled || cleared.list.FilterState() != list.Unfiltered {
				t.Fatal("Escape should clear search before cancelling the picker")
			}
			result, command := cleared.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if !result.(selectModel).cancelled || command == nil {
				t.Fatal("Escape without a search should cancel the picker")
			}
			for _, navigation := range []struct {
				key  rune
				want error
			}{{'b', ErrBack}, {'x', ErrClear}, {'s', ErrSkip}} {
				result, command := model.Update(tea.KeyPressMsg{Code: navigation.key, Mod: tea.ModCtrl})
				if !errors.Is(result.(selectModel).navigation.err, navigation.want) || command == nil {
					t.Fatalf("search swallowed navigation key Ctrl+%c", navigation.key)
				}
			}
			result, command = model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
			if !result.(selectModel).cancelled || command == nil {
				t.Fatal("Ctrl+C should cancel while searching")
			}
		})
	}
}

func TestUnmatchedSearchDoesNotSelectAnotherTheme(t *testing.T) {
	model, _ := themeSelectionModel(t)
	const query = "no-such-theme-2048"
	model.list.SetFilterText(query)
	model.list.SetFilterState(list.Filtering)
	for range 2 {
		result, command := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		model = result.(selectModel)
		if command != nil || model.cancelled || model.list.SelectedItem() != nil || model.list.FilterState() != list.Filtering || model.list.FilterValue() != query {
			t.Fatal("Enter on an unmatched search must preserve the query without selecting another theme")
		}
	}
}

func TestShortSelectionKeepsFilteringDisabled(t *testing.T) {
	model := newSelectModel("Shell", []string{"bash", "zsh"}, 1, true)
	if model.list.FilteringEnabled() || model.list.SelectedItem().(item).index != 1 {
		t.Fatal("short selection behavior changed")
	}
}

func TestThemeGalleryRemainsVisibleWhileSearching(t *testing.T) {
	metadata, _ := catalog.GetFeature("oh-my-posh")
	option := metadata.Options[0]
	model := newSelectModel(option.DisplayName+" - "+option.Description+" (required)", option.EnumValues, 0, true)
	result, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = result.(selectModel)
	for _, state := range []list.FilterState{list.Unfiltered, list.Filtering, list.FilterApplied} {
		model.list.SetFilterState(state)
		if !strings.Contains(model.View().Content, "https://ohmyposh.dev/docs/themes/") {
			t.Fatalf("gallery URL truncated in an 80-column terminal while %s", state)
		}
	}
}
