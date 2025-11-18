module github.com/yshuman1/loki

go 1.21

require (
	// TUI Framework
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/lipgloss v0.9.1
	github.com/charmbracelet/bubbles v0.17.1

	// Email
	github.com/emersion/go-imap/v2 v2.0.0-beta.1
	github.com/emersion/go-message v0.18.0
	github.com/emersion/go-sasl v0.0.0-20231106173351-e73c9f7bad43

	// Gmail API
	google.golang.org/api v0.150.0
	golang.org/x/oauth2 v0.15.0

	// Claude AI
	github.com/anthropics/anthropic-sdk-go v0.1.0

	// Storage
	github.com/mattn/go-sqlite3 v1.14.18
	modernc.org/sqlite v1.27.0

	// Vector search (for semantic email search)
	github.com/qdrant/go-client v1.7.0

	// Utilities
	github.com/google/uuid v1.5.0
	github.com/keybase/go-keychain v0.0.0-20231219164618-57a3676c3af6
)
