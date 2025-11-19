package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yshuman1/loki/internal/models"
)

type EmailListModel struct {
	emails   []*models.Email
	cursor   int
	selected int
	height   int
}

func NewEmailListModel() *EmailListModel {
	return &EmailListModel{
		emails: make([]*models.Email, 0),
		cursor: 0,
	}
}

func (m *EmailListModel) SetEmails(emails []*models.Email) {
	m.emails = emails
}

func (m *EmailListModel) SetHeight(height int) {
	m.height = height
}

func (m *EmailListModel) Update(msg tea.Msg) (*EmailListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		oldCursor := m.cursor
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.emails)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g":
			m.cursor = 0
		case "G":
			m.cursor = len(m.emails) - 1
		case "enter", " ":
			m.selected = m.cursor
			// Emit EmailSelectedMsg
			if m.cursor < len(m.emails) {
				return m, func() tea.Msg {
					return EmailSelectedMsg{Email: m.emails[m.cursor]}
				}
			}
		}

		// Auto-select email when cursor moves (lazygit-style)
		if m.cursor != oldCursor && m.cursor < len(m.emails) {
			return m, func() tea.Msg {
				return EmailSelectedMsg{Email: m.emails[m.cursor]}
			}
		}
	}
	return m, nil
}

func (m *EmailListModel) View() string {
	if len(m.emails) == 0 {
		return helpStyle.Render("No emails")
	}

	// Calculate how many emails can fit
	// Each email takes 2 lines (from+date line, subject line)
	maxLines := m.height
	if maxLines <= 0 {
		maxLines = 20
	}
	emailsPerScreen := maxLines / 2 // Changed from 3 to 2
	if emailsPerScreen <= 0 {
		emailsPerScreen = 1
	}

	// Calculate scroll offset to keep cursor visible
	scrollOffset := 0
	if m.cursor >= emailsPerScreen {
		scrollOffset = m.cursor - emailsPerScreen + 1
	}

	// Determine visible range
	visibleStart := scrollOffset
	visibleEnd := scrollOffset + emailsPerScreen
	if visibleEnd > len(m.emails) {
		visibleEnd = len(m.emails)
	}

	var b strings.Builder

	for i := visibleStart; i < visibleEnd; i++ {
		email := m.emails[i]

		// Format time
		timeStr := formatEmailTime(email.Date)

		// Priority icon
		priorityIcon := email.Priority.Icon()

		// Truncate subject and from to fit in panel (width ~31 after borders/padding)
		subject := truncate(email.Subject, 22)
		from := truncate(email.From.Address, 15)

		var line string
		if i == m.cursor {
			// Selected style - use more compact single-line format
			line = emailItemSelectedStyle.Render(
				fmt.Sprintf("%s %-15s %s\n%s", priorityIcon, from, timeStr, subject),
			)
		} else {
			style := emailItemStyle
			if !email.Read {
				style = emailItemUnreadStyle
			}
			// Unselected style - use compact single-line format
			line = style.Render(
				fmt.Sprintf("%s %-15s %s\n%s", priorityIcon, from, timeStr, subject),
			)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func formatEmailTime(t time.Time) string {
	now := time.Now()
	if t.Day() == now.Day() {
		return t.Format("3:04PM")
	} else if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}
	return t.Format("2006")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m *EmailListModel) GetSelectedEmail() *models.Email {
	if len(m.emails) == 0 || m.cursor < 0 || m.cursor >= len(m.emails) {
		return nil
	}
	return m.emails[m.cursor]
}

func (m *EmailListModel) RemoveEmail(id string) {
	for i, email := range m.emails {
		if email.ID == id {
			// Remove from slice
			m.emails = append(m.emails[:i], m.emails[i+1:]...)

			// Adjust cursor if needed
			if m.cursor >= len(m.emails) && m.cursor > 0 {
				m.cursor--
			}
			return
		}
	}
}
