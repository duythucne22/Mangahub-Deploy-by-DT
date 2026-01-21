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

// BrowseModel displays a paginated manga list
type BrowseModel struct {
	apiClient     *api.Client
	
	// Data
	manga         []models.Manga
	total         int
	genres        []string // Available genres for filtering
	
	// Filters
	selectedGenreID   string // Selected genre filter ("" for all)
	selectedGenreName string // Selected genre display name
	
	// Pagination
	page          int
	limit         int
	
	// State
	loading       bool
	err           error
	cursor        int
	genreMode     bool // true when selecting genre
	genreCursor   int  // cursor for genre selection
	
	// Window size
	width         int
	height        int
}

// Common manga genres
var commonGenres = []string{
	"All",
	"Action",
	"Adventure",
	"Comedy",
	"Drama",
	"Fantasy",
	"Horror",
	"Mystery",
	"Romance",
	"Sci-Fi",
	"Slice of Life",
	"Sports",
	"Supernatural",
	"Thriller",
}

// NewBrowseModel creates a new browse model
func NewBrowseModel(apiClient *api.Client) BrowseModel {
	return BrowseModel{
		apiClient:     apiClient,
		page:          1,
		limit:         20,
		cursor:        0,
		genres:        commonGenres,
		selectedGenreID:   "",
		selectedGenreName: "",
		genreMode:     false,
		genreCursor:   0,
	}
}

// Init initializes and loads data
func (m BrowseModel) Init() tea.Cmd {
	return m.loadManga()
}

// Update handles messages
func (m BrowseModel) Update(msg tea.Msg) (BrowseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Genre selection mode
		if m.genreMode {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "g"))):
				m.genreMode = false
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
				m.genreCursor++
				if m.genreCursor >= len(m.genres) {
					m.genreCursor = len(m.genres) - 1
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
				m.genreCursor--
				if m.genreCursor < 0 {
					m.genreCursor = 0
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Select genre
				if m.genreCursor == 0 {
					m.selectedGenreID = "" // "All" means no filter
					m.selectedGenreName = ""
				} else {
					m.selectedGenreName = m.genres[m.genreCursor]
					m.selectedGenreID = genreToID(m.selectedGenreName)
				}
				m.genreMode = false
				m.page = 1
				m.cursor = 0
				m.loading = true
				return m, m.loadManga()
			}
			return m, nil
		}
		
		// Normal navigation mode
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			// Toggle genre selection
			m.genreMode = true
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			m.cursor++
			if m.cursor >= len(m.manga) {
				m.cursor = len(m.manga) - 1
			}
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "pgdown"))):
			// Next page
			if m.hasNextPage() {
				m.page++
				m.cursor = 0
				m.loading = true
				return m, m.loadManga()
			}
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("p", "pgup"))):
			// Previous page
			if m.page > 1 {
				m.page--
				m.cursor = 0
				m.loading = true
				return m, m.loadManga()
			}
			return m, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loading = true
			return m, m.loadManga()
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if len(m.manga) > 0 {
				return m, func() tea.Msg {
					return SelectMangaMsg{MangaID: m.manga[m.cursor].ID}
				}
			}
			return m, nil
		}

	case MangaListLoadedMsg:
		m.loading = false
		m.manga = msg.Manga
		m.total = msg.Total
		return m, nil

	case BrowseErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	return m, nil
}

