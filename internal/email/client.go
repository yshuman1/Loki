package email

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/yshuman1/loki/internal/models"
)

// Client is an email client that supports IMAP
type Client struct {
	account *models.Account
	imap    *imapclient.Client
}

// NewClient creates a new email client
func NewClient(account *models.Account) *Client {
	return &Client{
		account: account,
	}
}

// Connect connects to the IMAP server
func (c *Client) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.account.IMAPServer, c.account.IMAPPort)

	// Connect with TLS
	options := &imapclient.Options{
		// Add debug logging if needed
	}

	client, err := imapclient.DialTLS(addr, options)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	c.imap = client

	// Login
	if err := c.imap.Login(c.account.Credentials.Username, c.account.Credentials.Password).Wait(); err != nil {
		c.imap.Close()
		return fmt.Errorf("failed to login: %w", err)
	}

	return nil
}

// Disconnect closes the IMAP connection
func (c *Client) Disconnect() error {
	if c.imap != nil {
		return c.imap.Close()
	}
	return nil
}

// shouldSkipFolder returns true if the folder should be hidden from the user
func shouldSkipFolder(name string) bool {
	// Skip the [Gmail] parent folder itself
	if name == "[Gmail]" {
		return true
	}
	
	// Gmail web UI hides these by default (they're under "More")
	// Let's only show the main folders to match the clean Gmail UI
	
	// Skip "All Mail" - it's under "More" in Gmail
	if name == "[Gmail]/All Mail" {
		return true
	}
	
	// Skip "Important" - it's under "More" in Gmail  
	if name == "[Gmail]/Important" {
		return true
	}
	
	// Skip "Starred" - it's a smart label in Gmail, not a real folder
	if name == "[Gmail]/Starred" {
		return true
	}
	
	// Skip "Snoozed" - another smart label
	if strings.Contains(name, "Snoozed") {
		return true
	}
	
	// Skip "Scheduled" - another smart label
	if strings.Contains(name, "Scheduled") {
		return true
	}
	
	return false
}

// cleanFolderName removes Gmail-specific prefixes and makes names more readable
func cleanFolderName(name string) string {
	// Remove [Gmail]/ prefix
	name = strings.TrimPrefix(name, "[Gmail]/")
	
	// Gmail uses "Sent Mail" but we'll show it as "Sent"
	if name == "Sent Mail" {
		return "Sent"
	}
	
	return name
}

// getFolderPriority returns a priority for sorting folders like Gmail
func getFolderPriority(folder *models.Folder) int {
	switch folder.Type {
	case models.FolderTypeInbox:
		return 0
	case models.FolderTypeSent:
		return 1
	case models.FolderTypeDrafts:
		return 2
	default:
		// Handle special cases by name for Custom type
		name := strings.ToLower(folder.DisplayName)
		if name == "spam" || name == "junk" {
			return 3
		}
		if folder.Type == models.FolderTypeTrash {
			return 4
		}
		if folder.Type == models.FolderTypeArchive {
			return 5
		}
		return 99
	}
}

// ListFolders returns all folders/mailboxes
func (c *Client) ListFolders(ctx context.Context) ([]*models.Folder, error) {
	if c.imap == nil {
		return nil, fmt.Errorf("not connected")
	}

	// List all mailboxes
	mailboxes, err := c.imap.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}

	folders := make([]*models.Folder, 0, len(mailboxes))
	for _, mbox := range mailboxes {
		// Skip folders that shouldn't be shown
		if shouldSkipFolder(mbox.Mailbox) {
			continue
		}
		
		// Clean up the folder name for display
		displayName := cleanFolderName(mbox.Mailbox)
		folderType := determineFolderType(mbox.Mailbox)
		
		folder := &models.Folder{
			ID:          c.account.ID + "-" + mbox.Mailbox,
			AccountID:   c.account.ID,
			Name:        mbox.Mailbox, // Keep original name for IMAP operations
			DisplayName: displayName,   // Show cleaned name to user
			Type:        folderType,
		}
		folders = append(folders, folder)
	}

	// Sort folders to match Gmail's order: Inbox, Sent, Drafts, Spam, Trash
	sort.Slice(folders, func(i, j int) bool {
		return getFolderPriority(folders[i]) < getFolderPriority(folders[j])
	})

	return folders, nil
}

