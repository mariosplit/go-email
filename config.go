// config.go - Configuration helpers for the email package
package email

import (
	"fmt"
	"os"
)

// ConfigFromEnv creates an email configuration from environment variables.
// This is a convenient way to configure the email client without hardcoding credentials.
//
// Environment variables:
//   - EMAIL_PROVIDER: The email provider to use ("outlook365" or "gmail"), defaults to "outlook365"
//   - For Outlook 365:
//   - OUTLOOK_TENANT_ID: Azure AD tenant ID (required)
//   - OUTLOOK_CLIENT_ID: Azure AD application client ID (required)
//   - OUTLOOK_CLIENT_SECRET: Azure AD application client secret (required)
//   - For Gmail:
//   - GMAIL_CREDENTIALS_FILE: Path to the OAuth2 credentials JSON file (required)
//   - GMAIL_TOKEN_FILE: Path to the OAuth2 token JSON file (defaults to "token.json")
//
// Example:
//
//	os.Setenv("EMAIL_PROVIDER", "gmail")
//	os.Setenv("GMAIL_CREDENTIALS_FILE", "credentials.json")
//	os.Setenv("GMAIL_TOKEN_FILE", "token.json")
//
//	config, err := email.ConfigFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
func ConfigFromEnv() (*Config, error) {
	provider := os.Getenv("EMAIL_PROVIDER")
	if provider == "" {
		provider = "outlook365" // default
	}

	config := &Config{
		Provider: provider,
	}

	switch provider {
	case "outlook365":
		outlook, err := outlookConfigFromEnv()
		if err != nil {
			return nil, fmt.Errorf("outlook config error: %w", err)
		}
		config.Outlook = outlook

	case "gmail":
		gmail, err := gmailConfigFromEnv()
		if err != nil {
			return nil, fmt.Errorf("gmail config error: %w", err)
		}
		config.Gmail = gmail

	default:
		return nil, fmt.Errorf("unsupported email provider: %s", provider)
	}

	return config, nil
}

// outlookConfigFromEnv reads Outlook 365 configuration from environment variables
func outlookConfigFromEnv() (*OutlookConfig, error) {
	config := &OutlookConfig{
		TenantID:     os.Getenv("OUTLOOK_TENANT_ID"),
		ClientID:     os.Getenv("OUTLOOK_CLIENT_ID"),
		ClientSecret: os.Getenv("OUTLOOK_CLIENT_SECRET"),
	}

	if config.TenantID == "" {
		return nil, fmt.Errorf("OUTLOOK_TENANT_ID is required")
	}
	if config.ClientID == "" {
		return nil, fmt.Errorf("OUTLOOK_CLIENT_ID is required")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("OUTLOOK_CLIENT_SECRET is required")
	}

	return config, nil
}

// gmailConfigFromEnv reads Gmail configuration from environment variables
func gmailConfigFromEnv() (*GmailConfig, error) {
	credsFile := os.Getenv("GMAIL_CREDENTIALS_FILE")
	if credsFile == "" {
		return nil, fmt.Errorf("GMAIL_CREDENTIALS_FILE is required")
	}

	tokenFile := os.Getenv("GMAIL_TOKEN_FILE")
	if tokenFile == "" {
		tokenFile = "token.json" // default
	}

	creds, err := os.ReadFile(credsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	return &GmailConfig{
		CredentialsJSON: creds,
		TokenJSON:       token,
	}, nil
}

// QuickClientFromEnv creates a client using environment variables.
// This combines ConfigFromEnv and NewClient for convenience.
//
// Example:
//
//	client, err := email.QuickClientFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = client.Send(&email.Message{
//	    From:    "sender@example.com",
//	    To:      []string{"recipient@example.com"},
//	    Subject: "Test",
//	    Body:    "Hello!",
//	})
func QuickClientFromEnv() (*Client, error) {
	config, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(config)
}
