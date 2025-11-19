package tui

import (
	"context"
	"fmt"

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
	viewMode    ViewMode
	focusPanel  FocusPanel
	width       int
	height      int

	// Data
	accounts      []*models.Account
	currentEmail  *models.Email
	currentMeeting *models.Meeting
	emails        []*models.Email
	meetings      []*models.Meeting

	// UI components
	tree         *TreeModel
	emailList    *EmailListModel
	preview      *PreviewModel
	calendar     *CalendarModel
	claudeChat   *ClaudeChatModel
	scheduler    *SchedulerModel

	// State
	loading       bool
	error         string
	statusMessage string
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case AccountsLoadedMsg:
		m.accounts = msg.Accounts
		m.tree.SetAccounts(msg.Accounts)
		return m, nil
		
	case FoldersLoadedMsg:
		m.tree.SetFolders(msg.AccountID, msg.Folders)
		m.statusMessage = fmt.Sprintf("Loaded %d folders", len(msg.Folders))
		return m, nil

	case EmailsLoadedMsg:
		m.emails = msg.Emails
		m.emailList.SetEmails(msg.Emails)
		m.statusMessage = fmt.Sprintf("Loaded %d emails", len(msg.Emails))
		return m, nil

	case EmailSelectedMsg:
		m.currentEmail = msg.Email
		m.preview.SetEmail(msg.Email)
		return m, nil

	case ErrorMsg:
		m.error = msg.Error
		m.statusMessage = "Error: " + msg.Error
		return m, nil
		
	case FolderSelectedMsg:
		m.statusMessage = fmt.Sprintf("Loading emails from %s...", msg.FolderName)
		return m, m.loadEmailsForFolder(msg.AccountID, msg.FolderName)
		
	case AccountExpandedMsg:
		m.statusMessage = fmt.Sprintf("Loading folders...")
		return m, m.loadFoldersForAccount(msg.AccountID)
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

	// Add status bar
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainView,
		statusBar,
	)
}

func (m *Model) renderEmailView() string {
	// Calculate panel widths
	treeWidth := 20
	previewWidth := m.width - treeWidth - 35 // Remaining space
	listWidth := 35

	// Adjust heights
	panelHeight := m.height - 2 // Reserve space for status bar

	// Render panels with focus indication
	treeView := m.renderPanel(
		m.tree.View(),
		treeWidth,
		panelHeight,
		"Accounts/Folders",
		m.focusPanel == FocusPanelTree,
	)

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

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		treeView,
		listView,
		previewView,
	)
}

func (m *Model) renderCalendarView() string {
	return m.calendar.View()
}

func (m *Model) renderPanel(content string, width, height int, title string, focused bool) string {
	style := borderStyle
	if focused {
		style = activeBorderStyle
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	if !focused {
		titleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
	}

	header := titleStyle.Render(title)

	return style.
		Width(width).
		Height(height).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			RenderDivider(width-4),
			content,
		))
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

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return statusBarStyle.
		Width(m.width).
		Render(left + lipgloss.NewStyle().Width(gap).Render("") + right)
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
		statusBarKeyStyle.Render("LOKI"),
		statusBarSeparatorStyle.Render("·"),
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
	for _, account := range accounts {
		folders, err := m.emailManager.GetFolders(ctx, account.ID)
		if err != nil {
			continue
		}
		
		// Update folder counts
		for _, folder := range folders {
			status, err := m.emailManager.GetFolderStatus(ctx, account.ID, folder.Name)
			if err == nil {
				folder.UnreadCount = status.UnreadCount
				folder.TotalCount = status.TotalCount
			}
		}
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

type ErrorMsg struct {
	Error string
}
