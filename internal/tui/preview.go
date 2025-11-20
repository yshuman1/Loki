package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yshuman1/loki/internal/email"
	"github.com/yshuman1/loki/internal/models"
)

type PreviewModel struct {
	email    *models.Email
	viewport viewport.Model
	width    int
	height   int
}

func NewPreviewModel() *PreviewModel {
	return &PreviewModel{
		viewport: viewport.New(0, 0),
	}
}

func (m *PreviewModel) SetEmail(email *models.Email) {
	m.email = email
	m.renderContent()
}

func (m *PreviewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.renderContent()
}

func (m *PreviewModel) renderContent() {
	if m.email == nil {
		m.viewport.SetContent(helpStyle.Render("Select an email to preview"))
		return
	}

	// Render the body using the new renderer
	// We pass the viewport width (minus padding) to ensure proper wrapping
	renderWidth := m.viewport.Width - 4
	if renderWidth < 40 {
		renderWidth = 40 // Minimum width
	}

	body, _ := email.RenderBody(m.email.Body, renderWidth, m.email.BodyParams)

	// Render headers
	headers := email.RenderHeaders(
		m.email.From.Address,
		strings.Join(func() []string {
			var addrs []string
			for _, t := range m.email.To {
				addrs = append(addrs, t.Address)
			}
			return addrs
		}(), ", "),
		m.email.Subject,
		m.email.Date.Format("Mon Jan 02, 2006 3:04 PM"),
		renderWidth,
	)

	var b strings.Builder
	b.WriteString(headers)
	b.WriteString("\n\n")
	b.WriteString(body)
	b.WriteString("\n\n")

	// Claude summary if available
	if m.email.Summary != "" {
		b.WriteString(RenderDivider(m.width) + "\n")
		b.WriteString(claudeBoxStyle.Width(m.width - 4).Render(
			claudeTitleStyle.Render("🤖 Claude Summary") + "\n\n" +
				claudeTextStyle.Render(m.email.Summary),
		))
		b.WriteString("\n")
	}

	// Meeting detection
	if m.email.HasMeeting && m.email.Meeting != nil {
		b.WriteString("\n")
		b.WriteString(meetingBoxStyle.Width(m.width - 4).Render(
			meetingTitleStyle.Render("📅 Meeting Detected") + "\n" +
				meetingTimeStyle.Render(m.email.Meeting.Title) + "\n" +
				meetingTimeStyle.Render(m.email.Meeting.StartTime.Format("Mon Jan 2, 3:04 PM")),
		))
	}

	m.viewport.SetContent(b.String())
	m.viewport.GotoTop()
}

func (m *PreviewModel) Update(msg tea.Msg) (*PreviewModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *PreviewModel) View() string {
	return m.viewport.View()
}
