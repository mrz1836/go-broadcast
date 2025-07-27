// Package types defines configuration types for the notification system
package types

// SlackConfig holds Slack-specific configuration
type SlackConfig struct {
	WebhookURL   string            `json:"webhook_url"`
	Channel      string            `json:"channel,omitempty"`
	Username     string            `json:"username,omitempty"`
	IconEmoji    string            `json:"icon_emoji,omitempty"`
	IconURL      string            `json:"icon_url,omitempty"`
	Enabled      bool              `json:"enabled"`
	Timeout      int               `json:"timeout"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// DiscordConfig holds Discord-specific configuration
type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
	Username   string `json:"username,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	EmbedColor int    `json:"embed_color,omitempty"`
	Enabled    bool   `json:"enabled"`
	Timeout    int    `json:"timeout"`
}

// EmailConfig holds email-specific configuration
type EmailConfig struct {
	SMTPHost    string   `json:"smtp_host"`
	SMTPPort    int      `json:"smtp_port"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	FromEmail   string   `json:"from_email"`
	FromName    string   `json:"from_name,omitempty"`
	ToEmails    []string `json:"to_emails"`
	CCEmails    []string `json:"cc_emails,omitempty"`
	BCCEmails   []string `json:"bcc_emails,omitempty"`
	UseTLS      bool     `json:"use_tls"`
	UseStartTLS bool     `json:"use_starttls"`
	Enabled     bool     `json:"enabled"`
	Timeout     int      `json:"timeout"`
}

// WebhookConfig holds webhook-specific configuration
type WebhookConfig struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers,omitempty"`
	ContentType  string            `json:"content_type"`
	AuthType     string            `json:"auth_type,omitempty"`
	AuthToken    string            `json:"auth_token,omitempty"`
	AuthUsername string            `json:"auth_username,omitempty"`
	AuthPassword string            `json:"auth_password,omitempty"`
	Enabled      bool              `json:"enabled"`
	Timeout      int               `json:"timeout"`
}

// TeamsConfig holds Microsoft Teams-specific configuration
type TeamsConfig struct {
	WebhookURL    string `json:"webhook_url"`
	ThemeColor    string `json:"theme_color,omitempty"`
	ActivityTitle string `json:"activity_title,omitempty"`
	ActivityImage string `json:"activity_image,omitempty"`
	Enabled       bool   `json:"enabled"`
	Timeout       int    `json:"timeout"`
}
