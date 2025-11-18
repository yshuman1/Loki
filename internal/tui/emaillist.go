package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/loki/internal/models"
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

func (m *EmailListModel) Update(msg tea.Msg) (*EmailListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
			// TODO: Emit EmailSelectedMsg
		}
	}
	return m, nil
}

func (m *EmailListModel) View() string {
	if len(m.emails) == 0 {
		return helpStyle.Render("No emails")
	}

	var b strings.Builder

	for i, email := range m.emails {
		// Format time
		timeStr := formatEmailTime(email.Date)

		// Priority icon
		priorityIcon := email.Priority.Icon()

		// Truncate subject
		subject := truncate(email.Subject, 25)
		from := truncate(email.From.Address, 20)

		var line string
		if i == m.cursor {
			line = emailItemSelectedStyle.Render(
				fmt.Sprintf("%s %s %4s\n   %s", 
					priorityIcon, from, timeStr, subject),
			)
		} else {
			style := emailItemStyle
			if !email.Read {
				style = emailItemUnreadStyle
			}
			line = style.Render(
				fmt.Sprintf("%s %s %4s\n   %s",
					priorityIcon, from, timeStr, subject),
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
