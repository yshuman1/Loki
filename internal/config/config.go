package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yshuman1/loki/internal/models"
)

// Config represents the application configuration
type Config struct {
	Accounts []AccountConfig `json:"accounts"`
	Claude   ClaudeConfig    `json:"claude"`
	Calendar CalendarConfig  `json:"calendar"`
}

// AccountConfig represents an email account configuration
type AccountConfig struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Type       string `json:"type"` // "imap" or "gmail"
	IMAPServer string `json:"imap_server,omitempty"`
	IMAPPort   int    `json:"imap_port,omitempty"`
	SMTPServer string `json:"smtp_server,omitempty"`
	SMTPPort   int    `json:"smtp_port,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"` // Will use keychain in production
}

// ClaudeConfig represents Claude AI configuration
type ClaudeConfig struct {
	APIKey string `json:"api_key,omitempty"` // Can also use ANTHROPIC_API_KEY env var
}

// CalendarConfig represents calendar configuration
type CalendarConfig struct {
	DefaultDuration int    `json:"default_duration"` // minutes
	DefaultLocation string `json:"default_location"`
}

// Load loads configuration from file
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		return createDefaultConfig(configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		config.Claude.APIKey = apiKey
	}

	return &config, nil
}

// Save saves configuration to file
func Save(config *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ToModels converts config accounts to model accounts
func (c *Config) ToModels() []*models.Account {
	accounts := make([]*models.Account, len(c.Accounts))
	for i, acc := range c.Accounts {
		var accType models.AccountType
		if acc.Type == "gmail" {
			accType = models.AccountTypeGmail
		} else {
			accType = models.AccountTypeIMAP
		}

		accounts[i] = &models.Account{
			ID:         fmt.Sprintf("%d", i+1),
			Name:       acc.Name,
			Email:      acc.Email,
			Type:       accType,
			IMAPServer: acc.IMAPServer,
			IMAPPort:   acc.IMAPPort,
			SMTPServer: acc.SMTPServer,
			SMTPPort:   acc.SMTPPort,
			Credentials: &models.Credentials{
				Username: acc.Username,
				Password: acc.Password,
			},
			Expanded: false,
		}
	}
	return accounts
}

func getConfigPath() (string, error) {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	return filepath.Join(configDir, "loki", "config.json"), nil
}

func createDefaultConfig(path string) (*Config, error) {
	config := &Config{
		Accounts: []AccountConfig{
			{
				Name:       "Example IMAP",
				Email:      "you@example.com",
				Type:       "imap",
				IMAPServer: "imap.example.com",
				IMAPPort:   993,
				SMTPServer: "smtp.example.com",
				SMTPPort:   587,
				Username:   "you@example.com",
				Password:   "",
			},
		},
		Claude: ClaudeConfig{
			APIKey: "",
		},
		Calendar: CalendarConfig{
			DefaultDuration: 30,
			DefaultLocation: "zoom",
		},
	}

	// Create config directory
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save default config
	if err := Save(config); err != nil {
		return nil, err
	}

	return config, nil
}

// GetConfigPath returns the path to the config file (for user reference)
func GetConfigPath() (string, error) {
	return getConfigPath()
}
