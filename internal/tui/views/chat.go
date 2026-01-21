package views

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"mangahub/internal/tui/api"
	"mangahub/internal/tui/styles"
)

// ChatMessage represents a chat message from the server
// Matches the server's hub.go Message struct exactly
type ChatMessage struct {
	Type      string    `json:"type"`      // "message", "join", "leave", "history", "error"
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	MangaID   string    `json:"manga_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRoom represents a manga chat room
type ChatRoom struct {
	ID    string
	Title string
}

// ChatModel handles real-time chat with WebSocket
type ChatModel struct {
	apiClient *api.Client
	wsURL     string
	token     string

	// Connection state
	conn       *websocket.Conn
	connected  bool
	connecting bool
	// connGen increments every time we (re)connect; used to ignore stale reads.
	connGen int64

	// Chat state
	messages         []ChatMessage
	currentRoomID    string
	currentRoomTitle string
	rooms            []ChatRoom
	roomsLoaded      bool

	// UI state
	messageInput  textinput.Model
	inputFocused  bool
	roomCursor    int
	scrollOffset  int
	showRoomList  bool

	// Error handling
	lastError error

	// Window dimensions
	width  int
	height int
}

// NewChatModel creates a new chat model
func NewChatModel(apiClient *api.Client, wsURL, token string) ChatModel {
	input := textinput.New()
	input.Placeholder = "Type your message... (Enter to send)"
	// Server hard-limit is 1KB per WebSocket message (SPEC.md); keep this conservative.
	input.CharLimit = 500
	input.Width = 60
	input.Focus()

	return ChatModel{
		apiClient:    apiClient,
		wsURL:        wsURL,
		token:        token,
		inputFocused: true,
		messageInput: input,
		messages:     make([]ChatMessage, 0),
		rooms:        make([]ChatRoom, 0),
	}
}

// Init initializes the chat model
func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadRooms())
}

// Update handles all messages for the chat view
func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.messageInput.Width = min(m.width-10, 60)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case ChatConnectedMsg:
		if msg.Gen != m.connGen { return m, nil }
		m.connecting = false
		m.connected = true
		m.conn = msg.Conn
		m.lastError = nil
		// Add system message for connection
		m.messages = append(m.messages, ChatMessage{
			Type:      "system",
			Username:  "System",
			Content:   fmt.Sprintf("Connected to #%s", m.currentRoomTitle),
			Timestamp: time.Now(),
		})
		return m, m.listenForMessages(msg.Gen)

	case ChatDisconnectedMsg:
		if msg.Gen != m.connGen { return m, nil }
		m.connected = false
		m.conn = nil
		m.messages = append(m.messages, ChatMessage{
			Type:      "system",
			Username:  "System",
			Content:   "Disconnected from server",
			Timestamp: time.Now(),
		})
		return m, nil

	case ChatMessageReceivedMsg:
		if msg.Gen != m.connGen { return m, nil }
		// Add the received message to our list
		m.messages = append(m.messages, ChatMessage{
			Type:      msg.Type,
			UserID:    msg.UserID,
			Username:  msg.Username,
			MangaID:   msg.MangaID,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		})
		// Auto-scroll to latest
		m.scrollOffset = 0
		// Continue listening
		return m, m.listenForMessages(msg.Gen)

	case ChatRoomsLoadedMsg:
		m.rooms = msg.Rooms
		m.roomsLoaded = true
		if len(m.rooms) > 0 {
			// Auto-select first room and connect
			m.currentRoomID = m.rooms[0].ID
			m.currentRoomTitle = m.rooms[0].Title
			return m.startConnect("Auto-joining first room")
		}
		m.messages = append(m.messages, ChatMessage{
			Type:      "system",
			Username:  "System",
			Content:   "No manga available for chat. Browse manga first!",
			Timestamp: time.Now(),
		})
		return m, nil

	case ChatErrorMsg:
		if msg.Gen != m.connGen { return m, nil }
		m.lastError = msg.Err
		m.connecting = false
		m.connected = false
		m.messages = append(m.messages, ChatMessage{
			Type:      "error",
			Username:  "Error",
			Content:   msg.Err.Error(),
			Timestamp: time.Now(),
		})
		return m, nil
	}

	// Update text input
	if m.inputFocused {
		var cmd tea.Cmd
		m.messageInput, cmd = m.messageInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input
func (m ChatModel) handleKeyPress(msg tea.KeyMsg) (ChatModel, tea.Cmd) {
	// Room selection mode
	if m.showRoomList {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.showRoomList = false
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.roomCursor < len(m.rooms)-1 {
				m.roomCursor++
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.roomCursor > 0 {
				m.roomCursor--
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if len(m.rooms) > 0 && m.roomCursor < len(m.rooms) {
				// Switch room
				m.currentRoomID = m.rooms[m.roomCursor].ID
				m.currentRoomTitle = m.rooms[m.roomCursor].Title
				m.showRoomList = false
				m.messages = nil // Clear messages for new room
				return m.startConnect("Switching room")
			}
			return m, nil
		}
		return m, nil
	}

	// Normal chat mode
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.inputFocused && m.messageInput.Value() != "" && m.connected {
			content := m.messageInput.Value()
			m.messageInput.SetValue("")
			return m, m.sendMessage(content)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		// Toggle room list
		m.showRoomList = !m.showRoomList
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+r"))):
		// Reconnect
		if m.currentRoomID != "" {
			return m.startConnect("Manual reconnect")
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
		// Scroll up
		if m.scrollOffset < len(m.messages)-1 {
			m.scrollOffset++
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
		// Scroll down
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil
	}

	// Pass to text input
	var cmd tea.Cmd
	m.messageInput, cmd = m.messageInput.Update(msg)
	return m, cmd
}

// startConnect updates UI state before dialing.
func (m ChatModel) startConnect(reason string) (ChatModel, tea.Cmd) {
	m.connGen++
	gen := m.connGen
	m.connecting = true
	m.connected = false
	m.lastError = nil
	if reason != "" {
		m.messages = append(m.messages, ChatMessage{
			Type:      "system",
			Username:  "System",
			Content:   reason + "…",
			Timestamp: time.Now(),
		})
	}
	return m, m.connect(gen)
}

// View renders the chat view
func (m ChatModel) View() string {
	var b strings.Builder

	// Header with room info and connection status
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Room selector overlay (if active)
	if m.showRoomList {
		b.WriteString(m.renderRoomSelector())
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("↑/↓ select • Enter join • Esc cancel"))
		return b.String()
	}

	// Messages area
	b.WriteString(m.renderMessages())
	b.WriteString("\n")

	// Divider
	dividerWidth := min(m.width-4, 70)
	b.WriteString(styles.RenderDivider(dividerWidth))
	b.WriteString("\n")

	// Input area
	b.WriteString(m.renderInput())
	b.WriteString("\n\n")

	// Help bar
	b.WriteString(m.renderHelp())

	return b.String()
}

// renderHeader renders the chat header with room and connection info
func (m ChatModel) renderHeader() string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("💬 Chat"))
	b.WriteString("  ")

	// Room badge
	roomName := m.currentRoomTitle
	if roomName == "" {
		roomName = "No Room Selected"
	}
	b.WriteString(styles.BadgePrimaryStyle.Render("#" + roomName))
	b.WriteString("  ")

	// Connection status
	if m.connected {
		b.WriteString(styles.SuccessStyle.Render("● Connected"))
	} else if m.connecting {
		b.WriteString(styles.WarningStyle.Render("○ Connecting..."))
	} else {
		b.WriteString(styles.ErrorStyle.Render("○ Disconnected"))
	}

	// User count (if we had it)
	b.WriteString("\n")

	return b.String()
}

// renderMessages renders the message list with proper formatting
func (m ChatModel) renderMessages() string {
	if len(m.messages) == 0 {
		return styles.HelpStyle.Render("\n  No messages yet. Be the first to say something!\n")
	}

	var b strings.Builder

	// Calculate visible messages (show last N)
	maxVisible := 12
	startIdx := len(m.messages) - maxVisible - m.scrollOffset
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := len(m.messages) - m.scrollOffset
	if endIdx > len(m.messages) {
		endIdx = len(m.messages)
	}
	if endIdx < startIdx {
		endIdx = startIdx
	}

	// Scroll indicator
	if startIdx > 0 {
		b.WriteString(styles.HelpStyle.Render("  ↑ " + fmt.Sprintf("%d more messages", startIdx)))
		b.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		msg := m.messages[i]
		b.WriteString(m.renderSingleMessage(msg))
		b.WriteString("\n")
	}

	// Scroll indicator
	if m.scrollOffset > 0 {
		b.WriteString(styles.HelpStyle.Render("  ↓ " + fmt.Sprintf("%d newer messages", m.scrollOffset)))
		b.WriteString("\n")
	}

	return b.String()
}

// renderSingleMessage renders a single chat message
func (m ChatModel) renderSingleMessage(msg ChatMessage) string {
	var b strings.Builder

	// Format timestamp
	timeStr := msg.Timestamp.Format("15:04:05")

	switch msg.Type {
	case "system", "error":
		// System/error messages centered with different styling
		style := styles.HelpStyle
		if msg.Type == "error" {
			style = styles.ErrorStyle
		}
		b.WriteString("  ")
		b.WriteString(style.Render("━━━ " + msg.Content + " ━━━"))

	case "join":
		// User joined
		b.WriteString("  ")
		b.WriteString(styles.SuccessStyle.Render("→ "))
		b.WriteString(styles.MetaKeyStyle.Render(msg.Username))
		b.WriteString(styles.HelpStyle.Render(" joined the chat"))

	case "leave":
		// User left
		b.WriteString("  ")
		b.WriteString(styles.WarningStyle.Render("← "))
		b.WriteString(styles.MetaKeyStyle.Render(msg.Username))
		b.WriteString(styles.HelpStyle.Render(" left the chat"))

	case "history":
		// Historical message (slightly dimmed)
		b.WriteString("  ")
		b.WriteString(styles.HelpStyle.Render("[" + timeStr + "] "))
		b.WriteString(styles.MetaKeyStyle.Render(msg.Username))
		b.WriteString(styles.HelpStyle.Render(": " + msg.Content))

	default: // "message" or regular
		// Regular chat message
		b.WriteString("  ")
		b.WriteString(styles.HelpStyle.Render("[" + timeStr + "] "))
		b.WriteString(styles.HighlightStyle.Render(msg.Username))
		b.WriteString(styles.CardContentStyle.Render(": " + msg.Content))
	}

	return b.String()
}

// renderRoomSelector renders the room selection panel
func (m ChatModel) renderRoomSelector() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styles.CardTitleStyle.Render("  Select Manga Chat Room"))
	b.WriteString("\n\n")

	if len(m.rooms) == 0 {
		b.WriteString(styles.HelpStyle.Render("  No manga available. Browse manga first!"))
		return b.String()
	}

	// Show rooms with selection cursor
	for i, room := range m.rooms {
		prefix := "    "
		style := styles.ListItemStyle

		if i == m.roomCursor {
			prefix = "  ▸ "
			style = styles.ListItemSelectedStyle
		}

		roomLabel := room.Title
		if room.ID == m.currentRoomID {
			roomLabel += " (current)"
		}

		b.WriteString(style.Render(prefix + "#" + roomLabel))
		b.WriteString("\n")
	}

	return b.String()
}

// renderInput renders the message input area
func (m ChatModel) renderInput() string {
	var b strings.Builder

	if !m.connected {
		b.WriteString(styles.HelpStyle.Render("  [Not connected - press Ctrl+R to reconnect]"))
		return b.String()
	}

	b.WriteString("  ")
	b.WriteString(styles.InputFocusedStyle.Render("> "))
	b.WriteString(m.messageInput.View())

	return b.String()
}

// renderHelp renders the help bar
func (m ChatModel) renderHelp() string {
	if m.showRoomList {
		return styles.HelpStyle.Render("↑/↓ navigate • Enter select • Esc cancel")
	}

	parts := []string{
		"Enter send",
		"Tab rooms",
		"Ctrl+R reconnect",
		"PgUp/PgDn scroll",
	}

	return styles.HelpStyle.Render(strings.Join(parts, " • "))
}

// connect establishes WebSocket connection to the manga chat room
func (m ChatModel) connect(gen int64) tea.Cmd {
	return func() tea.Msg {
		if m.currentRoomID == "" {
			return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("no room selected")}
		}

		// Close existing connection if any
		if m.conn != nil {
			m.conn.Close()
		}

		// Build WebSocket URL: ws://host:port/ws/manga/{manga_id}?token=xxx
		wsURL := strings.TrimRight(m.wsURL, "/") + "/" + m.currentRoomID
		if m.token != "" {
			wsURL += "?token=" + m.token
		}

		// Create dialer with timeout + subprotocol (server advertises mangahub.tui-v1)
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			Subprotocols:     []string{"mangahub.tui-v1"},
		}

		// Headers for the connection
		headers := make(map[string][]string)
		headers["Origin"] = []string{"http://localhost"}
		headers["User-Agent"] = []string{"mangahub-tui/1.0"}

		// Connect
		conn, resp, err := dialer.Dial(wsURL, headers)
		if err != nil {
			// If handshake failed, try to surface server response body (often contains auth error).
			if resp != nil && resp.Body != nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
				if len(body) > 0 {
					return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("connection failed: %w (status=%d body=%s)", err, resp.StatusCode, strings.TrimSpace(string(body)))}
				}
				return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("connection failed: %w (status=%d)", err, resp.StatusCode)}
			}
			return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("connection failed: %w", err)}
		}

		return ChatConnectedMsg{Gen: gen, Conn: conn}
	}
}

// sendMessage sends a chat message through WebSocket
func (m ChatModel) sendMessage(content string) tea.Cmd {
	gen := m.connGen
	return func() tea.Msg {
		if m.conn == nil {
			return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("not connected")}
		}

		// Server expects: {"content": "message text"}
		payload := map[string]string{
			"content": content,
		}

		// Server enforces max 1KB per message; validate before sending.
		data, err := json.Marshal(payload)
		if err != nil {
			return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("send failed: %w", err)}
		}
		if len(data) > 1024 {
			return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("message too large (%d bytes > 1024)", len(data))}
		}
		if err := m.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			if isExpectedWSCloseErr(err) {
				return ChatDisconnectedMsg{Gen: gen}
			}
			return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("send failed: %w", err)}
		}

		return nil
	}
}

// listenForMessages reads messages from WebSocket
func (m ChatModel) listenForMessages(gen int64) tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return ChatDisconnectedMsg{Gen: gen}
		}

		// Read message from server
		// Server sends: {type, user_id, username, manga_id, content, timestamp}
		var msg struct {
			Type      string    `json:"type"`
			UserID    string    `json:"user_id"`
			Username  string    `json:"username"`
			MangaID   string    `json:"manga_id"`
			Content   string    `json:"content"`
			Timestamp time.Time `json:"timestamp"`
		}

		if err := m.conn.ReadJSON(&msg); err != nil {
			// Expected close (including "use of closed network connection") => disconnect.
			if isExpectedWSCloseErr(err) {
				return ChatDisconnectedMsg{Gen: gen}
			}
			return ChatErrorMsg{Gen: gen, Err: fmt.Errorf("read failed: %w", err)}
		}

		// Handle the message based on type
		username := msg.Username
		if username == "" {
			username = "Unknown"
		}

		content := msg.Content
		timestamp := msg.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		return ChatMessageReceivedMsg{
			Gen:       gen,
			Type:      msg.Type,
			UserID:    msg.UserID,
			Username:  username,
			MangaID:   msg.MangaID,
			Content:   content,
			Timestamp: timestamp,
		}
	}
}

// loadRooms fetches manga list to use as chat rooms
func (m ChatModel) loadRooms() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := m.apiClient.ListManga(ctx, 1, 20)
		if err != nil {
			return ChatErrorMsg{Err: fmt.Errorf("failed to load rooms: %w", err)}
		}

		rooms := make([]ChatRoom, 0, len(resp.Data))
		for _, manga := range resp.Data {
			rooms = append(rooms, ChatRoom{
				ID:    manga.ID,
				Title: manga.Title,
			})
		}

		return ChatRoomsLoadedMsg{Rooms: rooms}
	}
}

// Close closes the WebSocket connection
func (m *ChatModel) Close() {
	m.connGen++
	if m.conn != nil {
		m.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		m.conn.Close()
		m.conn = nil
		m.connected = false
	}
}

// SetToken updates the auth token for WebSocket connection
func (m *ChatModel) SetToken(token string) {
	m.token = token
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isExpectedWSCloseErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}
	if websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
		websocket.CloseAbnormalClosure,
	) {
		return true
	}
	var ce *websocket.CloseError
	return errors.As(err, &ce) || strings.Contains(err.Error(), "use of closed network connection")
}

// ============================================================================
// Message Types for Bubble Tea communication
// ============================================================================

// ChatConnectedMsg signals successful WebSocket connection
type ChatConnectedMsg struct {
	Gen  int64
	Conn *websocket.Conn
}

// ChatDisconnectedMsg signals WebSocket disconnection
type ChatDisconnectedMsg struct{ Gen int64 }

// ChatMessageReceivedMsg carries a received chat message
type ChatMessageReceivedMsg struct {
	Gen       int64
	Type      string
	UserID    string
	Username  string
	MangaID   string
	Content   string
	Timestamp time.Time
}

// ChatErrorMsg carries an error
type ChatErrorMsg struct {
	Gen int64
	Err error
}

// ChatRoomsLoadedMsg signals rooms have been loaded
type ChatRoomsLoadedMsg struct {
	Rooms []ChatRoom
}