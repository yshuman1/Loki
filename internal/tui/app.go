package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yshuman1/loki/internal/config"
	"github.com/yshuman1/loki/internal/email"
	"github.com/yshuman1/loki/internal/models"
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeEmail ViewMode = iota
	ViewModeCalendar
)

// FocusPanel represents which panel has focus
type FocusPanel int

const (
	FocusPanelTree FocusPanel = iota
	FocusPanelList
	FocusPanelPreview
)

// Model is the main application model
type Model struct {
	// View state
	viewMode   ViewMode
	focusPanel FocusPanel
	width      int
	height     int

	// Data
	accounts       []*models.Account
	currentEmail   *models.Email
	currentMeeting *models.Meeting
	emails         []*models.Email
	meetings       []*models.Meeting

	// UI components
	tree       *TreeModel
	emailList  *EmailListModel
	preview    *PreviewModel
	calendar   *CalendarModel
	claudeChat *ClaudeChatModel
	scheduler  *SchedulerModel

	// State
	loading        bool
	error          string
	statusMessage  string
	showClaudeChat bool
	showScheduler  bool

	// Services
	emailManager    *email.Manager
	calendarService interface{} // Will be *calendar.Client
	claudeService   interface{} // Will be *agent.ClaudeService
	config          *config.Config
}

