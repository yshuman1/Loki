package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	colorPrimary   = lipgloss.Color("#00D9FF") // Cyan
	colorSecondary = lipgloss.Color("#FF00FF") // Magenta
	colorSuccess   = lipgloss.Color("#00FF00") // Green
	colorWarning   = lipgloss.Color("#FFFF00") // Yellow
	colorDanger    = lipgloss.Color("#FF0000") // Red
	colorMuted     = lipgloss.Color("#666666") // Gray
	colorText      = lipgloss.Color("#FFFFFF") // White
	colorBg        = lipgloss.Color("#000000") // Black
	colorBgLight   = lipgloss.Color("#1a1a1a") // Dark gray (formerly colorBg)
	colorBorder    = lipgloss.Color("#444444") // Border gray

	// Priority colors
	colorPriorityLow    = lipgloss.Color("#888888")
	colorPriorityMed    = lipgloss.Color("#FFA500")
	colorPriorityHigh   = lipgloss.Color("#FF6B6B")
	colorPriorityUrgent = lipgloss.Color("#FF0000")

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBg)

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	// Border styles
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Tree view styles
	treeStyle = lipgloss.NewStyle().
			Padding(0, 1)

	treeNodeStyle = lipgloss.NewStyle().
			Foreground(colorText)

	treeNodeSelectedStyle = lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorPrimary).
				Bold(true)

	treeNodeExpandedStyle = lipgloss.NewStyle().
				Foreground(colorSecondary)

	treeIndentStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Email list styles
	emailListStyle = lipgloss.NewStyle().
			Padding(0, 1)

	emailItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	emailItemSelectedStyle = lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorPrimary).
				Bold(true).
				Padding(0, 1)

	emailItemUnreadStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText)

	emailItemReadStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	emailSubjectStyle = lipgloss.NewStyle().
				Foreground(colorText)

	emailFromStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	emailTimeStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	emailPreviewStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	// Preview pane styles
	previewStyle = lipgloss.NewStyle().
			Padding(0, 1)

	previewHeaderStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true)

	previewFromStyle = lipgloss.NewStyle().
				Foreground(colorSecondary)

	previewSubjectStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true)

	previewBodyStyle = lipgloss.NewStyle().
				Foreground(colorText)

	previewMetaStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	// Claude styles
	claudeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorSecondary).
			Padding(1, 2).
			MarginTop(1)

	claudeTitleStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	claudeTextStyle = lipgloss.NewStyle().
			Foreground(colorText)

	claudeUserStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	claudeAssistantStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	// Meeting/Calendar styles
	meetingBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSuccess).
			Padding(0, 1).
			MarginTop(1)

	meetingTitleStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	meetingTimeStyle = lipgloss.NewStyle().
				Foreground(colorWarning)

	meetingAttendeesStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	// Calendar view styles
	calendarDayStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Align(lipgloss.Center)

	calendarTodayStyle = lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorPrimary).
				Bold(true).
				Padding(0, 1).
				Align(lipgloss.Center)

	calendarEventStyle = lipgloss.NewStyle().
				Background(colorBgLight).
				Foreground(colorText).
				Padding(0, 1)

	// Status bar styles
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBg).
			Padding(0, 1)

	statusBarKeyStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	statusBarSeparatorStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	// Modal/Popup styles
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			Background(colorBg)

	modalTitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Underline(true)

	// Input styles
	inputStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBgLight).
			Padding(0, 1)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	inputFocusedStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorBgLight).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Button styles
	buttonStyle = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorPrimary).
			Padding(0, 2).
			Bold(true)

	buttonInactiveStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorBgLight).
				Padding(0, 2)

	// Priority badge styles
	priorityHighStyle = lipgloss.NewStyle().
				Foreground(colorPriorityHigh).
				Bold(true)

	priorityMedStyle = lipgloss.NewStyle().
				Foreground(colorPriorityMed).
				Bold(true)

	priorityLowStyle = lipgloss.NewStyle().
				Foreground(colorPriorityLow)

	// Help text styles
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Error/Success message styles
	errorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	// Divider styles
	dividerStyle = lipgloss.NewStyle().
			Foreground(colorBorder)
)

// Helper functions for rendering

func RenderPriority(priority string) string {
	switch priority {
	case "HIGH", "URGENT":
		return priorityHighStyle.Render("⚡ " + priority)
	case "MED", "MEDIUM":
		return priorityMedStyle.Render("⚠  " + priority)
	case "LOW":
		return priorityLowStyle.Render("📧 " + priority)
	default:
		return "  "
	}
}

func RenderDivider(width int) string {
	return dividerStyle.Render(lipgloss.NewStyle().Width(width).Render("─"))
}

func RenderTitle(title string) string {
	return lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(title)
}

func RenderSubtitle(subtitle string) string {
	return lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(subtitle)
}

func RenderError(msg string) string {
	return errorStyle.Render("✗ " + msg)
}

func RenderSuccess(msg string) string {
	return successStyle.Render("✓ " + msg)
}

func RenderWarning(msg string) string {
	return warningStyle.Render("⚠ " + msg)
}
