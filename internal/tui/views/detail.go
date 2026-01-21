package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"mangahub/internal/tui/api"
	"mangahub/internal/tui/styles"
	"mangahub/pkg/models"
)

// DetailTab represents tabs in detail view
type DetailTab int

const (
	TabInfo DetailTab = iota
	TabComments
)

// DetailModel displays manga details and comments
type DetailModel struct {
	apiClient     *api.Client
	
	// Current manga
	mangaID       string
	manga         *models.Manga
	
	// Comments
	comments      []models.Comment
	commentsTotal int
	commentsPage  int
	
	// State
	loading       bool
	err           error
	selectedTab   DetailTab
	
	// Comment input
	commentInput  textinput.Model
	inputFocused  bool
	
	// Viewport for scrolling
	viewport      viewport.Model
	
	// Selection
	commentCursor int
	
	// Window size
	width         int
	height        int
}

// NewDetailModel creates a new detail model
func NewDetailModel(apiClient *api.Client) DetailModel {
	commentInput := textinput.New()
	commentInput.Placeholder = "Write a comment..."
	commentInput.CharLimit = 500
	commentInput.Width = 50
	
	return DetailModel{
		apiClient:    apiClient,
		selectedTab:  TabInfo,
		commentsPage: 1,
		commentInput: commentInput,
	}
}

// SetManga sets the manga to display
func (m *DetailModel) SetManga(mangaID string) tea.Cmd {
	m.mangaID = mangaID
	m.loading = true
	m.manga = nil
	m.comments = nil
	m.selectedTab = TabInfo
	return tea.Batch(m.loadManga(), m.loadComments())
}

// Init initializes the model
func (m DetailModel) Init() tea.Cmd {
	if m.mangaID != "" {
		return tea.Batch(m.loadManga(), m.loadComments())
	}
	return nil
}

// Update handles messages
func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		return m, nil

	case tea.KeyMsg:
		if m.inputFocused {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.inputFocused = false
				m.commentInput.Blur()
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.commentInput.Value() != "" {
					m.loading = true
					return m, m.submitComment()
				}
				return m, nil
			}
			
			var cmd tea.Cmd
			m.commentInput, cmd = m.commentInput.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
				m.selectedTab = (m.selectedTab + 1) % 2
				m.commentCursor = 0
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
				if m.selectedTab == TabComments {
					m.commentCursor++
					if m.commentCursor >= len(m.comments) {
						m.commentCursor = len(m.comments) - 1
					}
				} else {
					m.viewport.LineDown(1)
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
				if m.selectedTab == TabComments {
					m.commentCursor--
					if m.commentCursor < 0 {
						m.commentCursor = 0
					}
				} else {
					m.viewport.LineUp(1)
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
				if m.selectedTab == TabComments {
					m.inputFocused = true
					m.commentInput.Focus()
					return m, textinput.Blink
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
				m.loading = true
				return m, tea.Batch(m.loadManga(), m.loadComments())
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("n", "pgdown"))):
				if m.selectedTab == TabComments && m.hasMoreComments() {
					m.commentsPage++
					m.loading = true
					return m, m.loadComments()
				}
				return m, nil
			}
		}

	case MangaDetailLoadedMsg:
		m.loading = false
		m.manga = msg.Manga
		return m, nil

	case CommentsLoadedMsg:
		m.loading = false
		m.comments = msg.Comments
		m.commentsTotal = msg.Total
		return m, nil

	case CommentSubmittedMsg:
		m.loading = false
		m.commentInput.SetValue("")
		m.inputFocused = false
		m.commentInput.Blur()
		// Reload comments
		return m, m.loadComments()

	case DetailErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// View renders the detail view
func (m DetailModel) View() string {
	var b strings.Builder

	if m.manga == nil {
		if m.loading {
			b.WriteString(styles.SpinnerStyle.Render("⟳ "))
			b.WriteString(styles.InfoStyle.Render("Loading..."))
		} else if m.err != nil {
			b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		} else {
			b.WriteString(styles.InfoStyle.Render("No manga selected"))
		}
		return b.String()
	}

	// Title
	b.WriteString(styles.TitleStyle.Render("📖 " + m.manga.Title))
	b.WriteString("\n\n")

	// Tabs
	infoTab := styles.TabStyle.Render("📋 Info")
	commentsTab := styles.TabStyle.Render(fmt.Sprintf("💬 Comments (%d)", m.commentsTotal))
	
	if m.selectedTab == TabInfo {
		infoTab = styles.TabActiveStyle.Render("📋 Info")
	} else {
		commentsTab = styles.TabActiveStyle.Render(fmt.Sprintf("💬 Comments (%d)", m.commentsTotal))
	}
	
	b.WriteString(infoTab + " " + commentsTab)
	b.WriteString("\n")
	b.WriteString(styles.RenderDivider(50))
	b.WriteString("\n\n")

	// Content based on tab
	if m.selectedTab == TabInfo {
		b.WriteString(m.renderInfo())
	} else {
		b.WriteString(m.renderComments())
	}

	// Help
	b.WriteString("\n\n")
	if m.inputFocused {
		b.WriteString(styles.HelpStyle.Render("Enter submit • Esc cancel"))
	} else if m.selectedTab == TabComments {
		b.WriteString(styles.HelpStyle.Render("c comment • ↑/↓ navigate • Tab switch • n more • r refresh"))
	} else {
		b.WriteString(styles.HelpStyle.Render("↑/↓ scroll • Tab switch • r refresh"))
	}

	return b.String()
}

