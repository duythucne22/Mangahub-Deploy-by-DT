package tui

import (
	"context"
	"log"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/api"
	"mangahub/internal/tui/config"
	"mangahub/internal/tui/focus"
	"mangahub/internal/tui/grpc"
	"mangahub/internal/tui/styles"
	"mangahub/internal/tui/views"
)

// View represents different screens in the TUI
type View int

const (
	ViewAuth View = iota
	ViewDashboard
	ViewBrowse
	ViewSearch
	ViewDetail
	ViewChat
	ViewStats
)

// Model is the root Bubble Tea model
type Model struct {
	// Configuration
	config *config.Config

	// API client
	apiClient *api.Client
	
	// gRPC client for streaming search
	grpcClient *grpc.Client
	
	// Focus manager
	focusManager *focus.Manager

	// Current view
	currentView  View
	previousView View

	// Key bindings
	keys KeyMap

	// Window dimensions
	width  int
	height int

	// User state
	isAuthenticated bool
	currentUser     string
	currentUserID   string
	token           string

	// View models
	authModel      views.AuthModel
	dashboardModel views.DashboardModel
	browseModel    views.BrowseModel
	searchModel    views.SearchModel
	detailModel    views.DetailModel
	chatModel      views.ChatModel
	statsModel     views.StatsModel

	// Error state
	err error
}

// New creates a new TUI application
func New(cfg *config.Config) *Model {
	apiClient := api.NewClient(cfg.GetHTTPBaseURL())
	focusMgr := focus.NewManager()

	// Initialize gRPC client for streaming search
	grpcClient, err := grpc.NewClient(cfg.GetGRPCAddr())
	if err != nil {
		log.Printf("gRPC client unavailable (will fallback to HTTP): %v", err)
		grpcClient = nil
	}

	m := &Model{
		config:          cfg,
		apiClient:       apiClient,
		grpcClient:      grpcClient,
		focusManager:    focusMgr,
		currentView:     ViewAuth,
		keys:            DefaultKeyMap(),
		isAuthenticated: false,
	}

	// Initialize view models
	m.authModel = views.NewAuthModel(apiClient)
	m.dashboardModel = views.NewDashboardModel(apiClient)
	m.browseModel = views.NewBrowseModel(apiClient)
	m.searchModel = views.NewSearchModel(apiClient, grpcClient)
	m.detailModel = views.NewDetailModel(apiClient)
	m.chatModel = views.NewChatModel(apiClient, cfg.GetWebSocketURL(), "")
	m.statsModel = views.NewStatsModel(apiClient, "")

	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.authModel.Init()
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Propagate to views
		m.authModel, _ = m.authModel.Update(msg)
		m.dashboardModel, _ = m.dashboardModel.Update(msg)
		m.browseModel, _ = m.browseModel.Update(msg)
		m.searchModel, _ = m.searchModel.Update(msg)
		m.detailModel, _ = m.detailModel.Update(msg)
		m.chatModel, _ = m.chatModel.Update(msg)
		m.statsModel, _ = m.statsModel.Update(msg)
		return m, nil

	case tea.KeyMsg:
		// Global key bindings (only when not in input mode)
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.chatModel.Close()
			return m, tea.Quit

		case key.Matches(msg, m.keys.Dashboard):
			if m.isAuthenticated && m.currentView != ViewAuth {
				m.previousView = m.currentView
				m.currentView = ViewDashboard
				return m, m.dashboardModel.Init()
			}

		case key.Matches(msg, m.keys.Browse):
			if m.isAuthenticated && m.currentView != ViewAuth {
				m.previousView = m.currentView
				m.currentView = ViewBrowse
				return m, m.browseModel.Init()
			}

		case key.Matches(msg, m.keys.Search):
			if m.isAuthenticated && m.currentView != ViewAuth {
				m.previousView = m.currentView
				m.currentView = ViewSearch
				return m, m.searchModel.Init()
			}

		case key.Matches(msg, m.keys.Chat):
			if m.isAuthenticated && m.currentView != ViewAuth {
				m.previousView = m.currentView
				m.currentView = ViewChat
				return m, m.chatModel.Init()
			}

		case key.Matches(msg, m.keys.Stats):
			if m.isAuthenticated && m.currentView != ViewAuth {
				m.previousView = m.currentView
				m.currentView = ViewStats
				return m, m.statsModel.Init()
			}

		}

	// Handle auth messages
	case views.AuthSuccessMsg:
		m.isAuthenticated = true
		m.currentUser = msg.Username
		m.token = msg.Token
		m.apiClient.SetToken(msg.Token)
		m.chatModel.SetToken(msg.Token)
		if msg.User != nil {
			m.currentUserID = msg.User.ID
			m.statsModel.SetUserID(msg.User.ID)
		}
		m.currentView = ViewDashboard
		return m, m.dashboardModel.Init()

	case views.AuthErrorMsg:
		m.err = msg.Err
		return m, nil

	// Handle navigation from views
	case views.SelectMangaMsg:
		m.previousView = m.currentView
		m.currentView = ViewDetail
		return m, m.detailModel.SetManga(msg.MangaID)
	}

	// Route to current view
	return m.updateCurrentView(msg)
}

