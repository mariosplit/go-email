# Gmail Setup Guide

This guide walks you through setting up Gmail email sending using the go-email package.

## Prerequisites

- Google account
- Access to Google Cloud Console
- Go 1.20 or later

## Step 1: Create a Google Cloud Project

1. **Go to Google Cloud Console**
   - Visit [https://console.cloud.google.com](https://console.cloud.google.com)
   - Sign in with your Google account

2. **Create a New Project**
   - Click the project dropdown at the top
   - Click "New Project"
   - Enter a project name (e.g., "go-email-project")
   - Click "Create"

3. **Enable Gmail API**
   - In the search bar, type "Gmail API"
   - Click on "Gmail API" from the results
   - Click "Enable"

## Step 2: Create OAuth2 Credentials

1. **Navigate to Credentials**
   - In the left sidebar, click "APIs & Services" > "Credentials"

2. **Configure OAuth Consent Screen**
   - Click "Configure Consent Screen"
   - Choose user type:
     - **Internal**: For G Suite/Workspace users only
     - **External**: For any Google account (requires app review for production)
   - Fill in required fields:
     - App name: "go-email"
     - User support email: Your email
     - Developer contact: Your email
   - Click "Save and Continue"
   - On Scopes page, click "Add or Remove Scopes"
   - Search and select: `https://www.googleapis.com/auth/gmail.send`
   - Click "Update" then "Save and Continue"
   - Complete the remaining steps

3. **Create OAuth2 Credentials**
   - Go back to "Credentials" page
   - Click "Create Credentials" > "OAuth client ID"
   - Application type: "Desktop app"
   - Name: "go-email-client"
   - Click "Create"
   - Download the JSON file (click "Download JSON")
   - Save it as `credentials.json`

## Step 3: Authenticate and Get Token

### First-Time Authentication

```go
package main

import (
    "log"
    "os"
    "github.com/go-email/go-email"
)

func main() {
    // One-time authentication to get token
    token, err := email.AuthenticateGmailFromFile("credentials.json")
    if err != nil {
        log.Fatal(err)
    }
    
    // Save token for future use
    err = os.WriteFile("token.json", token, 0600)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Authentication successful! Token saved to token.json")
}
```

Run this program:
1. It will print a URL - open it in your browser
2. Sign in with your Google account
3. Grant the requested permissions
4. Copy the authorization code
5. Paste it back in the terminal
6. The token will be saved to `token.json`

## Step 4: Send Emails

### Using the Saved Token

```go
package main

import (
    "log"
    "os"
    "github.com/go-email/go-email"
)

func main() {
    // Read credentials and token
    creds, err := os.ReadFile("credentials.json")
    if err != nil {
        log.Fatal(err)
    }
    
    token, err := os.ReadFile("token.json")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create email client
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
    
    // Send email
    msg := &email.Message{
        From:    "your-email@gmail.com",
        To:      []string{"recipient@example.com"},
        Subject: "Hello from go-email",
        Body:    "This email was sent using the Gmail API!",
    }
    
    if err := client.Send(msg); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Email sent successfully!")
}
```

## Using Environment Variables

### Setup

Create a `.env` file:

```env
EMAIL_PROVIDER=gmail
GMAIL_CREDENTIALS_FILE=credentials.json
GMAIL_TOKEN_FILE=token.json
```

### Code

```go
package main

import (
    "log"
    "github.com/go-email/go-email"
    "github.com/joho/godotenv"
)

func main() {
    // Load environment variables
    godotenv.Load()
    
    // Create client from environment
    client, err := email.QuickClientFromEnv()
    if err != nil {
        log.Fatal(err)
    }
    
    // Send email...
}
```

## Important Notes

### Gmail Sending Limits

- **Free Gmail accounts**: 500 emails per day
- **Google Workspace**: 2,000 emails per day
- **Per-minute limits also apply**

### Security Considerations

1. **Protect your credentials**
   - Never commit `credentials.json` or `token.json` to version control
   - Add them to `.gitignore`
   - Use secure storage in production

2. **Token expiration**
   - Tokens may expire after extended periods
   - Implement token refresh logic for production apps
   - The package will return an authentication error when token expires

3. **Scopes**
   - We only request `gmail.send` scope (minimum required)
   - Don't request unnecessary permissions

## Troubleshooting

### Common Issues

1. **"invalid_grant" error**
   - Token has expired
   - Re-run the authentication process
   - Delete old `token.json` and authenticate again

2. **"403 Forbidden"**
   - Check if Gmail API is enabled in your project
   - Verify OAuth consent screen is configured
   - Ensure correct scopes are granted

3. **"invalid credentials"**
   - Verify `credentials.json` is from the correct project
   - Check if the file is properly formatted JSON

4. **Rate limit errors**
   - Implement exponential backoff
   - Reduce sending frequency
   - Consider upgrading to Google Workspace

### Testing Your Setup

```go
package main

import (
    "fmt"
    "log"
    "github.com/go-email/go-email"
)

func main() {
    // Test authentication
    fmt.Println("Testing Gmail setup...")
    
    client, err := email.QuickClientFromEnv()
    if err != nil {
        log.Fatalf("❌ Failed to create client: %v", err)
    }
    fmt.Println("✓ Client created successfully")
    
    // Test sending
    msg := &email.Message{
        From:    "your-email@gmail.com", // Must match authenticated account
        To:      []string{"test@example.com"},
        Subject: "go-email Gmail Test",
        Body:    "If you see this, Gmail is configured correctly!",
    }
    
    if err := client.Send(msg); err != nil {
        log.Fatalf("❌ Failed to send: %v", err)
    }
    
    fmt.Println("✓ Email sent successfully!")
    fmt.Println("✅ Gmail setup is complete!")
}
```

## Advanced Usage

### HTML Emails with Attachments

```go
msg := &email.Message{
    From:    "sender@gmail.com",
    To:      []string{"recipient@example.com"},
    Subject: "Invoice #12345",
    Body: `
        <h2>Invoice</h2>
        <p>Please find your invoice attached.</p>
        <p>Thank you for your business!</p>
    `,
    HTML: true,
    Attachments: []email.Attachment{
        {
            Filename: "invoice.pdf",
            Content:  invoiceBytes,
            MimeType: "application/pdf",
        },
    },
}
```

### Using Gmail Aliases

If you have aliases configured in Gmail:

```go
msg := &email.Message{
    From: "alias@yourdomain.com", // Gmail alias
    // ... rest of message
}
```

### Service Account Authentication

For Google Workspace users, you can use service accounts:

1. Create a service account in Google Cloud Console
2. Enable domain-wide delegation
3. Configure in Admin Console
4. Use service account key file

See [SERVICE-ACCOUNT-GUIDE.md](SERVICE-ACCOUNT-GUIDE.md) for details.

## Production Best Practices

1. **Token Management**
   - Store tokens securely (encrypted database, secret manager)
   - Implement token refresh logic
   - Monitor token expiration

2. **Error Handling**
   - Implement retry logic with exponential backoff
   - Log failures for monitoring
   - Have fallback mechanisms

3. **Rate Limiting**
   - Implement client-side rate limiting
   - Track daily quotas
   - Queue emails during high volume

4. **Monitoring**
   - Track send success/failure rates
   - Monitor API quotas in Google Cloud Console
   - Set up alerts for failures

## Next Steps

- Read the [Integration Guide](../INTEGRATION.md) for production patterns
- See [API Documentation](../API.md) for all features
- Check [examples](../examples/) for more use cases
