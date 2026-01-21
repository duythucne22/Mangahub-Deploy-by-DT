package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui/api"
	"mangahub/internal/tui/grpc"
	"mangahub/internal/tui/styles"
	"mangahub/pkg/models"
)

// SearchModel handles manga search
type SearchModel struct {
	apiClient     *api.Client
	grpcClient    *grpc.Client
	
	// Input
	searchInput   textinput.Model
	lastQuery     string
	searchDelay   *time.Timer
	
	// Results
	results       []models.Manga
	total         int
	streaming     bool // true when using gRPC streaming
	
	// State
	loading       bool
	err           error
	hasSearched   bool
	cursor        int
	inputFocused  bool
	
	// Pagination
	page          int
	limit         int
	
	// Window size
	width         int
	height        int
}

// NewSearchModel creates a new search model with gRPC support
func NewSearchModel(apiClient *api.Client, grpcClient *grpc.Client) SearchModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Type to search (real-time)..."
	searchInput.CharLimit = 100
	searchInput.Width = 50
	searchInput.Focus()
	
	return SearchModel{
		apiClient:    apiClient,
		grpcClient:   grpcClient,
		searchInput:  searchInput,
		inputFocused: true,
		page:         1,
		limit:        20,
		streaming:    grpcClient != nil,
	}
}

// Init initializes the model
func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.inputFocused {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Force immediate search on Enter
				if m.searchInput.Value() != "" {
					m.loading = true
					m.hasSearched = true
					m.page = 1
					return m, m.doSearch()
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
				if len(m.results) > 0 {
					m.inputFocused = false
					m.searchInput.Blur()
				}
				return m, nil
			}
			
			// Update search input
			var cmd tea.Cmd
			oldValue := m.searchInput.Value()
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)
			
			// Real-time search if using gRPC and query changed
			if m.streaming && m.searchInput.Value() != oldValue {
				newQuery := m.searchInput.Value()
				
				// Only search if query is at least 2 characters
				if len(newQuery) >= 2 {
					// Debounce: wait 300ms before searching
					if m.searchDelay != nil {
						m.searchDelay.Stop()
					}
					
					m.searchDelay = time.AfterFunc(300*time.Millisecond, func() {
						// This will trigger a search after delay
					})
					
					// Start streaming search immediately
					return m, m.doStreamingSearch()
				} else if len(newQuery) == 0 {
					// Clear results when query is cleared
					m.results = nil
					m.total = 0
					m.hasSearched = false
				}
			}
		} else {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("/"))):
				m.inputFocused = true
				m.searchInput.Focus()
				return m, textinput.Blink
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
				m.cursor++
				if m.cursor >= len(m.results) {
					m.cursor = len(m.results) - 1
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
				m.cursor--
				if m.cursor < 0 {
					m.cursor = 0
					m.inputFocused = true
					m.searchInput.Focus()
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("n", "pgdown"))):
				if m.hasNextPage() {
					m.page++
					m.cursor = 0
					m.loading = true
					return m, m.doSearch()
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("p", "pgup"))):
				if m.page > 1 {
					m.page--
					m.cursor = 0
					m.loading = true
					return m, m.doSearch()
				}
				return m, nil
				
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if len(m.results) > 0 {
					return m, func() tea.Msg {
						return SelectMangaMsg{MangaID: m.results[m.cursor].ID}
					}
				}
				return m, nil
			}
		}

	case SearchResultsMsg:
		m.loading = false
		m.results = msg.Results
		m.total = msg.Total
		m.cursor = 0
		return m, nil

	case SearchErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// View renders the search view