// NewModel creates a new TUI model
func NewModel() *Model {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Create with empty config if load fails
		cfg = &config.Config{
			Accounts: []config.AccountConfig{},
		}
	}

	// Convert config accounts to models
	accounts := cfg.ToModels()

	// Initialize email manager
	emailMgr := email.NewManager(accounts)

	m := &Model{
		viewMode:     ViewModeEmail,
		focusPanel:   FocusPanelTree,
		accounts:     accounts,
		emails:       make([]*models.Email, 0),
		meetings:     make([]*models.Meeting, 0),
		emailManager: emailMgr,
		config:       cfg,
	}

	// Initialize sub-models
	m.tree = NewTreeModel()
	m.emailList = NewEmailListModel()
	m.preview = NewPreviewModel()
	m.calendar = NewCalendarModel()
	m.claudeChat = NewClaudeChatModel()
	m.scheduler = NewSchedulerModel()

	return m
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	m.statusMessage = "Connecting to email server..."
	return tea.Batch(
		tea.EnterAltScreen,
		m.connectAndLoad,
	)
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keybindings
		switch msg.String() {
		case "ctrl+c", "q":
			if m.showClaudeChat {
				m.showClaudeChat = false
				return m, nil
			}
			if m.showScheduler {
				m.showScheduler = false
				return m, nil
			}
			return m, tea.Quit

		case "E":
			// Switch to email mode
			m.viewMode = ViewModeEmail
			m.statusMessage = "Email mode"
			return m, nil

		case "C":
			// Switch to calendar mode
			m.viewMode = ViewModeCalendar
			m.statusMessage = "Calendar mode"
			return m, nil

		case "c":
			// Open Claude chat
			if !m.showScheduler {
				m.showClaudeChat = !m.showClaudeChat
				if m.showClaudeChat {
					m.statusMessage = "Claude chat opened"
				}
			}
			return m, nil

		case "s":
			// Open scheduler (if email is selected)
			if m.currentEmail != nil && !m.showClaudeChat {
				m.showScheduler = !m.showScheduler
				if m.showScheduler {
					// Pre-fill scheduler with email data
					m.scheduler.SetFromEmail(m.currentEmail)
					m.statusMessage = "Scheduling meeting..."
				}
			}
			return m, nil

		case "t":
			// Triage inbox
			m.statusMessage = "Triaging inbox..."
			return m, m.triageInbox

		case "tab":
			// Cycle through panels
			m.cycleFocus()
			return m, nil

		case "h", "left":
			// Move focus left or collapse
			if m.focusPanel == FocusPanelList {
				m.focusPanel = FocusPanelTree
			} else if m.focusPanel == FocusPanelPreview {
				m.focusPanel = FocusPanelList
			}
			return m, nil

		case "l", "right":
			// Move focus right or expand
			if m.focusPanel == FocusPanelTree {
				m.focusPanel = FocusPanelList
			} else if m.focusPanel == FocusPanelList {
				m.focusPanel = FocusPanelPreview
			}
			return m, nil

		case "a":
			// Archive email
			if m.focusPanel == FocusPanelList {
				if email := m.emailList.GetSelectedEmail(); email != nil {
					m.statusMessage = fmt.Sprintf("Archiving %s...", truncate(email.Subject, 20))
					return m, m.archiveEmail(email.ID)
				}
			}
			return m, nil

		case "d":
			// Delete email
			if m.focusPanel == FocusPanelList {
				if email := m.emailList.GetSelectedEmail(); email != nil {
					m.statusMessage = fmt.Sprintf("Deleting %s...", truncate(email.Subject, 20))
					return m, m.deleteEmail(email.ID)
				}
			}
			return m, nil

		case "r":
			// Mark read/unread
			if m.focusPanel == FocusPanelList {
				if email := m.emailList.GetSelectedEmail(); email != nil {
					newReadStatus := !email.Read
					statusStr := "read"
					if !newReadStatus {
						statusStr = "unread"
					}
					m.statusMessage = fmt.Sprintf("Marking as %s...", statusStr)
					return m, m.markRead(email.ID, newReadStatus)
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case AccountsLoadedMsg:
		m.accounts = msg.Accounts
		// Tree already updated in connectAndLoad
		m.statusMessage = fmt.Sprintf("Connected. Ready to use. (%d account(s))", len(msg.Accounts))
		return m, nil

	case FoldersLoadedMsg:
		m.tree.SetFolders(msg.AccountID, msg.Folders)
		m.statusMessage = fmt.Sprintf("Loaded %d folders. Press Enter on a folder to view emails.", len(msg.Folders))
		return m, nil

	case EmailsLoadedMsg:
		m.emails = msg.Emails
		m.emailList.SetEmails(msg.Emails)
		m.statusMessage = fmt.Sprintf("Loaded %d emails. Use j/k to navigate.", len(msg.Emails))

		// Auto-select first email (lazygit-style)
		if len(msg.Emails) > 0 {
			m.currentEmail = msg.Emails[0]
			m.preview.SetEmail(msg.Emails[0])
		}
		return m, nil

	case EmailSelectedMsg:
		m.currentEmail = msg.Email
		m.preview.SetEmail(msg.Email)
		m.statusMessage = "Fetching email body..."
		// Fetch full body asynchronously
		return m, m.loadEmailBody(msg.Email)

	case ErrorMsg:
		m.error = msg.Error
		m.statusMessage = "Error: " + msg.Error
		return m, nil

	case EmailBodyLoadedMsg:
		// Update current email if it matches
		if m.currentEmail != nil && m.currentEmail.ID == msg.EmailID {
			m.currentEmail.Body = msg.TextBody
			m.currentEmail.BodyHTML = msg.HTMLBody
			m.preview.SetEmail(m.currentEmail)
			m.statusMessage = "Email loaded."
		}
		return m, nil

	case FolderSelectedMsg:
		m.statusMessage = fmt.Sprintf("Loading emails from %s...", msg.FolderName)
		return m, m.loadEmailsForFolder(msg.AccountID, msg.FolderName)

	case AccountExpandedMsg:
		m.statusMessage = fmt.Sprintf("Loading folders...")
		return m, m.loadFoldersForAccount(msg.AccountID)

	case EmailArchivedMsg:
		m.emailList.RemoveEmail(msg.ID)
		// Also remove from our local slice
		for i, e := range m.emails {
			if e.ID == msg.ID {
				m.emails = append(m.emails[:i], m.emails[i+1:]...)
				break
			}
		}
		m.statusMessage = "Email archived"
		return m, nil

	case EmailDeletedMsg:
		m.emailList.RemoveEmail(msg.ID)
		// Also remove from our local slice
		for i, e := range m.emails {
			if e.ID == msg.ID {
				m.emails = append(m.emails[:i], m.emails[i+1:]...)
				break
			}
		}
		m.statusMessage = "Email deleted"
		return m, nil

	case EmailMarkedReadMsg:
		// Update in list
		for _, e := range m.emails {
			if e.ID == msg.ID {
				e.Read = msg.Read
				break
			}
		}
		m.emailList.SetEmails(m.emails)
		statusStr := "read"
		if !msg.Read {
			statusStr = "unread"
		}
		m.statusMessage = fmt.Sprintf("Email marked as %s", statusStr)
		return m, nil
	}

	// Route updates to focused component
	if m.showClaudeChat {
		var cmd tea.Cmd
		m.claudeChat, cmd = m.claudeChat.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.showScheduler {
		var cmd tea.Cmd
		m.scheduler, cmd = m.scheduler.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		switch m.viewMode {
		case ViewModeEmail:
			cmds = append(cmds, m.updateEmailMode(msg)...)
		case ViewModeCalendar:
			var cmd tea.Cmd
			m.calendar, cmd = m.calendar.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateEmailMode(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	switch m.focusPanel {
	case FocusPanelTree:
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		cmds = append(cmds, cmd)

	case FocusPanelList:
		var cmd tea.Cmd
		m.emailList, cmd = m.emailList.Update(msg)
		cmds = append(cmds, cmd)

	case FocusPanelPreview:
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		cmds = append(cmds, cmd)
	}

	return cmds
}

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Show overlay if active
	if m.showClaudeChat {
		return m.renderWithOverlay(m.claudeChat.View())
	}
	if m.showScheduler {
		return m.renderWithOverlay(m.scheduler.View())
	}

	// Render based on view mode
	var mainView string
	switch m.viewMode {
	case ViewModeEmail:
		mainView = m.renderEmailView()
	case ViewModeCalendar:
		mainView = m.renderCalendarView()
	}

	// Add header
	header := m.renderHeader()

	// Add status bar
	statusBar := m.renderStatusBar()

	// Calculate content height
	// Height - header (1) - status bar (1)
	contentHeight := m.height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}

	return baseStyle.
		Width(m.width).
		Height(m.height).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			mainView,
			statusBar,
		))
}

func (m *Model) renderHeader() string {
	return headerStyle.
		Width(28).
		Padding(1, 1, 0, 1).
		Render("LOKI")
}

func (m *Model) renderEmailView() string {
	// Calculate panel widths
	treeWidth := 30
	listWidth := 35
	previewWidth := m.width - treeWidth - listWidth
	if previewWidth < 0 {
		previewWidth = 0
	}

	// Add top margin and adjust heights for cleaner borders
	// Total height - header (2) - status bar (1)
	panelHeight := m.height - 3
	if panelHeight < 0 {
		panelHeight = 0
	}

	// Render panels with focus indication
	treeView := m.renderPanel(
		m.tree.View(),
		treeWidth,
		panelHeight,
		"Folders",
		m.focusPanel == FocusPanelTree,
	)

	// Calculate content height (same as renderPanel does internally)
	contentHeight := panelHeight - 6 // borders, padding, title, separator
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Set height on email list so it knows how many to display
	m.emailList.SetHeight(contentHeight)
	listView := m.renderPanel(
		m.emailList.View(),
		listWidth,
		panelHeight,
		"Emails",
		m.focusPanel == FocusPanelList,
	)

	previewView := m.renderPanel(
		m.preview.View(),
		previewWidth,
		panelHeight,
		"Preview",
		m.focusPanel == FocusPanelPreview,
	)

	// Add top margin for cleaner rendering
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		treeView,
		listView,
		previewView,
	)

	// Add top spacing
	return panels
}

