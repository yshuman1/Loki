package email

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/yshuman1/loki/internal/models"
)

// Manager manages multiple email accounts
type Manager struct {
	accounts   []*models.Account
	clients    map[string]*Client
	emailCache map[string][]*models.Email // key: "accountID-folderName"
	mu         sync.RWMutex
}

// NewManager creates a new email manager
func NewManager(accounts []*models.Account) *Manager {
	return &Manager{
		accounts:   accounts,
		clients:    make(map[string]*Client),
		emailCache: make(map[string][]*models.Email),
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
	// Check cache first
	cacheKey := accountID + "-" + folderName
	m.mu.RLock()
	if cached, ok := m.emailCache[cacheKey]; ok {
		m.mu.RUnlock()
		return cached, nil
	}
	m.mu.RUnlock()

	// Not in cache, fetch from IMAP
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

	// Store in cache
	m.mu.Lock()
	m.emailCache[cacheKey] = emails
	m.mu.Unlock()

	return emails, nil
}

// GetEmailBody fetches the full body of an email
func (m *Manager) GetEmailBody(ctx context.Context, accountID, folderName string, seqNum uint32) (*models.Email, error) {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("account not connected: %s", accountID)
	}

	textBody, htmlBody, err := client.FetchEmailBody(ctx, folderName, seqNum)
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

// Archive archives emails
func (m *Manager) Archive(ctx context.Context, accountID string, ids []string) error {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("account not connected: %s", accountID)
	}

	uids, err := parseUIDs(ids)
	if err != nil {
		return err
	}

	return client.Archive(ctx, uids)
}

// Delete deletes emails
func (m *Manager) Delete(ctx context.Context, accountID string, ids []string) error {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("account not connected: %s", accountID)
	}

	uids, err := parseUIDs(ids)
	if err != nil {
		return err
	}

	return client.Delete(ctx, uids)
}

// MarkRead marks emails as read/unread
func (m *Manager) MarkRead(ctx context.Context, accountID string, ids []string, read bool) error {
	m.mu.RLock()
	client, ok := m.clients[accountID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("account not connected: %s", accountID)
	}

	uids, err := parseUIDs(ids)
	if err != nil {
		return err
	}

	return client.MarkRead(ctx, uids, read)
}

// GetAccounts returns all configured accounts
func (m *Manager) GetAccounts() []*models.Account {
	return m.accounts
}

func parseUIDs(ids []string) ([]uint32, error) {
	var uids []uint32
	for _, id := range ids {
		uid, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid UID %s: %w", id, err)
		}
		uids = append(uids, uint32(uid))
	}
	return uids, nil
}

// ClearCache clears the email cache (for refresh)
func (m *Manager) ClearCache(accountID, folderName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if accountID == "" {
		// Clear all cache
		m.emailCache = make(map[string][]*models.Email)
	} else if folderName == "" {
		// Clear all folders for account
		for key := range m.emailCache {
			if strings.HasPrefix(key, accountID+"-") {
				delete(m.emailCache, key)
			}
		}
	} else {
		// Clear specific folder
		cacheKey := accountID + "-" + folderName
		delete(m.emailCache, cacheKey)
	}
}
