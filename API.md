# API Documentation

## Package email

The `email` package provides a simple, provider-agnostic interface for sending emails through various providers.

### Constants

```go
const (
    Version      = "v1.0.0"
    VersionMajor = 1
    VersionMinor = 0
    VersionPatch = 0
)
```

### Types

#### type Config

```go
type Config struct {
    Provider string                 // "outlook365" or "gmail"
    Outlook  *OutlookConfig        // Outlook-specific config
    Gmail    *GmailConfig          // Gmail-specific config
    Custom   map[string]interface{} // For future providers
}
```

Config holds the configuration for creating email providers.

#### type OutlookConfig

```go
type OutlookConfig struct {
    TenantID     string
    ClientID     string
    ClientSecret string
}
```

OutlookConfig holds Outlook 365 specific configuration.

#### type GmailConfig

```go
type GmailConfig struct {
    CredentialsJSON []byte // OAuth2 credentials JSON content
    TokenJSON       []byte // Stored token (optional)
}
```

GmailConfig holds Gmail specific configuration.

#### type Message

```go
type Message struct {
    From        string
    To          []string
    Cc          []string
    Bcc         []string
    Subject     string
    Body        string
    HTML        bool         // If true, body is treated as HTML
    Attachments []Attachment
}
```

Message represents an email message.

##### Methods

###### func (*Message) Validate

```go
func (m *Message) Validate() error
```

Validate checks if the message has all required fields.

#### type Attachment

```go
type Attachment struct {
    Filename string
    Content  []byte
    MimeType string // Optional: will be auto-detected if empty
}
```

Attachment represents an email attachment.

#### type Client

```go
type Client struct {
    // contains filtered or unexported fields
}
```

Client is the main email client.

##### Methods

###### func NewClient

```go
func NewClient(config *Config) (*Client, error)
```

NewClient creates a new email client with the specified configuration.

**Example:**

```go
config := &email.Config{
    Provider: "outlook365",
    Outlook: &email.OutlookConfig{
        TenantID:     "your-tenant-id",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
    },
}

client, err := email.NewClient(config)
if err != nil {
    log.Fatal(err)
}
```

###### func (*Client) Send

```go
func (c *Client) Send(msg *Message) error
```

Send sends an email message with a default timeout of 30 seconds.

**Example:**

```go
msg := &email.Message{
    From:    "sender@example.com",
    To:      []string{"recipient@example.com"},
    Subject: "Test Email",
    Body:    "This is a test email.",
}

err := client.Send(msg)
```

###### func (*Client) SendWithContext

```go
func (c *Client) SendWithContext(ctx context.Context, msg *Message) error
```

SendWithContext sends an email message with a custom context.

**Example:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

err := client.SendWithContext(ctx, msg)
```

#### type Provider

```go
type Provider interface {
    Send(ctx context.Context, msg *Message) error
}
```

Provider is the interface that all email providers must implement.

### Functions

#### func ConfigFromEnv

```go
func ConfigFromEnv() (*Config, error)
```

ConfigFromEnv creates an email configuration from environment variables.

**Environment Variables:**

- `EMAIL_PROVIDER` - Provider name ("outlook365" or "gmail")
- `OUTLOOK_TENANT_ID` - Azure AD tenant ID
- `OUTLOOK_CLIENT_ID` - Azure AD application client ID
- `OUTLOOK_CLIENT_SECRET` - Azure AD application client secret
- `GMAIL_CREDENTIALS_FILE` - Path to Gmail OAuth2 credentials JSON
- `GMAIL_TOKEN_FILE` - Path to Gmail OAuth2 token JSON

#### func QuickClientFromEnv

```go
func QuickClientFromEnv() (*Client, error)
```

QuickClientFromEnv creates a client using environment variables.

#### func QuickSend

```go
func QuickSend(provider string, creds interface{}, from, to, subject, body string) error
```

QuickSend provides a simple way to send an email with minimal configuration.

**Example:**

```go
// Outlook 365
err := email.QuickSend("outlook365", 
    &email.OutlookConfig{
        TenantID:     "...",
        ClientID:     "...",
        ClientSecret: "...",
    },
    "from@company.com",
    "to@example.com",
    "Subject",
    "Body")

// Gmail
err := email.QuickSend("gmail",
    &email.GmailConfig{
        CredentialsJSON: creds,
        TokenJSON:       token,
    },
    "from@gmail.com",
    "to@example.com",
    "Subject",
    "Body")
```

#### func GetVersion

```go
func GetVersion() string
```

GetVersion returns the full version string.

#### func GetVersionInfo

```go
func GetVersionInfo() VersionInfo
```

GetVersionInfo returns detailed version information.

### Error Handling

The package returns descriptive errors that can be used to determine the type of failure:

```go
err := client.Send(msg)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "authentication"):
        // Authentication failure
    case strings.Contains(err.Error(), "invalid message"):
        // Message validation failure
    case strings.Contains(err.Error(), "rate limit"):
        // Rate limiting error
    case strings.Contains(err.Error(), "network"):
        // Network error
    default:
        // Other error
    }
}
```

### Common Error Messages

- `"outlook configuration is required"` - Missing Outlook configuration
- `"gmail configuration is required"` - Missing Gmail configuration
- `"unsupported provider: %s"` - Unknown provider specified
- `"from address is required"` - Missing sender address
- `"at least one recipient is required"` - No recipients specified
- `"subject is required"` - Missing email subject
- `"body is required"` - Missing email body
- `"authentication failed"` - Provider authentication failed
- `"rate limit exceeded"` - Provider rate limit hit

### Best Practices

1. **Reuse Clients**: Create one client and reuse it for multiple emails
2. **Handle Errors**: Always check and handle errors appropriately
3. **Use Context**: Use context for proper timeout and cancellation
4. **Validate Early**: Validate email addresses before sending
5. **Rate Limiting**: Implement rate limiting for bulk sending
6. **Secure Credentials**: Never hardcode credentials in source code

### Thread Safety

The Client is thread-safe and can be used concurrently from multiple goroutines.

### Performance Considerations

- Clients maintain persistent connections where possible
- Use `SendWithContext` for better control over timeouts
- For bulk sending, consider implementing a queue system
- Attachments are held in memory, be mindful of large files

### Limitations

- Maximum attachment size depends on the provider (typically 25MB)
- Rate limits apply based on provider policies
- Some providers may have daily sending limits
- HTML rendering varies by email client