// updateCurrentView routes updates to the active view
func (m Model) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentView {
	case ViewAuth:
		m.authModel, cmd = m.authModel.Update(msg)
	case ViewDashboard:
		m.dashboardModel, cmd = m.dashboardModel.Update(msg)
	case ViewBrowse:
		m.browseModel, cmd = m.browseModel.Update(msg)
	case ViewSearch:
		m.searchModel, cmd = m.searchModel.Update(msg)
	case ViewDetail:
		m.detailModel, cmd = m.detailModel.Update(msg)
	case ViewChat:
		m.chatModel, cmd = m.chatModel.Update(msg)
	case ViewStats:
		m.statsModel, cmd = m.statsModel.Update(msg)
	}

	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Render current view
	var content string
	switch m.currentView {
	case ViewAuth:
		content = m.authModel.View()
	case ViewDashboard:
		content = m.dashboardModel.View()
	case ViewBrowse:
		content = m.browseModel.View()
	case ViewSearch:
		content = m.searchModel.View()
	case ViewDetail:
		content = m.detailModel.View()
	case ViewChat:
		content = m.chatModel.View()
	case ViewStats:
		content = m.statsModel.View()
	default:
		content = "Unknown view"
	}

	// Add status bar if authenticated
	var statusBar string
	if m.isAuthenticated {
		statusBar = m.renderStatusBar()
	}

	// Apply app style and combine
	return styles.AppStyle.Render(content + "\n\n" + statusBar)
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	// Current view name
	viewName := ""
	switch m.currentView {
	case ViewDashboard:
		viewName = "Dashboard"
	case ViewBrowse:
		viewName = "Browse"
	case ViewSearch:
		viewName = "Search"
	case ViewDetail:
		viewName = "Detail"
	case ViewChat:
		viewName = "Chat"
	case ViewStats:
		viewName = "Statistics"
	}

	left := styles.StatusBarActiveStyle.Render("● " + viewName)
	right := styles.StatusBarStyle.Render("User: " + m.currentUser + " | 1-5 views | ? help | q quit")

	// Calculate spacing
	spacing := m.width - len(left) - len(right) - 4
	if spacing < 0 {
		spacing = 0
	}
	spaces := ""
	for i := 0; i < spacing; i++ {
		spaces += " "
	}

	return left + spaces + right
}

// Messages

// AuthSuccessMsg is sent when authentication succeeds
type AuthSuccessMsg struct {
	Username string
	Token    string
}

// LogoutMsg is sent when user logs out
type LogoutMsg struct{}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

// Helper to send auth success
func SendAuthSuccess(username, token string) tea.Cmd {
	return func() tea.Msg {
		return AuthSuccessMsg{
			Username: username,
			Token:    token,
		}
	}
}

// Helper to perform login
func (m *Model) Login(username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		authResp, err := m.apiClient.Login(ctx, username, password)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return views.AuthSuccessMsg{
			Username: authResp.User.Username,
			Token:    authResp.Token,
			User:     &authResp.User,
		}
	}
}

// Helper to perform registration
func (m *Model) Register(username, email, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		authResp, err := m.apiClient.Register(ctx, username, email, password)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return views.AuthSuccessMsg{
			Username: authResp.User.Username,
			Token:    authResp.Token,
			User:     &authResp.User,
		}
	}
}
