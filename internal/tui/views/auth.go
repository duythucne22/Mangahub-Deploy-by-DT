package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/api"
	"mangahub/internal/tui/styles"
	"mangahub/pkg/models"
)

// AuthMode represents login or register mode
type AuthMode int

const (
	ModeLogin AuthMode = iota
	ModeRegister
)

// AuthModel handles login/register forms
type AuthModel struct {
	mode           AuthMode
	apiClient      *api.Client
	
	// Input fields
	usernameInput  textinput.Model
	emailInput     textinput.Model
	passwordInput  textinput.Model
	confirmInput   textinput.Model
	
	// State
	focusIndex     int
	loading        bool
	err            error
	
	// Window size
	width          int
	height         int
}

// NewAuthModel creates a new auth model
func NewAuthModel(apiClient *api.Client) AuthModel {
	// Username input
	usernameInput := textinput.New()
	usernameInput.Placeholder = "Username"
	usernameInput.CharLimit = 50
	usernameInput.Width = 30
	usernameInput.Focus()
	
	// Email input (for registration)
	emailInput := textinput.New()
	emailInput.Placeholder = "Email"
	emailInput.CharLimit = 100
	emailInput.Width = 30
	
	// Password input
	passwordInput := textinput.New()
	passwordInput.Placeholder = "Password"
	passwordInput.CharLimit = 100
	passwordInput.Width = 30
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '•'
	
	// Confirm password input (for registration)
	confirmInput := textinput.New()
	confirmInput.Placeholder = "Confirm Password"
	confirmInput.CharLimit = 100
	confirmInput.Width = 30
	confirmInput.EchoMode = textinput.EchoPassword
	confirmInput.EchoCharacter = '•'
	
	return AuthModel{
		mode:          ModeLogin,
		apiClient:     apiClient,
		usernameInput: usernameInput,
		emailInput:    emailInput,
		passwordInput: passwordInput,
		confirmInput:  confirmInput,
		focusIndex:    0,
	}
}

// Init initializes the model
func (m AuthModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m AuthModel) Update(msg tea.Msg) (AuthModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			return m.nextField(), nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			return m.prevField(), nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.isSubmitFocused() {
				return m.submit()
			}
			return m.nextField(), nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+t"))):
			// Toggle between login and register
			m.toggleMode()
			return m, nil
		}

	case AuthSuccessMsg:
		m.loading = false
		// Parent model will handle navigation
		return m, nil

	case AuthErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	// Update focused input
	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	case 1:
		if m.mode == ModeRegister {
			m.emailInput, cmd = m.emailInput.Update(msg)
		} else {
			m.passwordInput, cmd = m.passwordInput.Update(msg)
		}
	case 2:
		if m.mode == ModeRegister {
			m.passwordInput, cmd = m.passwordInput.Update(msg)
		}
	case 3:
		if m.mode == ModeRegister {
			m.confirmInput, cmd = m.confirmInput.Update(msg)
		}
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the auth form
func (m AuthModel) View() string {
	var b strings.Builder

	// Title
	title := "🔐 Login"
	if m.mode == ModeRegister {
		title = "📝 Register"
	}
	b.WriteString(styles.TitleStyle.Render(title))
	b.WriteString("\n\n")

	// Form card
	var formContent strings.Builder

	// Username field
	formContent.WriteString(m.renderField("Username", m.usernameInput.View(), m.focusIndex == 0))
	formContent.WriteString("\n")

	if m.mode == ModeRegister {
		// Email field
		formContent.WriteString(m.renderField("Email", m.emailInput.View(), m.focusIndex == 1))
		formContent.WriteString("\n")
		
		// Password field
		formContent.WriteString(m.renderField("Password", m.passwordInput.View(), m.focusIndex == 2))
		formContent.WriteString("\n")
		
		// Confirm password field
		formContent.WriteString(m.renderField("Confirm", m.confirmInput.View(), m.focusIndex == 3))
		formContent.WriteString("\n\n")
		
		// Submit button
		submitStyle := styles.ButtonStyle
		if m.focusIndex == 4 {
			submitStyle = styles.ButtonActiveStyle
		}
		formContent.WriteString(submitStyle.Render("  Register  "))
	} else {
		// Password field
		formContent.WriteString(m.renderField("Password", m.passwordInput.View(), m.focusIndex == 1))
		formContent.WriteString("\n\n")
		
		// Submit button
		submitStyle := styles.ButtonStyle
		if m.focusIndex == 2 {
			submitStyle = styles.ButtonActiveStyle
		}
		formContent.WriteString(submitStyle.Render("  Login  "))
	}

	b.WriteString(styles.CardStyle.Render(formContent.String()))
	b.WriteString("\n\n")

	// Error message
	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// Loading indicator
	if m.loading {
		b.WriteString(styles.SpinnerStyle.Render("⟳ "))
		b.WriteString(styles.InfoStyle.Render("Processing..."))
		b.WriteString("\n\n")
	}

	// Toggle hint
	if m.mode == ModeLogin {
		b.WriteString(styles.HelpStyle.Render("Press Ctrl+T to switch to Register"))
	} else {
		b.WriteString(styles.HelpStyle.Render("Press Ctrl+T to switch to Login"))
	}

	return b.String()
}

