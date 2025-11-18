package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/loki/internal/models"
)

type TreeModel struct {
	nodes    []*TreeNode
	cursor   int
	selected int
	height   int
}

type TreeNode struct {
	nodeType models.TreeNodeType
	account  *models.Account
	folder   *models.Folder
	level    int
	expanded bool
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
			nodeType: models.TreeNodeTypeAccount,
			account:  account,
			level:    0,
			expanded: account.Expanded,
			hasChildren: true,
		}
		m.nodes = append(m.nodes, accountNode)
		
		// Add folders if expanded
		if account.Expanded {
			folders := m.getDefaultFolders(account.ID)
			for _, folder := range folders {
				folderNode := &TreeNode{
					nodeType: models.TreeNodeTypeFolder,
					folder:  folder,
					level:   1,
					expanded: false,
					hasChildren: false,
				}
				m.nodes = append(m.nodes, folderNode)
			}
		}
	}
}

func (m *TreeModel) getDefaultFolders(accountID string) []*models.Folder {
	return []*models.Folder{
		{
			ID:          accountID + "-inbox",
			AccountID:   accountID,
			Name:        "INBOX",
			DisplayName: "Inbox",
			Type:        models.FolderTypeInbox,
			UnreadCount: 23,
		},
		{
			ID:          accountID + "-sent",
			AccountID:   accountID,
			Name:        "Sent",
			DisplayName: "Sent",
			Type:        models.FolderTypeSent,
		},
		{
			ID:          accountID + "-drafts",
			AccountID:   accountID,
			Name:        "Drafts",
			DisplayName: "Drafts",
			Type:        models.FolderTypeDrafts,
			UnreadCount: 2,
		},
		{
			ID:          accountID + "-archive",
			AccountID:   accountID,
			Name:        "Archive",
			DisplayName: "Archive",
			Type:        models.FolderTypeArchive,
		},
		{
			ID:          accountID + "-trash",
			AccountID:   accountID,
			Name:        "Trash",
			DisplayName: "Trash",
			Type:        models.FolderTypeTrash,
		},
	}
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
				if node.nodeType == models.TreeNodeTypeAccount {
					node.expanded = true
					node.account.Expanded = true
					// Rebuild nodes
					m.rebuildNodes()
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
			// TODO: Load emails for selected folder
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
				line = treeNodeSelectedStyle.Render(fmt.Sprintf("%s%s%s %s", indent, icon, selectedIcon, name))
			} else {
				line = treeNodeStyle.Render(fmt.Sprintf("%s%s%s %s", indent, icon, selectedIcon, name))
			}
		} else if node.nodeType == models.TreeNodeTypeFolder {
			name := node.folder.DisplayName
			unread := ""
			if node.folder.UnreadCount > 0 {
				unread = fmt.Sprintf(" (%d)", node.folder.UnreadCount)
			}
			if i == m.cursor {
				line = treeNodeSelectedStyle.Render(fmt.Sprintf("%s%s%s %s%s", indent, icon, selectedIcon, name, unread))
			} else {
				line = treeNodeStyle.Render(fmt.Sprintf("%s%s%s %s%s", indent, icon, selectedIcon, name, unread))
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
