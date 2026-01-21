package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/styles"
)

// Spinner is a loading spinner component
type Spinner struct {
	spinner spinner.Model
	message string
}

// NewSpinner creates a new spinner
func NewSpinner(message string) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return Spinner{
		spinner: s,
		message: message,
	}
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(msg string) {
	s.message = msg
}

// Update updates the spinner
func (s *Spinner) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return cmd
}

// View renders the spinner
func (s Spinner) View() string {
	return s.spinner.View() + " " + styles.InfoStyle.Render(s.message)
}
