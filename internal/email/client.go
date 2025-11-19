package email

import (
	"context"
	"fmt"
	"sort"
	"strings"

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

	// List all folders using wildcard
	// For Gmail, this includes INBOX, [Gmail]/..., and custom labels
	mailboxes, err := c.imap.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	folders := make([]*models.Folder, 0, len(mailboxes))
	seen := make(map[string]bool)

	for _, mbox := range mailboxes {
		// Deduplicate
		if seen[mbox.Mailbox] {
			continue
		}
		seen[mbox.Mailbox] = true

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
			DisplayName: displayName,  // Show cleaned name to user
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

	// Calculate sequence set for most recent emails
	start := uint32(1)
	end := mboxData.NumMessages
	if mboxData.NumMessages > uint32(limit) {
		start = mboxData.NumMessages - uint32(limit) + 1
	}

	seqSet := imap.SeqSet{}
	seqSet.AddRange(start, end)

	// Fetch by sequence number - UID is automatically included in response
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
	}

	messages, err := c.imap.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	var emails []*models.Email
	for _, msg := range messages {
		email, err := c.parseMessage(msg)
		if err != nil {
			// Log error but continue
			continue
		}
		email.AccountID = c.account.ID
		emails = append(emails, email)
	}

	// Reverse to show newest first
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

// FetchEmailBody fetches the full body of an email by sequence number
func (c *Client) FetchEmailBody(ctx context.Context, folderName string, seqNum uint32) (string, string, error) {
	if c.imap == nil {
		return "", "", fmt.Errorf("not connected")
	}

	// Select mailbox if not selected
	_, err := c.imap.Select(folderName, nil).Wait()
	if err != nil {
		return "", "", fmt.Errorf("failed to select folder %s: %w", folderName, err)
	}

	// Fetch body by sequence number
	seqSet := imap.SeqSet{}
	seqSet.AddNum(seqNum)

	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	messages, err := c.imap.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		return "", "", fmt.Errorf("fetch failed for seq %d: %w", seqNum, err)
	}

	if len(messages) == 0 {
		return "", "", fmt.Errorf("message not found: seq %d in folder %s", seqNum, folderName)
	}

	msg := messages[0]

	// Find the body section in the buffer
	var bodyBytes []byte
	for _, buf := range msg.BodySection {
		if buf.Section == bodySection {
			bodyBytes = buf.Bytes
			break
		}
	}

	if bodyBytes == nil && len(msg.BodySection) > 0 {
		bodyBytes = msg.BodySection[0].Bytes
	}

	if bodyBytes == nil {
		return "", "", nil
	}

	// Parse MIME message to extract text and HTML parts
	textBody, htmlBody, err := parseMIMEBody(bodyBytes)
	if err != nil {
		// If MIME parsing fails, return raw body
		return string(bodyBytes), "", nil
	}

	return textBody, htmlBody, nil
}

// parseMIMEBody parses a MIME message and extracts text and HTML parts
func parseMIMEBody(bodyBytes []byte) (string, string, error) {
	// For now, just return the raw body as text
	// TODO: Implement proper MIME parsing with go-message library
	// This requires handling multipart messages, base64 decoding, etc.

	// Simple approach: try to detect if it's just plain text
	body := string(bodyBytes)

	// If it looks like HTML, put it in HTML
	if strings.Contains(body, "<html") || strings.Contains(body, "<HTML") {
		return "", body, nil
	}

	// Otherwise treat as plain text
	return body, "", nil
}

func (c *Client) parseMessage(msg *imapclient.FetchMessageBuffer) (*models.Email, error) {
	envelope := msg.Envelope
	if envelope == nil {
		return &models.Email{
			ID:      fmt.Sprintf("%d", msg.SeqNum),
			Subject: "(No Envelope)",
		}, nil
	}

	// Parse date
	date := envelope.Date

	// Parse sender
	var from *models.EmailAddress
	if len(envelope.From) > 0 {
		from = &models.EmailAddress{
			Name:    envelope.From[0].Name,
			Address: fmt.Sprintf("%s@%s", envelope.From[0].Mailbox, envelope.From[0].Host),
		}
	}

	// Parse recipient
	var to []*models.EmailAddress
	for _, addr := range envelope.To {
		to = append(to, &models.EmailAddress{
			Name:    addr.Name,
			Address: fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host),
		})
	}

	// Parse flags
	read := false
	starred := false
	for _, flag := range msg.Flags {
		if flag == imap.FlagSeen {
			read = true
		}
		if flag == imap.FlagFlagged {
			starred = true
		}
	}

	// Use sequence number as ID (stable within a folder session)
	email := &models.Email{
		ID:        fmt.Sprintf("%d", msg.SeqNum),
		MessageID: envelope.MessageID,
		Subject:   envelope.Subject,
		Date:      date,
		From:      from,
		To:        to,
		Read:      read,
		Starred:   starred,
		Priority:  models.PriorityMedium, // Default
	}

	return email, nil
}

// Archive moves emails to the Archive folder (or All Mail)
func (c *Client) Archive(ctx context.Context, uids []uint32) error {
	if c.imap == nil {
		return fmt.Errorf("not connected")
	}
	return c.Delete(ctx, uids)
}

// Delete marks emails as deleted
func (c *Client) Delete(ctx context.Context, uids []uint32) error {
	if c.imap == nil {
		return fmt.Errorf("not connected")
	}

	seqSet := imap.SeqSet{}
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	// Add \Deleted flag
	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{imap.FlagDeleted},
		Silent: true,
	}

	// Use Close() instead of Wait() for Store command
	return c.imap.Store(seqSet, &storeFlags, nil).Close()
}

// MarkRead marks emails as read or unread
func (c *Client) MarkRead(ctx context.Context, uids []uint32, read bool) error {
	if c.imap == nil {
		return fmt.Errorf("not connected")
	}

	seqSet := imap.SeqSet{}
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	op := imap.StoreFlagsAdd
	if !read {
		op = imap.StoreFlagsDel
	}

	storeFlags := imap.StoreFlags{
		Op:     op,
		Flags:  []imap.Flag{imap.FlagSeen},
		Silent: true,
	}

	return c.imap.Store(seqSet, &storeFlags, nil).Close()
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
