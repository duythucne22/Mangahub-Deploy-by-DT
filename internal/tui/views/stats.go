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

// StatsModel displays user statistics and leaderboards
type StatsModel struct {
	apiClient     *api.Client
	userID        string
	
	// Data
	userStats     *models.UserStatistics
	leaderboard   []models.RankedManga
	
	// State
	loading       bool
	err           error
	selectedTab   int // 0 = my stats, 1 = leaderboard
	cursor        int
	
	// Leaderboard category
	category      string
	categories    []string
	
	// Window size
	width         int
	height        int
}

// NewStatsModel creates a new stats model
func NewStatsModel(apiClient *api.Client, userID string) StatsModel {
	return StatsModel{
		apiClient:  apiClient,
		userID:     userID,
		category:   "weekly",
		categories: []string{"weekly"},
	}
}

// SetUserID sets the user ID
func (m *StatsModel) SetUserID(userID string) {
	m.userID = userID
}

// Init initializes and loads data
func (m StatsModel) Init() tea.Cmd {
	return tea.Batch(m.loadUserStats(), m.loadLeaderboard())
}

// Update handles messages
func (m StatsModel) Update(msg tea.Msg) (StatsModel, tea.Cmd) {
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
			if m.selectedTab == 1 {
				m.cursor++
				if m.cursor >= len(m.leaderboard) {
					m.cursor = len(m.leaderboard) - 1
				}
			}
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.selectedTab == 1 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loading = true
			return m, tea.Batch(m.loadUserStats(), m.loadLeaderboard())
		}

	case UserStatsLoadedMsg:
		m.loading = false
		m.userStats = msg.Stats
		return m, nil

	case LeaderboardLoadedMsg:
		m.loading = false
		m.leaderboard = msg.Entries
		return m, nil

	case StatsErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	return m, nil
}

// View renders the stats view
func (m StatsModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.TitleStyle.Render("📈 Statistics"))
	b.WriteString("\n\n")

	// Tabs
	myStatsTab := styles.TabStyle.Render("📊 My Stats")
	leaderboardTab := styles.TabStyle.Render("🏆 Leaderboard")
	
	if m.selectedTab == 0 {
		myStatsTab = styles.TabActiveStyle.Render("📊 My Stats")
	} else {
		leaderboardTab = styles.TabActiveStyle.Render("🏆 Leaderboard")
	}
	
	b.WriteString(myStatsTab + " " + leaderboardTab)
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

	// Content based on tab
	if m.selectedTab == 0 {
		b.WriteString(m.renderMyStats())
	} else {
		b.WriteString(m.renderLeaderboard())
	}

	// Help
	b.WriteString("\n\n")
	if m.selectedTab == 1 {
		b.WriteString(styles.HelpStyle.Render("↑/↓ navigate • c change category • Tab switch • r refresh"))
	} else {
		b.WriteString(styles.HelpStyle.Render("Tab switch • r refresh"))
	}

	return b.String()
}

// renderMyStats renders user statistics
func (m StatsModel) renderMyStats() string {
	if m.userStats == nil {
		return styles.InfoStyle.Render("No statistics available")
	}

	var b strings.Builder

	// Stats card
	var cardContent strings.Builder
	cardContent.WriteString(styles.CardTitleStyle.Render("Your Activity"))
	cardContent.WriteString("\n\n")

	// Stats grid
	stats := []struct {
		label string
		value string
		icon  string
	}{
		{"Comments", fmt.Sprintf("%d", m.userStats.TotalComments), "💬"},
		{"Chat Messages", fmt.Sprintf("%d", m.userStats.TotalChats), "🗨️"},
		{"Manga Explored", fmt.Sprintf("%d", m.userStats.MangaCount), "📚"},
		{"Current Streak", fmt.Sprintf("%d days", m.userStats.CurrentStreak), "🔥"},
	}

	for _, stat := range stats {
		cardContent.WriteString(fmt.Sprintf("%s %s: %s\n",
			stat.icon,
			styles.MetaKeyStyle.Render(stat.label),
			styles.MetaValueStyle.Render(stat.value),
		))
	}

	b.WriteString(styles.CardStyle.Render(cardContent.String()))
	b.WriteString("\n\n")

	// Top genres
	if len(m.userStats.TopGenres) > 0 {
		b.WriteString(styles.MetaKeyStyle.Render("Top Genres:"))
		b.WriteString("\n")
		for _, genre := range m.userStats.TopGenres {
			b.WriteString("  • ")
			b.WriteString(styles.BadgePrimaryStyle.Render(genre.Name))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderLeaderboard renders the leaderboard
func (m StatsModel) renderLeaderboard() string {
	var b strings.Builder

	// Category selector
	b.WriteString(styles.MetaKeyStyle.Render("Category: "))
	for _, cat := range m.categories {
		if cat == m.category {
			b.WriteString(styles.BadgePrimaryStyle.Render(cat))
		} else {
			b.WriteString(styles.TabStyle.Render(cat))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	if len(m.leaderboard) == 0 {
		b.WriteString(styles.InfoStyle.Render("No leaderboard data available"))
		return b.String()
	}

	// Leaderboard entries
	for i, entry := range m.leaderboard {
		if i >= 10 {
			break
		}

		selected := i == m.cursor
		prefix := "  "
		style := styles.ListItemStyle
		if selected {
			prefix = "▸ "
			style = styles.ListItemSelectedStyle
		}

		// Rank with medal for top 3
		rankStr := fmt.Sprintf("#%d", entry.Rank)
		switch entry.Rank {
		case 1:
			rankStr = "🥇"
		case 2:
			rankStr = "🥈"
		case 3:
			rankStr = "🥉"
		}

		username := styles.ListItemTitleStyle.Render(entry.Manga.Title)
		score := styles.MetaValueStyle.Render(fmt.Sprintf("%d pts", entry.Stats.WeeklyScore))

		line := fmt.Sprintf("%s%s %s - %s", prefix, rankStr, username, score)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// loadUserStats loads user statistics
func (m StatsModel) loadUserStats() tea.Cmd {
	return func() tea.Msg {
		if m.userID == "" {
			return StatsErrorMsg{Err: fmt.Errorf("user ID not set")}
		}
		
		ctx := context.Background()
		stats, err := m.apiClient.GetUserStatistics(ctx, m.userID)
		if err != nil {
			return StatsErrorMsg{Err: err}
		}
		return UserStatsLoadedMsg{Stats: stats}
	}
}

// loadLeaderboard loads leaderboard data
func (m StatsModel) loadLeaderboard() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.apiClient.GetTopManga(ctx, 10)
		if err != nil {
			return StatsErrorMsg{Err: err}
		}
		return LeaderboardLoadedMsg{Entries: resp.Data}
	}
}

// Messages

// UserStatsLoadedMsg is sent when user stats are loaded
type UserStatsLoadedMsg struct {
	Stats *models.UserStatistics
}

// LeaderboardLoadedMsg is sent when leaderboard is loaded
type LeaderboardLoadedMsg struct {
	Entries []models.RankedManga
}

// StatsErrorMsg is sent on stats errors
type StatsErrorMsg struct {
	Err error
}