func (m SearchModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.TitleStyle.Render("🔍 Search Manga"))
	b.WriteString("\n\n")

	// Search input
	inputStyle := styles.InputStyle
	if m.inputFocused {
		inputStyle = styles.InputFocusedStyle
	}
	b.WriteString(inputStyle.Render("Search: "))
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	// Loading state
	if m.loading {
		b.WriteString(styles.SpinnerStyle.Render("⟳ "))
		b.WriteString(styles.InfoStyle.Render("Searching..."))
		return b.String()
	}

	// Error state
	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	// No search yet
	if !m.hasSearched {
		b.WriteString(styles.HelpStyle.Render("Enter a search term and press Enter"))
		return b.String()
	}

	// No results
	if len(m.results) == 0 {
		b.WriteString(styles.InfoStyle.Render("No results found for: "))
		b.WriteString(styles.HighlightStyle.Render(m.searchInput.Value()))
		return b.String()
	}

	// Results header
	pageInfo := fmt.Sprintf("Found %d results (Page %d/%d)", m.total, m.page, m.totalPages())
	b.WriteString(styles.SubtitleStyle.Render(pageInfo))
	b.WriteString("\n")
	b.WriteString(styles.RenderDivider(40))
	b.WriteString("\n\n")

	// Results list
	for i, manga := range m.results {
		prefix := "  "
		style := styles.ListItemStyle
		if i == m.cursor && !m.inputFocused {
			prefix = "▸ "
			style = styles.ListItemSelectedStyle
		}
		
		title := styles.ListItemTitleStyle.Render(styles.Truncate(manga.Title, 40))
		status := m.renderStatus(manga.Status)
		
		line := fmt.Sprintf("%s%s %s", prefix, title, status)
		b.WriteString(style.Render(line))
		
		// Description for selected
		if i == m.cursor && !m.inputFocused && manga.Description != "" {
			desc := styles.Truncate(manga.Description, 60)
			b.WriteString("\n    ")
			b.WriteString(styles.ListItemDescStyle.Render(desc))
		}
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	help := "/ focus search • ↑/↓ navigate • Enter select"
	if m.page > 1 {
		help += " • p prev"
	}
	if m.hasNextPage() {
		help += " • n next"
	}
	b.WriteString(styles.HelpStyle.Render(help))

	return b.String()
}

// renderStatus renders manga status badge
func (m SearchModel) renderStatus(status string) string {
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
func (m SearchModel) hasNextPage() bool {
	return m.page < m.totalPages()
}

// totalPages calculates total pages
func (m SearchModel) totalPages() int {
	if m.total == 0 {
		return 1
	}
	pages := m.total / m.limit
	if m.total%m.limit > 0 {
		pages++
	}
	return pages
}

// doSearch performs the search API call (fallback or pagination)
func (m SearchModel) doSearch() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.apiClient.SearchManga(ctx, m.searchInput.Value(), m.page, m.limit)
		if err != nil {
			return SearchErrorMsg{Err: err}
		}
		return SearchResultsMsg{
			Results: resp.Data,
			Total:   resp.Meta.Total,
		}
	}
}

// doStreamingSearch performs real-time gRPC streaming search
func (m SearchModel) doStreamingSearch() tea.Cmd {
	query := m.searchInput.Value()
	
	return func() tea.Msg {
		if m.grpcClient == nil {
			// Fallback to HTTP if gRPC not available
			ctx := context.Background()
			resp, err := m.apiClient.SearchManga(ctx, query, 1, m.limit)
			if err != nil {
				return SearchErrorMsg{Err: err}
			}
			return SearchResultsMsg{
				Results: resp.Data,
				Total:   resp.Meta.Total,
			}
		}

		// Use gRPC streaming for fast results
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		results, err := m.grpcClient.StreamSearchResults(ctx, query)
		if err != nil {
			return SearchErrorMsg{Err: err}
		}

		return SearchResultsMsg{
			Results: results,
			Total:   len(results),
			IsStream: true,
		}
	}
}

// GetSelectedManga returns the currently selected manga ID
func (m SearchModel) GetSelectedManga() string {
	if m.cursor < len(m.results) {
		return m.results[m.cursor].ID
	}
	return ""
}

// SetQuery sets the search query programmatically
func (m *SearchModel) SetQuery(query string) {
	m.searchInput.SetValue(query)
}

// Messages

// SearchResultsMsg is sent when search completes
type SearchResultsMsg struct {
	Results  []models.Manga
	Total    int
	IsStream bool // indicates if from gRPC stream
}

// SearchErrorMsg is sent on search errors
type SearchErrorMsg struct {
	Err error
}