// renderField renders a form field with label
func (m AuthModel) renderField(label, input string, focused bool) string {
	labelStyle := styles.MetaKeyStyle
	if focused {
		labelStyle = styles.InputFocusedStyle
	}
	
	return fmt.Sprintf("%s\n%s", labelStyle.Render(label+":"), input)
}

// nextField moves focus to the next field
func (m AuthModel) nextField() AuthModel {
	maxIndex := 2 // login mode
	if m.mode == ModeRegister {
		maxIndex = 4
	}
	
	m.focusIndex = (m.focusIndex + 1) % (maxIndex + 1)
	m.updateFocus()
	return m
}

// prevField moves focus to the previous field
func (m AuthModel) prevField() AuthModel {
	maxIndex := 2
	if m.mode == ModeRegister {
		maxIndex = 4
	}
	
	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = maxIndex
	}
	m.updateFocus()
	return m
}

// updateFocus updates input focus states
func (m *AuthModel) updateFocus() {
	m.usernameInput.Blur()
	m.emailInput.Blur()
	m.passwordInput.Blur()
	m.confirmInput.Blur()

	switch m.focusIndex {
	case 0:
		m.usernameInput.Focus()
	case 1:
		if m.mode == ModeRegister {
			m.emailInput.Focus()
		} else {
			m.passwordInput.Focus()
		}
	case 2:
		if m.mode == ModeRegister {
			m.passwordInput.Focus()
		}
	case 3:
		if m.mode == ModeRegister {
			m.confirmInput.Focus()
		}
	}
}

// isSubmitFocused returns true if submit button is focused
func (m AuthModel) isSubmitFocused() bool {
	if m.mode == ModeLogin {
		return m.focusIndex == 2
	}
	return m.focusIndex == 4
}

// toggleMode switches between login and register
func (m *AuthModel) toggleMode() {
	if m.mode == ModeLogin {
		m.mode = ModeRegister
	} else {
		m.mode = ModeLogin
	}
	m.focusIndex = 0
	m.err = nil
	m.updateFocus()
}

// submit handles form submission
func (m AuthModel) submit() (AuthModel, tea.Cmd) {
	// Validate
	if m.usernameInput.Value() == "" {
		m.err = fmt.Errorf("username is required")
		return m, nil
	}
	
	if m.mode == ModeRegister && m.emailInput.Value() == "" {
		m.err = fmt.Errorf("email is required")
		return m, nil
	}
	
	if m.passwordInput.Value() == "" {
		m.err = fmt.Errorf("password is required")
		return m, nil
	}
	
	if m.mode == ModeRegister && m.passwordInput.Value() != m.confirmInput.Value() {
		m.err = fmt.Errorf("passwords do not match")
		return m, nil
	}

	m.loading = true
	m.err = nil

	if m.mode == ModeLogin {
		return m, m.doLogin()
	}
	return m, m.doRegister()
}

// doLogin performs login API call
func (m AuthModel) doLogin() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.apiClient.Login(ctx, m.usernameInput.Value(), m.passwordInput.Value())
		if err != nil {
			return AuthErrorMsg{Err: err}
		}
		return AuthSuccessMsg{
			Username: resp.User.Username,
			Token:    resp.Token,
			User:     &resp.User,
		}
	}
}

// doRegister performs registration API call
func (m AuthModel) doRegister() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.apiClient.Register(ctx, m.usernameInput.Value(), m.emailInput.Value(), m.passwordInput.Value())
		if err != nil {
			return AuthErrorMsg{Err: err}
		}
		return AuthSuccessMsg{
			Username: resp.User.Username,
			Token:    resp.Token,
			User:     &resp.User,
		}
	}
}

// GetCredentials returns the entered credentials
func (m AuthModel) GetCredentials() (username, password string) {
	return m.usernameInput.Value(), m.passwordInput.Value()
}

// Messages

// AuthSuccessMsg is sent when auth succeeds
type AuthSuccessMsg struct {
	Username string
	Token    string
	User     *models.UserProfile
}

// AuthErrorMsg is sent when auth fails
type AuthErrorMsg struct {
	Err error
}