func (m *Model) renderCalendarView() string {
	return m.calendar.View()
}

func (m *Model) renderPanel(content string, width, height int, title string, focused bool) string {
	// Border and title colors based on focus
	borderColor := colorBorder
	titleColor := colorMuted
	if focused {
		borderColor = colorPrimary
		titleColor = colorPrimary
	}

	// Style the title
	styledTitle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true).
		Render(title)

	// Create a horizontal line separator
	sepWidth := width - 4
	if sepWidth < 0 {
		sepWidth = 0
	}
	separator := lipgloss.NewStyle().
		Foreground(borderColor).
		Render(strings.Repeat("─", sepWidth))

	// Calculate content area height
	// Total height - 2 (borders) - 2 (padding) - 1 (title) - 1 (separator)
	contentHeight := height - 6
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Apply height constraint to content
	styledContent := lipgloss.NewStyle().
		Height(contentHeight).
		Render(content)

	// Stack title, separator, and content
	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		styledTitle,
		separator,
		styledContent,
	)

	// Wrap in a complete border
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width - 4).   // Subtract 2 for border + 2 for padding
		Height(height - 2). // Subtract 2 for border
		Render(inner)
}

func (m *Model) renderWithOverlay(overlay string) string {
	base := m.renderEmailView()

	// Center the overlay
	overlayStyle := modalStyle.
		Width(m.width - 20).
		Height(m.height - 10)

	centeredOverlay := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		overlayStyle.Render(overlay),
	)

	// Dim the background
	dimmedBase := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(base)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
		lipgloss.Top,
		dimmedBase+centeredOverlay,
	)
}

