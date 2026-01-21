package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/styles"
)

// ButtonState represents button states
type ButtonState int

const (
	ButtonNormal ButtonState = iota
	ButtonActive
	ButtonDisabled
	ButtonLoading
)

// Button is a professional button component
type Button struct {
	label    string
	state    ButtonState
	onSubmit func() tea.Msg
}

// NewButton creates a new button
func NewButton(label string, onSubmit func() tea.Msg) Button {
	return Button{
		label:    label,
		state:    ButtonNormal,
		onSubmit: onSubmit,
	}
}

// SetState sets the button state
func (b *Button) SetState(state ButtonState) {
	b.state = state
}

// SetLabel updates the button label
func (b *Button) SetLabel(label string) {
	b.label = label
}

// Submit triggers the button action
func (b *Button) Submit() tea.Msg {
	if b.state == ButtonDisabled || b.state == ButtonLoading {
		return nil
	}
	if b.onSubmit != nil {
		return b.onSubmit()
	}
	return nil
}

// View renders the button
func (b Button) View() string {
	label := b.label
	
	switch b.state {
	case ButtonActive:
		return styles.ButtonActiveStyle.Render("[ " + label + " ]")
	case ButtonDisabled:
		return styles.HelpStyle.Render("[ " + label + " ]")
	case ButtonLoading:
		return styles.ButtonActiveStyle.Render("[ ⟳ " + label + "... ]")
	default:
		return styles.ButtonStyle.Render("[ " + label + " ]")
	}
}
