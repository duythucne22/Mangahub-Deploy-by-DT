package focus

// Mode represents different focus modes in the TUI
type Mode int

const (
	// ModeNavigation allows navigation between views and lists
	ModeNavigation Mode = iota
	// ModeInput disables navigation, enables text input
	ModeInput
	// ModeDialog modal dialog is active
	ModeDialog
)

// Manager handles focus mode state
type Manager struct {
	mode       Mode
	inputCount int
}

// NewManager creates a new focus manager
func NewManager() *Manager {
	return &Manager{
		mode: ModeNavigation,
	}
}

// SetMode changes the focus mode
func (m *Manager) SetMode(mode Mode) {
	m.mode = mode
	if mode == ModeInput {
		m.inputCount++
	}
}

// GetMode returns the current focus mode
func (m *Manager) GetMode() Mode {
	return m.mode
}

// IsNavigationMode returns true if in navigation mode
func (m *Manager) IsNavigationMode() bool {
	return m.mode == ModeNavigation
}

// IsInputMode returns true if in input mode
func (m *Manager) IsInputMode() bool {
	return m.mode == ModeInput
}

// IsDialogMode returns true if in dialog mode
func (m *Manager) IsDialogMode() bool {
	return m.mode == ModeDialog
}

// ExitInputMode exits input mode and returns to navigation
func (m *Manager) ExitInputMode() {
	if m.inputCount > 0 {
		m.inputCount--
	}
	if m.inputCount == 0 {
		m.mode = ModeNavigation
	}
}

// ExitDialogMode exits dialog mode and returns to previous mode
func (m *Manager) ExitDialogMode() {
	if m.inputCount > 0 {
		m.mode = ModeInput
	} else {
		m.mode = ModeNavigation
	}
}