func (m *Model) renderStatusBar() string {
	left := m.renderStatusLeft()
	right := m.renderStatusRight()

	// Calculate available space for centering
	// We want 'right' (shortcuts) to be centered if possible
	// But 'left' (status) needs to be on the left

	// Actually, user asked to center the bottom menu (shortcuts)
	// So we keep status on left, and center the shortcuts

	width := m.width
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	// Calculate padding to center 'right'
	// Total width = left + gap1 + right + gap2
	// We want gap1 + right + gap2 to be centered relative to the remaining space?
	// Or just center 'right' in the whole bar, but ensure it doesn't overlap 'left'

	// Simple approach: Left aligned status, Center aligned shortcuts

	centerPos := width / 2
	rightStart := centerPos - (rightWidth / 2)

	if rightStart < leftWidth+2 {
		rightStart = leftWidth + 2
	}

	gap := rightStart - leftWidth
	if gap < 0 {
		gap = 0
	}

	// Remaining space after right
	endGap := width - leftWidth - gap - rightWidth
	if endGap < 0 {
		endGap = 0
	}

	return statusBarStyle.
		Width(m.width).
		Render(left + lipgloss.NewStyle().Width(gap).Render("") + right + lipgloss.NewStyle().Width(endGap).Render(""))
}

func (m *Model) renderStatusLeft() string {
	var mode string
	switch m.viewMode {
	case ViewModeEmail:
		mode = "EMAIL"
	case ViewModeCalendar:
		mode = "CALENDAR"
	}

	parts := []string{
		mode,
	}

	if m.statusMessage != "" {
		parts = append(parts,
			statusBarSeparatorStyle.Render("·"),
			m.statusMessage,
		)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m *Model) renderStatusRight() string {
	shortcuts := []string{
		"Archive:a",
		"Reply:r",
		"Claude:c",
		"Schedule:s",
		"Triage:t",
		"Search:/",
		"Quit:q",
	}

	parts := make([]string, 0)
	for i, shortcut := range shortcuts {
		parts = append(parts, helpKeyStyle.Render(shortcut))
		if i < len(shortcuts)-1 {
			parts = append(parts, statusBarSeparatorStyle.Render(" │ "))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m *Model) cycleFocus() {
	switch m.focusPanel {
	case FocusPanelTree:
		m.focusPanel = FocusPanelList
	case FocusPanelList:
		m.focusPanel = FocusPanelPreview
	case FocusPanelPreview:
		m.focusPanel = FocusPanelTree
	}
}

// Commands

func (m *Model) connectAndLoad() tea.Msg {
	// Connect to email accounts
	ctx := context.Background()
	if err := m.emailManager.Connect(ctx); err != nil {
		return ErrorMsg{Error: fmt.Sprintf("Failed to connect: %v", err)}
	}

	// Get accounts with real folders
	accounts := m.emailManager.GetAccounts()

	// Auto-expand accounts so folders are visible on startup
	for _, account := range accounts {
		account.Expanded = true
	}

	// Set accounts on tree FIRST so they exist when we add folders
	m.tree.SetAccounts(accounts)

	// Now fetch and set folders for each account
	for _, account := range accounts {
		// Use timeout for folder listing
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		folders, err := m.emailManager.GetFolders(ctx, account.ID)
		cancel()

		if err != nil {
			continue
		}

		// Set folders on tree view for this account
		m.tree.SetFolders(account.ID, folders)
	}

	return AccountsLoadedMsg{Accounts: accounts}
}

func (m *Model) loadEmailsForFolder(accountID, folderName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		emails, err := m.emailManager.GetEmails(ctx, accountID, folderName, 50)
		if err != nil {
			return ErrorMsg{Error: fmt.Sprintf("Failed to load emails: %v", err)}
		}
		return EmailsLoadedMsg{Emails: emails}
	}
}

func (m *Model) loadFoldersForAccount(accountID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		folders, err := m.emailManager.GetFolders(ctx, accountID)
		if err != nil {
			return ErrorMsg{Error: fmt.Sprintf("Failed to load folders: %v", err)}
		}
		return FoldersLoadedMsg{AccountID: accountID, Folders: folders}
	}
}

func (m *Model) loadEmailBody(email *models.Email) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse sequence number from email ID
		var seqNum uint32
		fmt.Sscanf(email.ID, "%d", &seqNum)

		// Extract folder name from FolderID (format: "accountID-folderName")
		parts := strings.SplitN(email.FolderID, "-", 2)
		if len(parts) != 2 {
			return ErrorMsg{Error: "Invalid folder ID"}
		}
		folderName := parts[1]

		// Fetch body using sequence number
		emailWithBody, err := m.emailManager.GetEmailBody(ctx, email.AccountID, folderName, seqNum)
		if err != nil {
			return ErrorMsg{Error: fmt.Sprintf("Failed to load email body: %v", err)}
		}

		return EmailBodyLoadedMsg{
			EmailID:  email.ID,
			TextBody: emailWithBody.Body,
			HTMLBody: emailWithBody.BodyHTML,
		}
	}
}