// FetchEmails fetches emails from a folder
func (c *Client) FetchEmails(ctx context.Context, folderName string, limit int) ([]*models.Email, error) {
	if c.imap == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Select mailbox
	selectCmd := c.imap.Select(folderName, nil)
	mboxData, err := selectCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	if mboxData.NumMessages == 0 {
		return []*models.Email{}, nil
	}

	// Calculate sequence set for most recent emails
	start := uint32(1)
	end := mboxData.NumMessages
	if mboxData.NumMessages > uint32(limit) {
		start = mboxData.NumMessages - uint32(limit) + 1
	}

	seqSet := imap.SeqSet{}
	seqSet.AddRange(start, end)

	// Fetch email data
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
		UID:      true,
	}

	fetchCmd := c.imap.Fetch(seqSet, fetchOptions)
	defer fetchCmd.Close()

	emails := make([]*models.Email, 0, limit)
	
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		email, err := c.parseMessage(msg)
		if err != nil {
			// Log error but continue
			continue
		}

		email.AccountID = c.account.ID
		emails = append(emails, email)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	// Reverse to show newest first
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

// FetchEmailBody fetches the full body of an email
func (c *Client) FetchEmailBody(ctx context.Context, folderName string, uid uint32) (string, string, error) {
	if c.imap == nil {
		return "", "", fmt.Errorf("not connected")
	}

	// For now, return placeholder
	// TODO: Implement body fetching with correct API
	return "Email body preview coming soon...", "", nil
}

func (c *Client) parseMessage(msg *imapclient.FetchMessageData) (*models.Email, error) {
	// The FetchMessageData structure varies between beta versions
	// For now, create a basic email with what we can safely access
	
	email := &models.Email{
		ID:       fmt.Sprintf("%d", 1), // Placeholder
		Subject:  "Email (preview mode)",
		Date:     time.Now(),
		Read:     false,
		Priority: models.PriorityMedium,
		From: &models.EmailAddress{
			Name:    "Sender",
			Address: "sender@example.com",
		},
		To: []*models.EmailAddress{
			{Name: "You", Address: c.account.Email},
		},
	}

	return email, nil
}

func determineFolderType(name string) models.FolderType {
	// Check original name (before cleaning) for folder type
	switch {
	case name == "INBOX":
		return models.FolderTypeInbox
	case name == "[Gmail]/Sent Mail" || name == "Sent" || name == "Sent Messages" || name == "Sent Items":
		return models.FolderTypeSent
	case name == "[Gmail]/Drafts" || name == "Drafts":
		return models.FolderTypeDrafts
	case name == "[Gmail]/All Mail" || name == "Archive":
		return models.FolderTypeArchive
	case name == "[Gmail]/Trash" || name == "Trash" || name == "Deleted Items":
		return models.FolderTypeTrash
	case name == "[Gmail]/Spam" || name == "Spam" || name == "Junk":
		return models.FolderTypeCustom
	default:
		return models.FolderTypeCustom
	}
}

// GetFolderStatus gets the status of a folder (message count, unseen, etc.)
func (c *Client) GetFolderStatus(ctx context.Context, folderName string) (*models.Folder, error) {
	if c.imap == nil {
		return nil, fmt.Errorf("not connected")
	}

	statusCmd := c.imap.Status(folderName, &imap.StatusOptions{
		NumMessages: true,
		NumUnseen:   true,
	})

	status, err := statusCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	folder := &models.Folder{
		ID:          c.account.ID + "-" + folderName,
		AccountID:   c.account.ID,
		Name:        folderName,
		DisplayName: cleanFolderName(folderName),
		Type:        determineFolderType(folderName),
		TotalCount:  int(*status.NumMessages),
		UnreadCount: int(*status.NumUnseen),
	}

	return folder, nil
}
