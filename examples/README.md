# Go-Email Examples

This directory contains example code demonstrating how to use the go-email package with different providers.

## Structure

- `basic-usage.go` - Simple examples showing basic package usage
- `outlook/` - Outlook 365 specific examples
- `gmail/` - Gmail specific examples

## Running the Examples

### Prerequisites

1. Install the go-email package:
   ```bash
   go get github.com/go-email/go-email
   ```

2. Set up your provider credentials (see provider-specific guides)

### Outlook 365 Example

1. Set up environment variables in a `.env` file:
   ```env
   OUTLOOK_TENANT_ID=your-tenant-id
   OUTLOOK_CLIENT_ID=your-client-id
   OUTLOOK_CLIENT_SECRET=your-client-secret
   ```

2. Run the example:
   ```bash
   cd outlook
   go run main.go
   ```

### Gmail Example

1. Download your `credentials.json` from Google Cloud Console

2. Place it in the `gmail` directory

3. Run the example:
   ```bash
   cd gmail
   go run main.go
   ```

   On first run, it will guide you through authentication to create `token.json`.

## Basic Usage Example

The `basic-usage.go` file demonstrates:
- Creating clients for different providers
- Sending simple text emails
- Sending HTML emails
- Adding attachments
- Using environment variables

Run it with:
```bash
go run basic-usage.go
```

## Important Notes

- **Never commit credentials** to version control
- Replace example email addresses with real ones
- Check provider-specific sending limits
- Handle errors appropriately in production code

## Common Patterns

### Error Handling
```go
if err := client.Send(msg); err != nil {
    switch {
    case strings.Contains(err.Error(), "authentication"):
        // Handle auth errors
    case strings.Contains(err.Error(), "rate limit"):
        // Handle rate limiting
    default:
        // Handle other errors
    }
}
```

### Retry Logic
```go
for i := 0; i < 3; i++ {
    err := client.Send(msg)
    if err == nil {
        break
    }
    if i < 2 {
        time.Sleep(time.Second * time.Duration(i+1))
    }
}
```

## More Examples

For more advanced usage patterns, see the [Integration Guide](../INTEGRATION.md).
