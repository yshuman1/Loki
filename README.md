# Loki - AI-Powered Email & Calendar TUI

Loki is a terminal-based email client with integrated calendar and AI assistance powered by Claude. Designed for developers who live in the terminal.

## Features

### Current (v0.1 - Skeleton)
- ✅ 3-panel email interface (accounts/folders, email list, preview)
- ✅ Tree-view navigation for accounts and folders
- ✅ Basic TUI framework with Bubbletea
- ✅ Modal overlays for Claude chat and meeting scheduler
- ✅ Keyboard-driven navigation

### Coming Soon
- 🔜 IMAP/Gmail OAuth integration
- 🔜 Claude AI agent with function calling
- 🔜 Email triage and priority scoring
- 🔜 Semantic email search with vector embeddings
- 🔜 Meeting detection and scheduling
- 🔜 Calendar views (month/week/day)
- 🔜 Natural language commands

## Architecture

```
┌─Accounts/Folders─┐┌─Emails────────────────┐┌─Preview──────────────────────────┐
│▼ Personal        ││                        ││From: alice@company.com            │
│  ● Inbox (23)    ││⚡ alice@company    2h  ││To: yasin@dispotag.com             │
│    Sent          ││  RE: Q4 Budget Review  ││Subject: RE: Q4 Budget Review      │
│    Drafts (2)    ││  Need approval by...   ││                                   │
│    Archive       ││                        ││[Email content...]                 │
│▶ DispoTag        ││⚠  bob@company      5h  ││                                   │
│▶ Work            ││  Team Sync Notes       ││┌─🤖 Claude─────────────────────┐ │
│                  ││  Action items from...  │││Priority: HIGH                  │ │
└──────────────────┘└────────────────────────┘└───────────────────────────────────┘
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- macOS, Linux, or WSL

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/loki.git
cd loki

# Initialize Go modules
go mod download

# Build
go build -o loki cmd/loki/main.go

# Run
./loki
```

### Development

```bash
# Run in development mode
go run cmd/loki/main.go

# Run tests
go test ./...

# Format code
go fmt ./...
```

## Keyboard Shortcuts

### Global
- `q` - Quit
- `E` - Email mode
- `C` - Calendar mode
- `c` - Open Claude chat
- `s` - Schedule meeting (from email)
- `t` - Triage inbox with AI
- `/` - Search
- `?` - Help

### Navigation
- `j/k` or `↓/↑` - Move up/down
- `h/l` or `←/→` - Move between panels / collapse/expand
- `Tab` - Cycle through panels
- `Enter` - Select/Open
- `g` - Go to top
- `G` - Go to bottom

### Tree View
- `o` - Expand node
- `O` - Collapse node
- `za` - Toggle expand/collapse
- `zM` - Collapse all
- `zR` - Expand all

### Email Actions
- `a` - Archive
- `d` - Delete
- `r` - Reply
- `R` - Reply all
- `f` - Forward
- `Space` - Mark read/unread
- `*` - Star/unstar

## Project Structure

```
loki/
├── cmd/loki/              # Main entry point
├── internal/
│   ├── models/            # Data structures
│   ├── tui/               # Terminal UI components
│   ├── email/             # Email client (IMAP/Gmail)
│   ├── calendar/          # Calendar integration
│   ├── agent/             # Claude AI agent
│   ├── storage/           # SQLite cache
│   └── config/            # Configuration
├── go.mod
└── README.md
```

## Configuration

Configuration file will be located at:
- macOS: `~/.config/loki/config.yaml`
- Linux: `~/.config/loki/config.yaml`

Example config (coming soon):
```yaml
accounts:
  - name: Personal
    email: you@example.com
    type: imap
    imap_server: imap.example.com
    imap_port: 993
    
  - name: Work
    email: you@company.com
    type: gmail
    # OAuth credentials stored in keychain

claude:
  api_key: sk-ant-... # Or use environment variable ANTHROPIC_API_KEY
  
calendar:
  default_duration: 30 # minutes
  default_location: zoom
```

## Roadmap

### Phase 1: Core Email (Week 1-2)
- [ ] IMAP client implementation
- [ ] Gmail OAuth flow
- [ ] Email fetching and caching
- [ ] SQLite storage
- [ ] Basic email operations (read, archive, delete)

### Phase 2: AI Integration (Week 2-3)
- [ ] Claude API integration
- [ ] Email triage and priority scoring
- [ ] Smart categorization
- [ ] Email summarization
- [ ] Action item extraction

### Phase 3: Calendar (Week 3-4)
- [ ] Google Calendar API
- [ ] CalDAV support
- [ ] Meeting detection in emails
- [ ] Quick meeting scheduler
- [ ] Calendar views

### Phase 4: Advanced Features (Week 4+)
- [ ] Vector search for semantic email queries
- [ ] Natural language commands
- [ ] Smart compose with Claude
- [ ] Meeting prep assistance
- [ ] Conflict detection
- [ ] Email rules and automation

## Technologies

- **TUI**: [Bubbletea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Email**: [go-imap](https://github.com/emersion/go-imap)
- **AI**: [Anthropic SDK](https://github.com/anthropics/anthropic-sdk-go)
- **Storage**: SQLite
- **Vector Search**: Qdrant

## Contributing

This is currently a personal project, but contributions are welcome! Please open an issue first to discuss what you'd like to change.

## License

MIT

## Acknowledgments

- Inspired by [lazygit](https://github.com/jesseduffield/lazygit) for the TUI design
- Built with [Bubbletea](https://github.com/charmbracelet/bubbletea)
- Powered by [Claude](https://www.anthropic.com/claude)
