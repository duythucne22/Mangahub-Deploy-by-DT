package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/api"
	"mangahub/internal/tui/styles"
	"mangahub/pkg/models"
)

// DashboardModel displays trending manga and recent activity
type DashboardModel struct {
	apiClient     *api.Client
	
	// Data
	trending      []models.Manga
	activities    []models.ActivityResponse
	
	// State
	loading       bool
	err           error
	selectedTab   int // 0 = trending, 1 = activity
	cursor        int
	
	// Window size
	width         int
	height        int
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(apiClient *api.Client) DashboardModel {
	return DashboardModel{
		apiClient:   apiClient,
		selectedTab: 0,
		cursor:      0,
	}
}

// Init initializes and loads data
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(m.loadTrending(), m.loadActivity())
}

// Update handles messages
func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			m.selectedTab = (m.selectedTab + 1) % 2
			m.cursor = 0
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			m.cursor++
			m.clampCursor()
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			m.cursor--
			m.clampCursor()
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loading = true
			return m, tea.Batch(m.loadTrending(), m.loadActivity())
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.selectedTab == 0 && len(m.trending) > 0 {
				return m, func() tea.Msg {
					return SelectMangaMsg{MangaID: m.trending[m.cursor].ID}
				}
			}
			return m, nil
		}

	case TrendingLoadedMsg:
		m.loading = false
		m.trending = msg.Manga
		return m, nil

	case ActivityLoadedMsg:
		m.loading = false
		m.activities = msg.Activities
		return m, nil

	case DashboardErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	return m, nil
}

// View renders the dashboard
func (m DashboardModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.TitleStyle.Render("📊 Dashboard"))
	b.WriteString("\n\n")

	// Tabs
	trendingTab := styles.TabStyle.Render("🔥 Trending")
	activityTab := styles.TabStyle.Render("📰 Activity")
	
	if m.selectedTab == 0 {
		trendingTab = styles.TabActiveStyle.Render("🔥 Trending")
	} else {
		activityTab = styles.TabActiveStyle.Render("📰 Activity")
	}
	
	b.WriteString(trendingTab + " " + activityTab)
	b.WriteString("\n")
	b.WriteString(styles.RenderDivider(40))
	b.WriteString("\n\n")

	// Loading state
	if m.loading {
		b.WriteString(styles.SpinnerStyle.Render("⟳ "))
		b.WriteString(styles.InfoStyle.Render("Loading..."))
		return b.String()
	}

	// Error state
	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("Press 'r' to retry"))
		return b.String()
	}

	// Content based on selected tab
	if m.selectedTab == 0 {
		b.WriteString(m.renderTrending())
	} else {
		b.WriteString(m.renderActivity())
	}

	// Help
	b.WriteString("\n\n")
	b.WriteString(styles.HelpStyle.Render("↑/↓ navigate • Tab switch • Enter select • r refresh"))

	return b.String()
}

// renderTrending renders the trending manga list
func (m DashboardModel) renderTrending() string {
	if len(m.trending) == 0 {
		return styles.InfoStyle.Render("No trending manga found")
	}

	var b strings.Builder
	for i, manga := range m.trending {
		if i >= 10 { // Limit display
			break
		}
		
		prefix := "  "
		style := styles.ListItemStyle
		if i == m.cursor {
			prefix = "▸ "
			style = styles.ListItemSelectedStyle
		}
		
		rank := styles.BadgePrimaryStyle.Render(fmt.Sprintf("#%d", i+1))
		title := styles.ListItemTitleStyle.Render(styles.Truncate(manga.Title, 30))
		status := m.renderStatus(manga.Status)
		
		line := fmt.Sprintf("%s%s %s %s", prefix, rank, title, status)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}
	
	return b.String()
}

// renderActivity renders the recent activity list
func (m DashboardModel) renderActivity() string {
	if len(m.activities) == 0 {
		return styles.InfoStyle.Render("No recent activity")
	}

	var b strings.Builder
	for i, activity := range m.activities {
		if i >= 10 { // Limit display
			break
		}
		
		prefix := "  "
		style := styles.ListItemStyle
		if i == m.cursor {
			prefix = "▸ "
			style = styles.ListItemSelectedStyle
		}
		
		icon := m.getActivityIcon(activity.Type)
		
		// Build activity text with user and manga info
		activityText := fmt.Sprintf("%s %s", icon, activity.Type)
		if activity.User != nil {
			activityText += fmt.Sprintf(" by %s", activity.User.Username)
		}
		if activity.Manga != nil {
			mangaTitle := styles.Truncate(activity.Manga.Title, 20)
			activityText += fmt.Sprintf(" on %s", mangaTitle)
		}
		
		line := fmt.Sprintf("%s%s", prefix, activityText)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}
	
	return b.String()
}

// renderStatus renders manga status badge
func (m DashboardModel) renderStatus(status string) string {
	switch status {
	case "ongoing":
		return styles.BadgeSuccessStyle.Render("ongoing")
	case "completed":
		return styles.BadgePrimaryStyle.Render("completed")
	case "hiatus":
		return styles.BadgeWarningStyle.Render("hiatus")
	default:
		return styles.BadgeWarningStyle.Render(status)
	}
}

// getActivityIcon returns icon for activity type
func (m DashboardModel) getActivityIcon(activityType string) string {
	switch activityType {
	case "comment":
		return "💬"
	case "chat":
		return "🗨️"
	case "manga_update":
		return "📖"
	default:
		return "📌"
	}
}

// clampCursor keeps cursor in bounds
func (m *DashboardModel) clampCursor() {
	var max int
	if m.selectedTab == 0 {
		max = len(m.trending) - 1
	} else {
		max = len(m.activities) - 1
	}
	
	if max < 0 {
		max = 0
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > max {
		m.cursor = max
	}
}

// loadTrending loads trending manga
func (m DashboardModel) loadTrending() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		manga, err := m.apiClient.GetTrending(ctx, 10)
		if err != nil {
			return DashboardErrorMsg{Err: err}
		}
		return TrendingLoadedMsg{Manga: manga}
	}
}

// loadActivity loads recent activity
func (m DashboardModel) loadActivity() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		activities, err := m.apiClient.GetRecentActivity(ctx, 10)
		if err != nil {
			return DashboardErrorMsg{Err: err}
		}
		return ActivityLoadedMsg{Activities: activities}
	}
}

// GetSelectedManga returns the currently selected manga ID
func (m DashboardModel) GetSelectedManga() string {
	if m.selectedTab == 0 && m.cursor < len(m.trending) {
		return m.trending[m.cursor].ID
	}
	return ""
}

// Messages

// TrendingLoadedMsg is sent when trending manga is loaded
type TrendingLoadedMsg struct {
	Manga []models.Manga
}

// ActivityLoadedMsg is sent when activity is loaded
type ActivityLoadedMsg struct {
	Activities []models.ActivityResponse
}

// DashboardErrorMsg is sent on dashboard errors
type DashboardErrorMsg struct {
	Err error
}

// SelectMangaMsg is sent when user selects a manga
type SelectMangaMsg struct {
	MangaID string
}
