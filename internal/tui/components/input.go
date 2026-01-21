package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"mangahub/internal/tui/styles"
)

// Input is a professional input component with validation and states
type Input struct {
	textInput textinput.Model
	label     string
	error     string
	required  bool
	disabled  bool
	validator func(string) error
}

// NewInput creates a new input component
func NewInput(label, placeholder string) Input {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 200
	ti.Width = 40

	return Input{
		textInput: ti,
		label:     label,
	}
}

// NewPasswordInput creates a password input
func NewPasswordInput(label string) Input {
	ti := textinput.New()
	ti.Placeholder = "••••••••"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.CharLimit = 100
	ti.Width = 40

	return Input{
		textInput: ti,
		label:     label,
	}
}

// Focus sets the input as focused
func (i *Input) Focus() tea.Cmd {
	return i.textInput.Focus()
}

// Blur removes focus from input
func (i *Input) Blur() {
	i.textInput.Blur()
}

// Focused returns whether input is focused
func (i *Input) Focused() bool {
	return i.textInput.Focused()
}

// SetValue sets the input value
func (i *Input) SetValue(v string) {
	i.textInput.SetValue(v)
}

// Value returns the current input value
func (i *Input) Value() string {
	return i.textInput.Value()
}

// SetError sets an error message
func (i *Input) SetError(err string) {
	i.error = err
}

// ClearError clears the error message
func (i *Input) ClearError() {
	i.error = ""
}

// SetRequired marks input as required
func (i *Input) SetRequired(required bool) {
	i.required = required
}

// SetValidator sets a validation function
func (i *Input) SetValidator(fn func(string) error) {
	i.validator = fn
}

// Validate runs the validator if set
func (i *Input) Validate() error {
	if i.validator != nil {
		return i.validator(i.Value())
	}
	return nil
}

// SetDisabled enables/disables the input
func (i *Input) SetDisabled(disabled bool) {
	i.disabled = disabled
}

// Update handles input updates
func (i *Input) Update(msg tea.Msg) tea.Cmd {
	if i.disabled {
		return nil
	}

	var cmd tea.Cmd
	i.textInput, cmd = i.textInput.Update(msg)

	// Clear error on input change
	if i.error != "" {
		i.error = ""
	}

	return cmd
}

// View renders the input
func (i Input) View() string {
	var labelStyle lipgloss.Style
	var inputStyle lipgloss.Style

	if i.disabled {
		labelStyle = styles.HelpStyle
		inputStyle = styles.HelpStyle
	} else if i.Focused() {
		labelStyle = styles.InputFocusedStyle
		inputStyle = styles.InputFocusedStyle
	} else {
		labelStyle = styles.InputPromptStyle
		inputStyle = styles.InputStyle
	}

	// Render label with required indicator
	label := i.label
	if i.required {
		label += " " + styles.ErrorStyle.Render("*")
	}

	result := labelStyle.Render(label) + "\n"
	result += inputStyle.Render(i.textInput.View())

	// Show error if present
	if i.error != "" {
		result += "\n" + styles.ErrorStyle.Render("✗ "+i.error)
	}

	return result
}
