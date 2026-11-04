package interactive

import (
	"errors"

	tea "charm.land/bubbletea/v2"
)

// ErrBack requests the previous option without discarding other answers.
var ErrBack = errors.New("previous option")

// ErrClear removes an optional answer without changing other answers.
var ErrClear = errors.New("clear optional answer")

// ErrSkip requests skipping this feature, while continuing with other features.
var ErrSkip = errors.New("skip feature")

type promptNavigation struct {
	enabled bool
	err     error
}

func (n *promptNavigation) handle(msg tea.Msg) bool {
	if !n.enabled {
		return false
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}
	switch key.String() {
	case "ctrl+b":
		n.err = ErrBack
	case "ctrl+x":
		n.err = ErrClear
	case "ctrl+s":
		n.err = ErrSkip
	default:
		return false
	}
	return true
}

func (n promptNavigation) help() string {
	if !n.enabled {
		return ""
	}
	return "\nCtrl+B: previous option; Ctrl+X: clear optional answer; Ctrl+S: skip feature; Esc/Ctrl+C: cancel operation"
}