// renderInfo renders manga information
func (m DetailModel) renderInfo() string {
	if m.manga == nil {
		return ""
	}

	var b strings.Builder

	// Status
	b.WriteString(styles.RenderKeyValue("Status", m.renderStatus(m.manga.Status)))
	b.WriteString("\n\n")

	// Description
	b.WriteString(styles.MetaKeyStyle.Render("Description:"))
	b.WriteString("\n")
	if m.manga.Description != "" {
		b.WriteString(styles.CardContentStyle.Render(m.manga.Description))
	} else {
		b.WriteString(styles.HelpStyle.Render("No description available"))
	}
	b.WriteString("\n\n")

	// Cover URL
	if m.manga.CoverURL != "" {
		b.WriteString(styles.RenderKeyValue("Cover", m.manga.CoverURL))
		b.WriteString("\n")
	}

	// Created date
	b.WriteString(styles.RenderKeyValue("Added", m.manga.CreatedAt.Format("Jan 2, 2006")))

	return b.String()
}

// renderComments renders comments list
func (m DetailModel) renderComments() string {
	var b strings.Builder

	// Comment input
	if m.inputFocused {
		b.WriteString(styles.InputFocusedStyle.Render("New Comment:"))
		b.WriteString("\n")
		b.WriteString(m.commentInput.View())
		b.WriteString("\n\n")
	}

	if len(m.comments) == 0 {
		b.WriteString(styles.HelpStyle.Render("No comments yet. Press 'c' to add one!"))
		return b.String()
	}

	// Comments list
	for i, comment := range m.comments {
		selected := i == m.commentCursor

		// Comment card
		var commentContent strings.Builder
		
		// Header: username and date
		username := "Anonymous"
		if comment.UserID != "" {
			username = comment.UserID[:8] + "..."
		}
		commentContent.WriteString(styles.CardTitleStyle.Render(username))
		commentContent.WriteString("  ")
		commentContent.WriteString(styles.HelpStyle.Render(comment.CreatedAt.Format("Jan 2, 2006 15:04")))
		commentContent.WriteString("\n")
		
		// Content
		commentContent.WriteString(styles.CardContentStyle.Render(comment.Content))

		style := styles.CardStyle
		if selected {
			style = style.BorderForeground(lipgloss.Color(styles.Pink))
		}
		
		b.WriteString(style.Render(commentContent.String()))
		b.WriteString("\n")
	}

	// More indicator
	if m.hasMoreComments() {
		b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("Showing %d of %d comments", len(m.comments), m.commentsTotal)))
	}

	return b.String()
}

// renderStatus renders status as styled text
func (m DetailModel) renderStatus(status string) string {
	switch status {
	case "ongoing":
		return styles.SuccessStyle.Render("Ongoing")
	case "completed":
		return styles.InfoStyle.Render("Completed")
	case "hiatus":
		return styles.WarningStyle.Render("Hiatus")
	default:
		return status
	}
}

// hasMoreComments returns true if there are more comments to load
func (m DetailModel) hasMoreComments() bool {
	return len(m.comments) < m.commentsTotal
}

// loadManga loads manga details
func (m DetailModel) loadManga() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		manga, err := m.apiClient.GetManga(ctx, m.mangaID)
		if err != nil {
			return DetailErrorMsg{Err: err}
		}
		return MangaDetailLoadedMsg{Manga: manga}
	}
}

// loadComments loads comments for the manga
func (m DetailModel) loadComments() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.apiClient.ListComments(ctx, m.mangaID, m.commentsPage, 20)
		if err != nil {
			return DetailErrorMsg{Err: err}
		}
		return CommentsLoadedMsg{
			Comments: resp.Data,
			Total:    resp.Meta.Total,
		}
	}
}

// submitComment submits a new comment
func (m DetailModel) submitComment() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := m.apiClient.CreateComment(ctx, m.mangaID, m.commentInput.Value(), nil)
		if err != nil {
			return DetailErrorMsg{Err: err}
		}
		return CommentSubmittedMsg{}
	}
}

// Messages

// MangaDetailLoadedMsg is sent when manga details are loaded
type MangaDetailLoadedMsg struct {
	Manga *models.Manga
}

// CommentsLoadedMsg is sent when comments are loaded
type CommentsLoadedMsg struct {
	Comments []models.Comment
	Total    int
}

// CommentSubmittedMsg is sent when comment is submitted
type CommentSubmittedMsg struct{}

// DetailErrorMsg is sent on detail errors
type DetailErrorMsg struct {
	Err error
}
