package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yshuman1/loki/internal/models"
)

type TreeModel struct {
	nodes    []*TreeNode
	cursor   int
	selected int
	height   int
}

type TreeNode struct {
	nodeType    models.TreeNodeType
	account     *models.Account
	folder      *models.Folder
	level       int
	expanded    bool
	hasChildren bool
}

func NewTreeModel() *TreeModel {
	return &TreeModel{
		nodes:    make([]*TreeNode, 0),
		cursor:   0,
		selected: 0,
	}
}

func (m *TreeModel) SetAccounts(accounts []*models.Account) {
	m.nodes = make([]*TreeNode, 0)

	for _, account := range accounts {
		// Add account node
		accountNode := &TreeNode{
			nodeType:    models.TreeNodeTypeAccount,
			account:     account,
			level:       0,
			expanded:    account.Expanded,
			hasChildren: true,
		}
		m.nodes = append(m.nodes, accountNode)

		// Note: Folders will be loaded from IMAP and set separately
		// For now, just show the account
	}
}

// SetFolders adds folders for an account
func (m *TreeModel) SetFolders(accountID string, folders []*models.Folder) {
	// Find the account node
	accountIndex := -1
	for i, node := range m.nodes {
		if node.nodeType == models.TreeNodeTypeAccount && node.account.ID == accountID {
			accountIndex = i
			break
		}
	}

	if accountIndex == -1 {
		return
	}

	// Remove old folder nodes for this account
	newNodes := make([]*TreeNode, 0)
	for i, node := range m.nodes {
		if i <= accountIndex {
			newNodes = append(newNodes, node)
		} else if node.nodeType == models.TreeNodeTypeAccount {
			// Hit next account, add it and everything after
			newNodes = append(newNodes, m.nodes[i:]...)
			break
		}
	}

	// Add new folder nodes if account is expanded
	if m.nodes[accountIndex].expanded {
		for _, folder := range folders {
			folderNode := &TreeNode{
				nodeType:    models.TreeNodeTypeFolder,
				folder:      folder,
				level:       1,
				expanded:    false,
				hasChildren: false,
			}
			newNodes = append(newNodes, folderNode)
		}
	}

	m.nodes = newNodes
}

func (m *TreeModel) Update(msg tea.Msg) (*TreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "g":
			// Go to top
			m.cursor = 0

		case "G":
			// Go to bottom
			m.cursor = len(m.nodes) - 1

		case "o", "enter", "l", "right":
			// Expand node
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.nodeType == models.TreeNodeTypeAccount && !node.expanded {
					node.expanded = true
					node.account.Expanded = true
					// Emit message to load folders
					return m, func() tea.Msg {
						return AccountExpandedMsg{AccountID: node.account.ID}
					}
				}
			}

		case "O", "h", "left":
			// Collapse node
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.nodeType == models.TreeNodeTypeAccount {
					node.expanded = false
					node.account.Expanded = false
					m.rebuildNodes()
				}
			}

		case " ":
			// Select folder
			m.selected = m.cursor
			// Emit folder selected message
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.nodeType == models.TreeNodeTypeFolder {
					return m, func() tea.Msg {
						return FolderSelectedMsg{
							AccountID:  node.folder.AccountID,
							FolderName: node.folder.Name,
						}
					}
				}
			}
		}
	}

	return m, nil
}

func (m *TreeModel) rebuildNodes() {
	// Store all accounts
	accounts := make([]*models.Account, 0)
	for _, node := range m.nodes {
		if node.nodeType == models.TreeNodeTypeAccount {
			accounts = append(accounts, node.account)
		}
	}
	m.SetAccounts(accounts)
}

func (m *TreeModel) View() string {
	var b strings.Builder

	for i, node := range m.nodes {
		// Indent based on level
		indent := strings.Repeat("  ", node.level)

		// Expand/collapse icon
		icon := ""
		if node.hasChildren {
			if node.expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
		} else {
			icon = "  "
		}

		// Selected indicator
		selectedIcon := " "
		if i == m.selected {
			selectedIcon = "●"
		}

		// Render node
		var line string
		if node.nodeType == models.TreeNodeTypeAccount {
			name := node.account.Name
			if i == m.cursor {
				line = treeNodeSelectedStyle.Render(fmt.Sprintf("%s%s%s %s", indent, selectedIcon, name, icon))
			} else {
				line = treeNodeStyle.Render(fmt.Sprintf("%s%s%s %s", indent, selectedIcon, name, icon))
			}
		} else if node.nodeType == models.TreeNodeTypeFolder {
			// Use the cleaned display name
			name := node.folder.DisplayName

			// Truncate if too long
			if len(name) > 15 {
				name = name[:12] + "..."
			}

			unread := ""
			if node.folder.UnreadCount > 0 {
				unread = fmt.Sprintf(" (%d)", node.folder.UnreadCount)
			}
			if i == m.cursor {
				line = treeNodeSelectedStyle.Render(fmt.Sprintf("%s%s%s %s%s", indent, selectedIcon, name, unread, icon))
			} else {
				line = treeNodeStyle.Render(fmt.Sprintf("%s%s%s %s%s", indent, selectedIcon, name, unread, icon))
			}
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m *TreeModel) GetSelectedFolder() *models.Folder {
	if m.selected >= 0 && m.selected < len(m.nodes) {
		node := m.nodes[m.selected]
		if node.nodeType == models.TreeNodeTypeFolder {
			return node.folder
		}
	}
	return nil
}

// FolderSelectedMsg is emitted when a folder is selected
type FolderSelectedMsg struct {
	AccountID  string
	FolderName string
}

// AccountExpandedMsg is emitted when an account is expanded
type AccountExpandedMsg struct {
	AccountID string
}
