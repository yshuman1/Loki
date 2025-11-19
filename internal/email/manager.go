package email

import (
	"context"
	"fmt"
	"sync"

	"github.com/yshuman1/loki/internal/models"
)

// Manager manages multiple email accounts
type Manager struct {
	accounts []*models.Account
	clients  map[string]*Client
	mu       sync.RWMutex
}

// NewManager creates a new email manager
func NewManager(accounts []*models.Account) *Manager {
	return &Manager{
		accounts: accounts,
		clients:  make(map[string]*Client),
	}
}

// Connect connects to all accounts
func (m *Manager) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, account := range m.accounts {
		if account.Type != models.AccountTypeIMAP {
			// Skip non-IMAP accounts for now
			continue
		}

		client := NewClient(account)
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to %s: %w", account.Email, err)
		}

		m.clients[account.ID] = client
	}

	return nil
}

// Disconnect disconnects from all accounts
func (m *Manager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		if err := client.Disconnect(); err != nil {
			// Log error but continue
			continue
		}
	}

	m.clients = make(map[string]*Client)
	return nil
}

// GetFolders returns folders for a specific account
func (m *Manager) GetFolders(ctx context.Context, accountID string) ([]*models.Folder, error) {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("account not connected: %s", accountID)
	}

	return client.ListFolders(ctx)
}

// GetEmails fetches emails from a specific folder
func (m *Manager) GetEmails(ctx context.Context, accountID, folderName string, limit int) ([]*models.Email, error) {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("account not connected: %s", accountID)
	}

	emails, err := client.FetchEmails(ctx, folderName, limit)
	if err != nil {
		return nil, err
	}

	// Set folder ID for each email
	for _, email := range emails {
		email.FolderID = accountID + "-" + folderName
	}

	return emails, nil
}

// GetEmailBody fetches the full body of an email
func (m *Manager) GetEmailBody(ctx context.Context, accountID, folderName string, uid uint32) (*models.Email, error) {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("account not connected: %s", accountID)
	}

	textBody, htmlBody, err := client.FetchEmailBody(ctx, folderName, uid)
	if err != nil {
		return nil, err
	}

	// Create a simple email object with body
	email := &models.Email{
		Body:     textBody,
		BodyHTML: htmlBody,
	}

	return email, nil
}

// GetFolderStatus gets the status of a folder
func (m *Manager) GetFolderStatus(ctx context.Context, accountID, folderName string) (*models.Folder, error) {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("account not connected: %s", accountID)
	}

	return client.GetFolderStatus(ctx, folderName)
}

// GetAccounts returns all configured accounts
func (m *Manager) GetAccounts() []*models.Account {
	return m.accounts
}
