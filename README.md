# go-email

[![Go Version](https://img.shields.io/github/go-mod/go-version/go-email/go-email)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GoDoc](https://pkg.go.dev/badge/github.com/go-email/go-email.svg)](https://pkg.go.dev/github.com/go-email/go-email)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-email/go-email)](https://goreportcard.com/report/github.com/go-email/go-email)

A simple, provider-agnostic Go package for sending emails through Outlook 365 and Gmail.

## 🚀 Features

- **Simple, intuitive API** - Send emails with just a few lines of code
- **Multiple Providers** - Support for Outlook 365 (Microsoft Graph) and Gmail (Gmail API)
- **Rich Email Features** - HTML content, attachments, CC/BCC recipients
- **Secure Authentication** - OAuth2 authentication for both providers
- **Environment Configuration** - Easy setup via environment variables
- **Zero External Dependencies** - Only provider SDKs required
- **Context Support** - Full context.Context support for timeouts and cancellation
- **Well Tested** - Comprehensive test coverage

## 📦 Installation

```bash
go get github.com/go-email/go-email@v1.0.0
```

## 🏃 Quick Start

### Outlook 365

```go
package main

import (
    "log"
    "github.com/go-email/go-email"
)

func main() {
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

    msg := &email.Message{
        From:    "sender@company.com",
        To:      []string{"recipient@example.com"},
        Subject: "Hello from go-email",
        Body:    "This is a test email!",
    }

    if err := client.Send(msg); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Email sent successfully!")
}
```

### Gmail

```go
package main

import (
    "log"
    "os"
    "github.com/go-email/go-email"
)

func main() {
    // Read credentials and token
    creds, _ := os.ReadFile("credentials.json")
    token, _ := os.ReadFile("token.json")

    config := &email.Config{
        Provider: "gmail",
        Gmail: &email.GmailConfig{
            CredentialsJSON: creds,
            TokenJSON:       token,
        },
    }

    client, err := email.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }

    msg := &email.Message{
        From:    "sender@gmail.com",
        To:      []string{"recipient@example.com"},
        Subject: "Hello from go-email",
        Body:    "This is a test email!",
    }

    if err := client.Send(msg); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Email sent successfully!")
}
```

## 📚 Documentation

- [Integration Guide](INTEGRATION.md) - Comprehensive guide for integrating into your applications
- [Gmail Setup Guide](docs/GMAIL-SETUP.md) - Step-by-step Gmail configuration
- [Outlook Setup Guide](docs/OUTLOOK-SETUP.md) - Outlook 365 configuration
- [Examples](examples/) - Working examples for common use cases
- [API Documentation](https://pkg.go.dev/github.com/go-email/go-email) - Complete API reference

## 🔧 Configuration

### Environment Variables

Configure the package using environment variables:

```bash
# Provider selection
EMAIL_PROVIDER=outlook365  # or "gmail"

# Outlook 365
OUTLOOK_TENANT_ID=your-tenant-id
OUTLOOK_CLIENT_ID=your-client-id
OUTLOOK_CLIENT_SECRET=your-client-secret

# Gmail
GMAIL_CREDENTIALS_FILE=path/to/credentials.json
GMAIL_TOKEN_FILE=path/to/token.json
```

Then use the simplified client creation:

```go
client, err := email.QuickClientFromEnv()
```

### Provider Setup

#### Outlook 365 Setup

1. Register an application in [Azure Portal](https://portal.azure.com)
2. Grant `Mail.Send` permission
3. Create a client secret
4. Use the tenant ID, client ID, and client secret in your configuration

See the [Outlook Setup Guide](docs/OUTLOOK-SETUP.md) for detailed instructions.

#### Gmail Setup

1. Create a project in [Google Cloud Console](https://console.cloud.google.com)
2. Enable Gmail API
3. Create OAuth2 credentials (Desktop application type)
4. Download credentials.json
5. Run the authentication to get your token

See the [Gmail Setup Guide](docs/GMAIL-SETUP.md) for detailed instructions.

## 📧 Advanced Usage

### HTML Email with Attachments

```go
// Read file content
content, _ := os.ReadFile("document.pdf")

msg := &email.Message{
    From:    "sender@company.com",
    To:      []string{"recipient@example.com"},
    Cc:      []string{"cc@example.com"},
    Bcc:     []string{"bcc@example.com"},
    Subject: "Monthly Report",
    Body:    `
        <h1>Monthly Report</h1>
        <p>Please find the attached report for this month.</p>
        <p>Best regards,<br>Your Team</p>
    `,
    HTML:    true,
    Attachments: []email.Attachment{
        {
            Filename: "report.pdf",
            Content:  content,
            MimeType: "application/pdf",
        },
    },
}

err := client.Send(msg)
```

### Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := client.SendWithContext(ctx, msg)
```

### Error Handling

```go
err := client.Send(msg)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "authentication"):
        // Handle auth errors - check credentials
        log.Printf("Authentication failed: %v", err)
    case strings.Contains(err.Error(), "rate limit"):
        // Handle rate limiting - implement backoff
        log.Printf("Rate limited, retry later: %v", err)
    case strings.Contains(err.Error(), "invalid recipient"):
        // Handle validation errors
        log.Printf("Invalid recipient address: %v", err)
    default:
        // Handle other errors
        log.Printf("Failed to send email: %v", err)
    }
}
```

## 🏗️ Architecture

The package follows a clean architecture with provider abstraction:

```
email.Client
    ├── Provider Interface
    │   ├── OutlookProvider (Microsoft Graph API)
    │   └── GmailProvider (Gmail API)
    └── Message
        ├── Recipients (To, Cc, Bcc)
        ├── Content (Plain/HTML)
        └── Attachments
```

## 🧪 Testing

Run the test suite:

```bash
go test ./...
```

For integration tests with real email sending:

```bash
# Set up test credentials
export EMAIL_PROVIDER=gmail
export TEST_FROM_EMAIL=test@example.com
export TEST_TO_EMAIL=recipient@example.com

# Run integration tests
go test -tags=integration ./...
```

## 📊 Benchmarks

```bash
go test -bench=. ./...
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for more details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Microsoft Graph SDK for Go
- Google Gmail API Client Library for Go
- The Go community for excellent tools and libraries

## 📞 Support

- 📧 Email: support@go-email.dev
- 💬 Discussions: [GitHub Discussions](https://github.com/go-email/go-email/discussions)
- 🐛 Issues: [GitHub Issues](https://github.com/go-email/go-email/issues)

## 🗺️ Roadmap

- [ ] Add support for SendGrid provider
- [ ] Add support for AWS SES provider
- [ ] Add email template engine
- [ ] Add webhook support for email events
- [ ] Add batch sending optimization
- [ ] Add email validation utilities

---

Made with ❤️ by the go-email team
