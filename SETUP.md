# Loki Email Setup Guide

## Files to Download and Update

Download these files and replace in your repo:

1. **New Files:**
   - `internal/config/config.go`
   - `internal/email/client.go`
   - `internal/email/manager.go`

2. **Updated Files:**
   - `internal/tui/app.go`
   - `internal/tui/tree.go`

## Configuration

Loki will create a default config file at `~/.config/loki/config.json` on first run.

### Step 1: Run Loki Once

```bash
go run cmd/loki/main.go
```

It will create the config file and show an error (expected).

### Step 2: Edit Configuration

Edit `~/.config/loki/config.json`:

```json
{
  "accounts": [
    {
      "name": "Personal",
      "email": "your.email@gmail.com",
      "type": "imap",
      "imap_server": "imap.gmail.com",
      "imap_port": 993,
      "smtp_server": "smtp.gmail.com",
      "smtp_port": 587,
      "username": "your.email@gmail.com",
      "password": "your-app-password-here"
    }
  ],
  "claude": {
    "api_key": ""
  },
  "calendar": {
    "default_duration": 30,
    "default_location": "zoom"
  }
}
```

### Step 3: Gmail App Password

**For Gmail accounts**, you need an App Password:

1. Go to https://myaccount.google.com/apppasswords
2. Create a new app password for "Mail"
3. Copy the 16-character password
4. Use it in the `password` field in your config

**Note:** Regular Gmail passwords won't work with IMAP. You MUST use an App Password.

### Step 4: Run Loki

```bash
go run cmd/loki/main.go
```

Now you should see:
- Your real email accounts in the tree
- Real folders when you expand an account (press `o` or `Enter`)
- Real emails when you select a folder (press `Space`)

## Usage

### Navigation
- `j/k` or `↓/↑` - Move in tree/list
- `o` or `Enter` - Expand account
- `O` or `h` - Collapse account
- `Space` - Select folder (loads emails)
- `Tab` - Switch between panels
- `q` - Quit

### Current Features
✅ Connect to IMAP accounts
✅ List folders
✅ Fetch emails from folders
✅ Display email list
✅ Navigate emails

### Coming Soon
- Email preview with full body
- Claude AI integration
- Meeting scheduling
- Calendar views

## Troubleshooting

### "Failed to connect"
- Check your IMAP server and port
- Verify username and password
- For Gmail, make sure you're using an App Password

### "No emails showing"
- Make sure you've selected a folder (press `Space` on a folder)
- Check that the folder has emails in your email client

### Config file location
- macOS/Linux: `~/.config/loki/config.json`
- To find it: Look at the error message on first run

## Multiple Accounts

You can add multiple accounts in the config:

```json
{
  "accounts": [
    {
      "name": "Personal",
      "email": "personal@gmail.com",
      "type": "imap",
      "imap_server": "imap.gmail.com",
      "imap_port": 993,
      "username": "personal@gmail.com",
      "password": "app-password-1"
    },
    {
      "name": "Work",
      "email": "work@company.com",
      "type": "imap",
      "imap_server": "imap.company.com",
      "imap_port": 993,
      "username": "work@company.com",
      "password": "work-password"
    }
  ]
}
```

## Next Steps

Once you have email working, we'll add:
1. Full email body preview
2. Claude AI for triage and summarization
3. Meeting scheduling
4. Calendar integration

Try it out and let me know how it works!
