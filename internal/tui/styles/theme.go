package styles

import "github.com/charmbracelet/lipgloss"

// Dracula color palette
const (
	Background  = "#282a36"
	CurrentLine = "#44475a"
	Foreground  = "#f8f8f2"
	Comment     = "#6272a4"
	Cyan        = "#8be9fd"
	Green       = "#50fa7b"
	Orange      = "#ffb86c"
	Pink        = "#ff79c6"
	Purple      = "#bd93f9"
	Red         = "#ff5555"
	Yellow      = "#f1fa8c"
)

var (
	// App-level styles
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Background(lipgloss.Color(Background)).
			Foreground(lipgloss.Color(Foreground))

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(Purple)).
			Background(lipgloss.Color(Background)).
			Padding(0, 1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Cyan)).
			Background(lipgloss.Color(Background))

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Foreground)).
			Background(lipgloss.Color(CurrentLine)).
			Padding(0, 1)

	StatusBarActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Green)).
				Background(lipgloss.Color(CurrentLine)).
				Bold(true).
				Padding(0, 1)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Foreground)).
			Background(lipgloss.Color(CurrentLine)).
			Padding(0, 1)

	InputFocusedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Pink)).
				Background(lipgloss.Color(CurrentLine)).
				Bold(true).
				Padding(0, 1)

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Purple)).
				Bold(true)

	// List styles
	ListItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Foreground)).
			PaddingLeft(2)

	ListItemSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Pink)).
				Background(lipgloss.Color(CurrentLine)).
				Bold(true).
				PaddingLeft(1).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color(Purple))

	ListItemTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Cyan)).
				Bold(true)

	ListItemDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Comment))

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Foreground)).
			Background(lipgloss.Color(CurrentLine)).
			Padding(0, 2).
			MarginRight(2)

	ButtonActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Background)).
				Background(lipgloss.Color(Purple)).
				Bold(true).
				Padding(0, 2).
				MarginRight(2)

	ButtonSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Background)).
				Background(lipgloss.Color(Green)).
				Bold(true).
				Padding(0, 2).
				MarginRight(2)

	// Card/Box styles
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(Purple)).
			Padding(1, 2).
			MarginBottom(1)

	CardTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Pink)).
			Bold(true)

	CardContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Foreground))

	// Info/Alert styles
	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Cyan)).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Green)).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Yellow)).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Red)).
			Bold(true)

	// Help/Hints styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Comment)).
			Italic(true)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Purple)).
			Bold(true)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Pink)).
				Background(lipgloss.Color(CurrentLine)).
				Bold(true).
				Padding(0, 1).
				Align(lipgloss.Center)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Foreground)).
			Padding(0, 1)

	TableRowSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Yellow)).
				Background(lipgloss.Color(CurrentLine)).
				Bold(true)

	// Badge styles
	BadgePrimaryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Background)).
				Background(lipgloss.Color(Purple)).
				Bold(true).
				Padding(0, 1)

	BadgeSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Background)).
				Background(lipgloss.Color(Green)).
				Bold(true).
				Padding(0, 1)

	BadgeWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Background)).
				Background(lipgloss.Color(Yellow)).
				Bold(true).
				Padding(0, 1)

	BadgeDangerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Background)).
				Background(lipgloss.Color(Red)).
				Bold(true).
				Padding(0, 1)

	// Link/Highlight styles
	LinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Cyan)).
			Underline(true)

	HighlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Yellow)).
			Bold(true)

	// Divider/Border styles
	DividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(CurrentLine))

	// Spinner styles
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Purple))

	// Progress bar styles
	ProgressBarFilled = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Green))

	ProgressBarEmpty = lipgloss.NewStyle().
				Foreground(lipgloss.Color(CurrentLine))

	// Metadata/Stats styles
	MetaKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Purple)).
			Bold(true)

	MetaValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Cyan))

	// Rating styles
	RatingStarFilledStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Yellow)).
				Bold(true)

	RatingStarEmptyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Comment))

	// Tab styles
	TabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Comment)).
			Padding(0, 2)

	TabActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Pink)).
			Background(lipgloss.Color(CurrentLine)).
			Bold(true).
			Padding(0, 2)

	// Dialog/Modal styles
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(Pink)).
			Padding(1, 2).
			Background(lipgloss.Color(Background))

	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(Pink)).
				Bold(true).
				Align(lipgloss.Center)
)

// Helper functions for common operations

// Truncate truncates text to maxLen and adds "..." if needed
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// RenderStars renders a star rating
func RenderStars(rating float64, max int) string {
	filled := int(rating)
	result := ""
	for i := 0; i < max; i++ {
		if i < filled {
			result += RatingStarFilledStyle.Render("★")
		} else {
			result += RatingStarEmptyStyle.Render("★")
		}
	}
	return result
}

// RenderDivider renders a horizontal divider
func RenderDivider(width int) string {
	divider := ""
	for i := 0; i < width; i++ {
		divider += "─"
	}
	return DividerStyle.Render(divider)
}

// RenderProgressBar renders a progress bar
func RenderProgressBar(current, total, width int) string {
	if total == 0 {
		return ProgressBarEmpty.Render(lipgloss.NewStyle().Width(width).Render(""))
	}

	percentage := float64(current) / float64(total)
	filledWidth := int(float64(width) * percentage)

	filled := ""
	for i := 0; i < filledWidth; i++ {
		filled += "█"
	}

	empty := ""
	for i := filledWidth; i < width; i++ {
		empty += "░"
	}

	return ProgressBarFilled.Render(filled) + ProgressBarEmpty.Render(empty)
}

// RenderKeyValue renders a key-value pair with styling
func RenderKeyValue(key, value string) string {
	return MetaKeyStyle.Render(key+":") + " " + MetaValueStyle.Render(value)
}
