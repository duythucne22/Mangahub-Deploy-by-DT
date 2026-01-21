package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/styles"
)

// ErrorView displays errors with retry capability
type ErrorView struct {
	err     error
	message string
	onRetry func() tea.Msg
}

// NewErrorView creates a new error view
func NewErrorView(err error, message string, onRetry func() tea.Msg) ErrorView {
	return ErrorView{
		err:     err,
		message: message,
		onRetry: onRetry,
	}
}

// SetError updates the error
func (e *ErrorView) SetError(err error) {
	e.err = err
}

// Clear clears the error
func (e *ErrorView) Clear() {
	e.err = nil
	e.message = ""
}

// HasError returns whether an error is present
func (e ErrorView) HasError() bool {
	return e.err != nil
}

// Retry triggers the retry action
func (e ErrorView) Retry() tea.Msg {
	if e.onRetry != nil {
		return e.onRetry()
	}
	return nil
}

// View renders the error
func (e ErrorView) View() string {
	if !e.HasError() {
		return ""
	}

	result := styles.CardStyle.Render(
		styles.ErrorStyle.Render("⚠ Error") + "\n\n" +
			styles.CardContentStyle.Render(e.message) + "\n" +
			styles.HelpStyle.Render(e.err.Error()) + "\n\n" +
			styles.ButtonStyle.Render("[ Press R to Retry ]") + " " +
			styles.HelpStyle.Render("or ESC to go back"),
	)

	return result
}
