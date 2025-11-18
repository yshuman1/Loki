package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/loki/internal/models"
)

// CalendarModel - stub for calendar view
type CalendarModel struct {
	meetings []*models.Meeting
	viewMode string // month, week, day
}

func NewCalendarModel() *CalendarModel {
	return &CalendarModel{
		meetings: make([]*models.Meeting, 0),
		viewMode: "month",
	}
}

func (m *CalendarModel) Update(msg tea.Msg) (*CalendarModel, tea.Cmd) {
	// TODO: Implement calendar navigation
	return m, nil
}

func (m *CalendarModel) View() string {
	return claudeBoxStyle.Render("Calendar View (Coming Soon)")
}

// ClaudeChatModel - stub for Claude chat overlay
type ClaudeChatModel struct {
	messages []models.ClaudeMessage
	input    string
}

func NewClaudeChatModel() *ClaudeChatModel {
	return &ClaudeChatModel{
		messages: make([]models.ClaudeMessage, 0),
	}
}

func (m *ClaudeChatModel) Update(msg tea.Msg) (*ClaudeChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			// Close chat - handled by parent
			return m, nil
		}
	}
	return m, nil
}

func (m *ClaudeChatModel) View() string {
	return modalTitleStyle.Render("🤖 Claude Chat") + "\n\n" +
		helpStyle.Render("Chat with Claude about your emails\n\n") +
		helpStyle.Render("Press Esc to close")
}

// SchedulerModel - stub for meeting scheduler
type SchedulerModel struct {
	email     *models.Email
	title     string
	date      string
	time      string
	duration  int
	location  string
	attendees []*models.EmailAddress
	focused   int
}

func NewSchedulerModel() *SchedulerModel {
	return &SchedulerModel{
		duration: 30,
	}
}

func (m *SchedulerModel) SetFromEmail(email *models.Email) {
	m.email = email
	// Pre-fill from email
	m.title = "RE: " + email.Subject
	m.attendees = append([]*models.EmailAddress{email.From}, email.To...)
}

func (m *SchedulerModel) Update(msg tea.Msg) (*SchedulerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			// Close scheduler - handled by parent
			return m, nil
		case "enter":
			// TODO: Create meeting
			return m, nil
		}
	}
	return m, nil
}

func (m *SchedulerModel) View() string {
	title := modalTitleStyle.Render("📅 Schedule Meeting") + "\n\n"
	
	content := inputLabelStyle.Render("Title: ") + m.title + "\n\n"
	
	if len(m.attendees) > 0 {
		content += inputLabelStyle.Render("Attendees:\n")
		for _, attendee := range m.attendees {
			content += "  • " + attendee.Address + "\n"
		}
	}
	
	content += "\n"
	content += inputLabelStyle.Render("When: ") + "[t]omorrow [w]eek [c]ustom\n"
	content += inputLabelStyle.Render("Duration: ") + "30 min\n"
	content += inputLabelStyle.Render("Location: ") + "[z]oom [o]ffice [c]ustom\n"
	content += "\n"
	content += helpStyle.Render("[Enter] Create  [Esc] Cancel")
	
	return title + content
}
