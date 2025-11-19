package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yshuman1/loki/internal/models"
)

type PreviewModel struct {
	email  *models.Email
	scroll int
	height int
}

func NewPreviewModel() *PreviewModel {
	return &PreviewModel{}
}

func (m *PreviewModel) SetEmail(email *models.Email) {
	m.email = email
	m.scroll = 0
}

func (m *PreviewModel) Update(msg tea.Msg) (*PreviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.scroll++
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
		case "g":
			m.scroll = 0
		}
	}
	return m, nil
}

func (m *PreviewModel) View() string {
	if m.email == nil {
		return helpStyle.Render("Select an email to preview")
	}

	var b strings.Builder

	// Headers
	b.WriteString(previewFromStyle.Render("From: ") + m.email.From.Address + "\n")
	b.WriteString(previewMetaStyle.Render("To: ") + m.email.To[0].Address + "\n")
	b.WriteString(previewSubjectStyle.Render("Subject: ") + m.email.Subject + "\n")
	b.WriteString(previewMetaStyle.Render("Date: ") + m.email.Date.Format("Mon Jan 2, 2006 3:04 PM") + "\n")
	b.WriteString("\n")
	b.WriteString(RenderDivider(60))
	b.WriteString("\n")

	// Body
	body := m.email.Body
	if body == "" {
		body = "No content"
	}
	b.WriteString(previewBodyStyle.Render(body))
	b.WriteString("\n\n")

	// Claude summary if available
	if m.email.Summary != "" {
		b.WriteString(RenderDivider(60))
		b.WriteString("\n")
		b.WriteString(claudeBoxStyle.Render(
			claudeTitleStyle.Render("🤖 Claude Summary") + "\n\n" +
				claudeTextStyle.Render(m.email.Summary),
		))
	}

	// Meeting detection
	if m.email.HasMeeting && m.email.Meeting != nil {
		b.WriteString("\n")
		b.WriteString(meetingBoxStyle.Render(
			meetingTitleStyle.Render("📅 Meeting Detected") + "\n" +
				meetingTimeStyle.Render(m.email.Meeting.Title) + "\n" +
				meetingTimeStyle.Render(m.email.Meeting.StartTime.Format("Mon Jan 2, 3:04 PM")),
		))
	}

	return b.String()
}