// View renders the browse view
func (m BrowseModel) View() string {
	var b strings.Builder

	// Genre selection overlay
	if m.genreMode {
		return m.renderGenreSelection()
	}

	// Title with pagination info and genre filter
	pageInfo := fmt.Sprintf("Page %d/%d", m.page, m.totalPages())
	b.WriteString(styles.TitleStyle.Render("📚 Browse Manga"))
	b.WriteString("  ")
	b.WriteString(styles.SubtitleStyle.Render(pageInfo))
	
	// Show active genre filter
	if m.selectedGenreName != "" {
		b.WriteString("  ")
		genreTag := styles.BadgePrimaryStyle.Render(m.selectedGenreName)
		b.WriteString(genreTag)
	}
	b.WriteString("\n\n")

	// Loading state
	if m.loading {
		b.WriteString(styles.SpinnerStyle.Render("⟳ "))
		b.WriteString(styles.InfoStyle.Render("Loading manga..."))
		return b.String()
	}

	// Error state
	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("Press 'r' to retry"))
		return b.String()
	}

	// Empty state
	if len(m.manga) == 0 {
		b.WriteString(styles.InfoStyle.Render("No manga found"))
		return b.String()
	}

	// Manga list
	for i, manga := range m.manga {
		prefix := "  "
		style := styles.ListItemStyle
		if i == m.cursor {
			prefix = "▸ "
			style = styles.ListItemSelectedStyle
		}
		
		title := styles.ListItemTitleStyle.Render(styles.Truncate(manga.Title, 40))
		status := m.renderStatus(manga.Status)
		
		line := fmt.Sprintf("%s%s %s", prefix, title, status)
		b.WriteString(style.Render(line))
		
		// Description on next line for selected item
		if i == m.cursor && manga.Description != "" {
			desc := styles.Truncate(manga.Description, 60)
			b.WriteString("\n    ")
			b.WriteString(styles.ListItemDescStyle.Render(desc))
		}
		b.WriteString("\n")
	}

	// Pagination help
	b.WriteString("\n")
	b.WriteString(styles.RenderDivider(40))
	b.WriteString("\n")
	
	navHelp := "↑/↓ navigate • Enter select • g genre"
	if m.page > 1 {
		navHelp += " • p prev"
	}
	if m.hasNextPage() {
		navHelp += " • n next"
	}
	navHelp += " • r refresh"
	
	b.WriteString(styles.HelpStyle.Render(navHelp))

	return b.String()
}

// renderGenreSelection renders the genre selection overlay
func (m BrowseModel) renderGenreSelection() string {
	var b strings.Builder
	
	b.WriteString(styles.TitleStyle.Render("🎭 Select Genre"))
	b.WriteString("\n\n")
	
	for i, genre := range m.genres {
		prefix := "  "
		style := styles.ListItemStyle
		if i == m.genreCursor {
			prefix = "▸ "
			style = styles.ListItemSelectedStyle
		}
		
		// Highlight currently active genre
		if (i == 0 && m.selectedGenreID == "") || genre == m.selectedGenreName {
			genre = "✓ " + genre
		}
		
		line := fmt.Sprintf("%s%s", prefix, genre)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}
	
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("↑/↓ navigate • Enter select • ESC cancel"))
	
	return b.String()
}

// renderStatus renders manga status badge
func (m BrowseModel) renderStatus(status string) string {
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

// hasNextPage returns true if there are more pages
func (m BrowseModel) hasNextPage() bool {
	return m.page < m.totalPages()
}

// totalPages calculates total pages
func (m BrowseModel) totalPages() int {
	if m.total == 0 {
		return 1
	}
	pages := m.total / m.limit
	if m.total%m.limit > 0 {
		pages++
	}
	return pages
}

// loadManga loads manga list from API with genre filter
func (m BrowseModel) loadManga() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *models.PaginatedResponse[models.Manga]
		var err error
		
		if m.selectedGenreID != "" {
			resp, err = m.apiClient.ListMangaByGenre(ctx, m.selectedGenreID, m.page, m.limit)
		} else {
			resp, err = m.apiClient.ListManga(ctx, m.page, m.limit)
		}
		
		if err != nil {
			return BrowseErrorMsg{Err: err}
		}
		return MangaListLoadedMsg{
			Manga: resp.Data,
			Total: resp.Meta.Total,
		}
	}
}

// GetSelectedManga returns the currently selected manga ID
func (m BrowseModel) GetSelectedManga() string {
	if m.cursor < len(m.manga) {
		return m.manga[m.cursor].ID
	}
	return ""
}

// Messages

// MangaListLoadedMsg is sent when manga list is loaded
type MangaListLoadedMsg struct {
	Manga []models.Manga
	Total int
}

// BrowseErrorMsg is sent on browse errors
type BrowseErrorMsg struct {
	Err error
}

// genreToID converts display genre names to API genre IDs
func genreToID(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, "--", "-")
	s = strings.ReplaceAll(s, "sci-fi", "sci-fi")
	return s
}