func (m *Model) triageInbox() tea.Msg {
	// TODO: Call Claude service to triage emails
	return nil
}

// Messages

type AccountsLoadedMsg struct {
	Accounts []*models.Account
}

type FoldersLoadedMsg struct {
	AccountID string
	Folders   []*models.Folder
}

type EmailsLoadedMsg struct {
	Emails []*models.Email
}

type EmailSelectedMsg struct {
	Email *models.Email
}

type EmailBodyLoadedMsg struct {
	EmailID  string
	TextBody string
	HTMLBody string
}

type ErrorMsg struct {
	Error string
}

// Messages for email operations
type EmailArchivedMsg struct {
	ID string
}

type EmailDeletedMsg struct {
	ID string
}

type EmailMarkedReadMsg struct {
	ID   string
	Read bool
}

func (m *Model) archiveEmail(id string) tea.Cmd {
	return func() tea.Msg {
		var email *models.Email
		for _, e := range m.emails {
			if e.ID == id {
				email = e
				break
			}
		}

		if email == nil {
			return ErrorMsg{Error: "Email not found"}
		}

		err := m.emailManager.Archive(context.Background(), email.AccountID, []string{email.ID})
		if err != nil {
			return ErrorMsg{Error: fmt.Sprintf("Failed to archive: %v", err)}
		}
		return EmailArchivedMsg{ID: id}
	}
}

func (m *Model) deleteEmail(id string) tea.Cmd {
	return func() tea.Msg {
		var email *models.Email
		for _, e := range m.emails {
			if e.ID == id {
				email = e
				break
			}
		}

		if email == nil {
			return ErrorMsg{Error: "Email not found"}
		}

		err := m.emailManager.Delete(context.Background(), email.AccountID, []string{email.ID})
		if err != nil {
			return ErrorMsg{Error: fmt.Sprintf("Failed to delete: %v", err)}
		}
		return EmailDeletedMsg{ID: id}
	}
}

func (m *Model) markRead(id string, read bool) tea.Cmd {
	return func() tea.Msg {
		var email *models.Email
		for _, e := range m.emails {
			if e.ID == id {
				email = e
				break
			}
		}

		if email == nil {
			return ErrorMsg{Error: "Email not found"}
		}

		err := m.emailManager.MarkRead(context.Background(), email.AccountID, []string{email.ID}, read)
		if err != nil {
			return ErrorMsg{Error: fmt.Sprintf("Failed to mark read: %v", err)}
		}
		return EmailMarkedReadMsg{ID: id, Read: read}
	}
}
