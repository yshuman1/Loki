package models

import (
	"time"
)

// Account represents an email account
type Account struct {
	ID          string
	Name        string
	Email       string
	Type        AccountType
	IMAPServer  string
	IMAPPort    int
	SMTPServer  string
	SMTPPort    int
	Credentials *Credentials
	Expanded    bool // For tree view
}

type AccountType string

const (
	AccountTypeIMAP  AccountType = "imap"
	AccountTypeGmail AccountType = "gmail"
)

// Credentials stores authentication info
type Credentials struct {
	Username     string
	Password     string // Stored in keychain
	AccessToken  string // For OAuth
	RefreshToken string
	ExpiresAt    time.Time
}

// Folder represents an email folder/mailbox
type Folder struct {
	ID          string
	AccountID   string
	Name        string
	DisplayName string
	Type        FolderType
	UnreadCount int
	TotalCount  int
	Expanded    bool
	Children    []*Folder
}

type FolderType string

const (
	FolderTypeInbox   FolderType = "inbox"
	FolderTypeSent    FolderType = "sent"
	FolderTypeDrafts  FolderType = "drafts"
	FolderTypeArchive FolderType = "archive"
	FolderTypeTrash   FolderType = "trash"
	FolderTypeCustom  FolderType = "custom"
	FolderTypeSmart   FolderType = "smart" // AI-powered smart views
)

// Email represents an email message
type Email struct {
	ID          string
	AccountID   string
	FolderID    string
	MessageID   string
	Subject     string
	From        *EmailAddress
	To          []*EmailAddress
	Cc          []*EmailAddress
	Bcc         []*EmailAddress
	Date        time.Time
	Body        string
	BodyHTML    string
	BodyParams  map[string]string // Content-Type params (e.g. format=flowed)
	Attachments []*Attachment
	Flags       []string
	Labels      []string
	ThreadID    string
	InReplyTo   string
	References  []string

	// AI-enhanced fields
	Priority    Priority
	Category    string
	Summary     string
	ActionItems []string
	HasMeeting  bool
	Meeting     *MeetingInEmail

	// UI state
	Read     bool
	Starred  bool
	Selected bool
	Loaded   bool // Whether the full body has been fetched
}

type EmailAddress struct {
	Name    string
	Address string
}

type Attachment struct {
	ID          string
	Filename    string
	ContentType string
	Size        int64
	Data        []byte // Lazy loaded
}

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityUrgent
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "LOW"
	case PriorityMedium:
		return "MED"
	case PriorityHigh:
		return "HIGH"
	case PriorityUrgent:
		return "URGENT"
	default:
		return "UNKNOWN"
	}
}

func (p Priority) Icon() string {
	switch p {
	case PriorityLow:
		return "📧"
	case PriorityMedium:
		return "⚠ "
	case PriorityHigh:
		return "⚡"
	case PriorityUrgent:
		return "🔥"
	default:
		return "  "
	}
}

// MeetingInEmail represents meeting details extracted from an email
type MeetingInEmail struct {
	Title       string
	StartTime   time.Time
	EndTime     time.Time
	Duration    int // minutes
	Location    string
	Attendees   []*EmailAddress
	Description string
	Confidence  float64 // How confident Claude is about the extraction
}

// Meeting represents a calendar event
type Meeting struct {
	ID          string
	CalendarID  string
	Title       string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	AllDay      bool
	Attendees   []*Attendee
	Organizer   *EmailAddress
	Status      MeetingStatus
	Recurrence  *Recurrence

	// Link back to email if created from one
	FromEmailID string

	// Video conferencing
	ConferenceData *ConferenceData
}

type Attendee struct {
	Email    string
	Name     string
	Status   AttendeeStatus
	Optional bool
}

type AttendeeStatus string

const (
	AttendeeStatusNeedsAction AttendeeStatus = "needsAction"
	AttendeeStatusAccepted    AttendeeStatus = "accepted"
	AttendeeStatusDeclined    AttendeeStatus = "declined"
	AttendeeStatusTentative   AttendeeStatus = "tentative"
)

type MeetingStatus string

const (
	MeetingStatusConfirmed MeetingStatus = "confirmed"
	MeetingStatusTentative MeetingStatus = "tentative"
	MeetingStatusCancelled MeetingStatus = "cancelled"
)

type Recurrence struct {
	Frequency string // DAILY, WEEKLY, MONTHLY, YEARLY
	Interval  int
	Until     *time.Time
	Count     int
}

type ConferenceData struct {
	Type        string // zoom, meet, teams
	URL         string
	PhoneNumber string
	PIN         string
}

// TreeNode represents a node in the account/folder tree
type TreeNode struct {
	Type     TreeNodeType
	Account  *Account
	Folder   *Folder
	Level    int
	Expanded bool
	Parent   *TreeNode
	Children []*TreeNode
}

type TreeNodeType string

const (
	TreeNodeTypeAccount TreeNodeType = "account"
	TreeNodeTypeFolder  TreeNodeType = "folder"
)

// ClaudeConversation represents a chat with Claude
type ClaudeConversation struct {
	ID       string
	Messages []*ClaudeMessage
}

type ClaudeMessage struct {
	Role      string // "user" or "assistant"
	Content   string
	Timestamp time.Time
}

// PriorityScore represents Claude's email priority analysis
type PriorityScore struct {
	EmailID         string
	Score           int
	Priority        Priority
	Reason          string
	Category        string
	RequiresAction  bool
	SuggestedAction string
	Keywords        []string
}

// ScheduleMeetingRequest represents a request to schedule a meeting
type ScheduleMeetingRequest struct {
	FromEmailID string
	Title       string
	Attendees   []*EmailAddress
	StartTime   time.Time
	Duration    int // minutes
	Location    string
	Description string
}
