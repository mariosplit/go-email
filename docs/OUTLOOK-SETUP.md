# Outlook 365 Setup Guide

This guide walks you through setting up Outlook 365 (Microsoft 365) email sending using the go-email package.

## Prerequisites

- Microsoft 365 account with admin access
- Azure Active Directory access
- Go 1.20 or later

## Step 1: Register an Application in Azure AD

1. **Sign in to Azure Portal**
   - Go to [https://portal.azure.com](https://portal.azure.com)
   - Sign in with your Microsoft 365 admin account

2. **Navigate to Azure Active Directory**
   - Click on "Azure Active Directory" in the left menu
   - If you don't see it, search for it in the top search bar

3. **Create a New App Registration**
   - Click on "App registrations" in the left menu
   - Click "New registration"
   - Fill in the details:
     - **Name**: `go-email-app` (or your preferred name)
     - **Supported account types**: Choose based on your needs:
       - "Single tenant" for your organization only
       - "Multitenant" for any Azure AD tenant
     - **Redirect URI**: Leave blank (not needed for this flow)
   - Click "Register"

4. **Save Your Application IDs**
   - After registration, you'll see the app overview
   - Copy and save these values:
     - **Application (client) ID**: This is your `ClientID`
     - **Directory (tenant) ID**: This is your `TenantID`

## Step 2: Create Client Secret

1. **Navigate to Certificates & Secrets**
   - In your app registration, click "Certificates & secrets" in the left menu

2. **Create New Client Secret**
   - Click "New client secret"
   - Add a description (e.g., "go-email secret")
   - Choose expiration period (recommended: 24 months)
   - Click "Add"

3. **Save the Secret Value**
   - **IMPORTANT**: Copy the secret value immediately
   - This is your `ClientSecret`
   - You won't be able to see it again!

## Step 3: Grant API Permissions

1. **Navigate to API Permissions**
   - Click "API permissions" in the left menu

2. **Add Microsoft Graph Permission**
   - Click "Add a permission"
   - Choose "Microsoft Graph"
   - Choose "Application permissions" (not delegated)
   - Search for and select: `Mail.Send`
   - Click "Add permissions"

3. **Grant Admin Consent**
   - Click "Grant admin consent for [Your Organization]"
   - Confirm by clicking "Yes"
   - The status should show "Granted" with a green checkmark

## Step 4: Configure go-email

### Option 1: Using Environment Variables

Create a `.env` file in your project:

```env
EMAIL_PROVIDER=outlook365
OUTLOOK_TENANT_ID=your-tenant-id
OUTLOOK_CLIENT_ID=your-client-id
OUTLOOK_CLIENT_SECRET=your-client-secret
```

Use in your code:

```go
package main

import (
    "log"
    "github.com/go-email/go-email"
    "github.com/joho/godotenv"
)

func main() {
    // Load .env file
    godotenv.Load()
    
    // Create client from environment
    client, err := email.QuickClientFromEnv()
    if err != nil {
        log.Fatal(err)
    }
    
    // Send email
    msg := &email.Message{
        From:    "sender@yourcompany.com",
        To:      []string{"recipient@example.com"},
        Subject: "Test Email",
        Body:    "Hello from go-email!",
    }
    
    if err := client.Send(msg); err != nil {
        log.Fatal(err)
    }
}
```

### Option 2: Direct Configuration

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
    
    // Send email...
}
```

## Important Notes

### About the From Address

The `From` address in your emails must be:
- A user in your Azure AD tenant
- A valid mailbox that the application has access to
- For application permissions, this can be any mailbox in your tenant

### Security Best Practices

1. **Never commit credentials to source control**
   - Use environment variables or secure vaults
   - Add `.env` to `.gitignore`

2. **Rotate secrets regularly**
   - Set calendar reminders before secret expiration
   - Azure AD secrets can expire in 3, 6, 12, or 24 months

3. **Use least privilege**
   - Only grant `Mail.Send` permission
   - Don't grant unnecessary permissions

4. **Monitor usage**
   - Check Azure AD sign-in logs regularly
   - Set up alerts for unusual activity

## Troubleshooting

### Common Errors

1. **"authentication error"**
   - Verify your TenantID, ClientID, and ClientSecret
   - Ensure the secret hasn't expired
   - Check if admin consent was granted

2. **"failed to send email: 404"**
   - The From address doesn't exist
   - The mailbox is disabled or unlicensed

3. **"failed to send email: 403"**
   - Permissions not granted correctly
   - Admin consent not provided
   - Application not authorized for the mailbox

### Testing Your Setup

Create a simple test program:

```go
package main

import (
    "fmt"
    "log"
    "github.com/go-email/go-email"
)

func main() {
    // Your configuration here
    client, err := email.QuickClientFromEnv()
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    
    msg := &email.Message{
        From:    "test@yourdomain.com", // Must be valid mailbox
        To:      []string{"your-email@example.com"},
        Subject: "go-email Test",
        Body:    "If you receive this, your setup is working!",
    }
    
    if err := client.Send(msg); err != nil {
        log.Fatalf("Failed to send: %v", err)
    }
    
    fmt.Println("Email sent successfully!")
}
```

## Advanced Configuration

### Using Shared Mailboxes

To send from a shared mailbox:

1. Ensure the shared mailbox exists in your tenant
2. The application permissions allow sending from any mailbox
3. Use the shared mailbox address as the `From` address

### Rate Limiting

Microsoft Graph has rate limits:
- 10,000 requests per 10 minutes per app per tenant
- Plan accordingly for bulk sending

### Monitoring

Use Azure Monitor to track:
- API usage
- Failed requests
- Performance metrics

## Next Steps

- Read the [Integration Guide](../INTEGRATION.md) for production patterns
- Check [API Documentation](../API.md) for all features
- See [examples](../examples/) for more use cases
