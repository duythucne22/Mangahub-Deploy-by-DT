package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/focus"
)

// KeyMap defines all key bindings for the TUI
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Actions
	Enter   key.Binding
	Back    key.Binding
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding

	// View switching
	Dashboard key.Binding
	Browse    key.Binding
	Search    key.Binding
	Chat      key.Binding
	Stats     key.Binding

	// Tab navigation
	NextTab key.Binding
	PrevTab key.Binding

	// Input/Edit
	Submit key.Binding
	Cancel key.Binding
}

// ShouldHandleKey returns true if the key should be handled based on focus mode
func (k KeyMap) ShouldHandleKey(mode focus.Mode, msg tea.KeyMsg) bool {
	// In input mode, only allow ESC, Enter, Tab
	if mode == focus.ModeInput {
		return key.Matches(msg, k.Cancel) ||
			key.Matches(msg, k.Submit) ||
			key.Matches(msg, k.NextTab) ||
			key.Matches(msg, k.PrevTab)
	}

	// In dialog mode, limited keys
	if mode == focus.ModeDialog {
		return key.Matches(msg, k.Enter) ||
			key.Matches(msg, k.Cancel) ||
			key.Matches(msg, k.Quit)
	}

	// Navigation mode allows all keys
	return true
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "u"),
			key.WithHelp("pgup/u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "d"),
			key.WithHelp("pgdown/d", "page down"),
		),

		// Actions
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "ctrl+r"),
			key.WithHelp("r", "refresh"),
		),

		// View switching
		Dashboard: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "dashboard"),
		),
		Browse: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "browse"),
		),
		Search: key.NewBinding(
			key.WithKeys("3", "/"),
			key.WithHelp("3/", "search"),
		),
		Chat: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "chat"),
		),
		Stats: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "stats"),
		),

		// Tab navigation
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),

		// Input/Edit
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// ShortHelp returns a short help message
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Enter, k.Back, k.Help, k.Quit,
	}
}

// FullHelp returns the full help message
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.PageUp, k.PageDown, k.Enter, k.Back},
		{k.Dashboard, k.Browse, k.Search, k.Chat},
		{k.Stats, k.Refresh, k.Quit},
	}
}
