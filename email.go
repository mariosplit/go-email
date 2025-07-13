// Package email provides a simple, provider-agnostic interface for sending emails
// through various providers including Outlook 365 and Gmail.
//
// Basic usage:
//
//	config := &email.Config{
//	    Provider: "outlook365",
//	    Outlook: &email.OutlookConfig{
//	        TenantID:     "your-tenant-id",
//	        ClientID:     "your-client-id",
//	        ClientSecret: "your-client-secret",
//	    },
//	}
//
//	client, err := email.NewClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	msg := &email.Message{
//	    From:    "sender@company.com",
//	    To:      []string{"recipient@example.com"},
//	    Subject: "Hello",
//	    Body:    "This is a test email!",
//	}
//
//	if err := client.Send(msg); err != nil {
//	    log.Fatal(err)
//	}
package email

import (
	"context"
	"fmt"
	"time"
)

// Message represents an email message with all necessary fields for sending.
type Message struct {
	// From is the sender's email address (required)
	From string

	// To contains the primary recipient email addresses (at least one required)
	To []string

	// Cc contains carbon copy recipient email addresses (optional)
	Cc []string

	// Bcc contains blind carbon copy recipient email addresses (optional)
	Bcc []string

	// Subject is the email subject line (required)
	Subject string

	// Body is the email content (required)
	Body string

	// HTML indicates whether the body should be treated as HTML.
	// If false, the body is treated as plain text.
	HTML bool

	// Attachments contains file attachments (optional)
	Attachments []Attachment
}

// Attachment represents a file attachment for an email.
type Attachment struct {
	// Filename is the name of the file as it will appear in the email
	Filename string

	// Content is the file content as bytes
	Content []byte

	// MimeType is the MIME type of the file (optional).
	// If empty, it will be automatically detected based on the filename.
	MimeType string
}

// Provider is the interface that all email providers must implement.
// This allows for easy addition of new email providers.
type Provider interface {
	// Send sends an email message using the provider's implementation.
	// The context can be used for timeout and cancellation.
	Send(ctx context.Context, msg *Message) error
}

// Config holds the configuration for creating email providers.
// Only one provider configuration should be set.
type Config struct {
	// Provider specifies which email provider to use.
	// Supported values: "outlook365", "gmail"
	Provider string

	// Outlook contains Outlook 365 specific configuration.
	// Required when Provider is "outlook365".
	Outlook *OutlookConfig

	// Gmail contains Gmail specific configuration.
	// Required when Provider is "gmail".
	Gmail *GmailConfig

	// Custom is reserved for future provider extensions
	Custom map[string]interface{}
}

// OutlookConfig holds Outlook 365 specific configuration for OAuth2 authentication.
type OutlookConfig struct {
	// TenantID is the Azure AD tenant ID
	TenantID string

	// ClientID is the Azure AD application client ID
	ClientID string

	// ClientSecret is the Azure AD application client secret
	ClientSecret string
}

// GmailConfig holds Gmail specific configuration for OAuth2 authentication.
type GmailConfig struct {
	// CredentialsJSON contains the OAuth2 credentials downloaded from Google Cloud Console
	CredentialsJSON []byte

	// TokenJSON contains the stored OAuth2 token.
	// If not provided, authentication will be required on first use.
	TokenJSON []byte
}

// Client is the main email client that wraps a provider implementation.
// It is thread-safe and can be used concurrently.
type Client struct {
	provider Provider
}

// NewClient creates a new email client with the specified configuration.
// It returns an error if the configuration is invalid or the provider
// fails to initialize.
//
// Example:
//
//	config := &email.Config{
//	    Provider: "gmail",
//	    Gmail: &email.GmailConfig{
//	        CredentialsJSON: credentialsJSON,
//	        TokenJSON:       tokenJSON,
//	    },
//	}
//
//	client, err := email.NewClient(config)
func NewClient(config *Config) (*Client, error) {
	var provider Provider
	var err error

	switch config.Provider {
	case "outlook365":
		if config.Outlook == nil {
			return nil, fmt.Errorf("outlook configuration is required")
		}
		provider, err = newOutlookProvider(config.Outlook)
	case "gmail":
		if config.Gmail == nil {
			return nil, fmt.Errorf("gmail configuration is required")
		}
		provider, err = newGmailProvider(config.Gmail)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &Client{provider: provider}, nil
}

// Send sends an email message with a default timeout of 30 seconds.
// It validates the message before sending and returns an error if
// validation fails or the send operation fails.
func (c *Client) Send(msg *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.SendWithContext(ctx, msg)
}

// SendWithContext sends an email message with a custom context.
// This allows for custom timeouts, cancellation, and passing request-scoped values.
// The message is validated before sending.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
//	defer cancel()
//
//	err := client.SendWithContext(ctx, msg)
func (c *Client) SendWithContext(ctx context.Context, msg *Message) error {
	// Validate message
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	return c.provider.Send(ctx, msg)
}

// Validate checks if the message has all required fields.
// It returns an error describing the first validation failure found.
func (m *Message) Validate() error {
	if m.From == "" {
		return fmt.Errorf("from address is required")
	}
	if len(m.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if m.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if m.Body == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

// QuickSend provides a simple way to send an email with minimal configuration.
// This is useful for simple use cases where you don't need to reuse the client.
//
// Example:
//
//	err := email.QuickSend("gmail",
//	    &email.GmailConfig{
//	        CredentialsJSON: creds,
//	        TokenJSON:       token,
//	    },
//	    "from@example.com",
//	    "to@example.com",
//	    "Subject",
//	    "Body")
func QuickSend(provider string, creds interface{}, from, to, subject, body string) error {
	config := &Config{Provider: provider}

	switch provider {
	case "outlook365":
		if outlook, ok := creds.(*OutlookConfig); ok {
			config.Outlook = outlook
		} else {
			return fmt.Errorf("invalid credentials for outlook365")
		}
	case "gmail":
		if gmail, ok := creds.(*GmailConfig); ok {
			config.Gmail = gmail
		} else {
			return fmt.Errorf("invalid credentials for gmail")
		}
	}

	client, err := NewClient(config)
	if err != nil {
		return err
	}

	msg := &Message{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Body:    body,
	}

	return client.Send(msg)
}
