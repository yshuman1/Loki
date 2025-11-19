package email

import (
	"context"
	"fmt"
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
		folderType := determineFolderType(mbox.Mailbox)
		
		folder := &models.Folder{
			ID:          c.account.ID + "-" + mbox.Mailbox,
			AccountID:   c.account.ID,
			Name:        mbox.Mailbox,
			DisplayName: mbox.Mailbox,
			Type:        folderType,
		}
		folders = append(folders, folder)
	}

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
	switch name {
	case "INBOX":
		return models.FolderTypeInbox
	case "Sent", "Sent Messages", "Sent Items":
		return models.FolderTypeSent
	case "Drafts":
		return models.FolderTypeDrafts
	case "Archive", "All Mail":
		return models.FolderTypeArchive
	case "Trash", "Deleted Items":
		return models.FolderTypeTrash
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
		DisplayName: folderName,
		Type:        determineFolderType(folderName),
		TotalCount:  int(*status.NumMessages),
		UnreadCount: int(*status.NumUnseen),
	}

	return folder, nil
}
