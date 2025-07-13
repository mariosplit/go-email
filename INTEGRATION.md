# 📧 Go-Email Integration Guide

This guide covers how to integrate `go-email` into your Go applications for sending emails through multiple providers.

## Table of Contents
- [Installation](#installation)
- [Basic Integration](#basic-integration)
- [Advanced Integration Patterns](#advanced-integration-patterns)
- [Production Best Practices](#production-best-practices)
- [Common Use Cases](#common-use-cases)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Performance Optimization](#performance-optimization)

## Installation

```bash
go get github.com/go-email/go-email@v1.0.0
```

## Basic Integration

### 1. Simple Email Service

Create an email service wrapper for your application:

```go
package services

import (
    "fmt"
    "github.com/go-email/go-email"
)

type EmailService struct {
    client *email.Client
}

// NewEmailService creates a new email service instance
func NewEmailService(provider string) (*EmailService, error) {
    config := &email.Config{
        Provider: provider,
    }
    
    // Configure based on provider
    switch provider {
    case "outlook365":
        config.Outlook = &email.OutlookConfig{
            TenantID:     os.Getenv("AZURE_TENANT_ID"),
            ClientID:     os.Getenv("AZURE_CLIENT_ID"),
            ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
        }
    case "gmail":
        // Load credentials from files
        creds, _ := os.ReadFile("credentials.json")
        token, _ := os.ReadFile("token.json")
        config.Gmail = &email.GmailConfig{
            CredentialsJSON: creds,
            TokenJSON:       token,
        }
    }
    
    client, err := email.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create email client: %w", err)
    }
    
    return &EmailService{client: client}, nil
}

// SendWelcomeEmail sends a welcome email to new users
func (s *EmailService) SendWelcomeEmail(userEmail, userName string) error {
    msg := &email.Message{
        From:    "noreply@yourcompany.com",
        To:      []string{userEmail},
        Subject: fmt.Sprintf("Welcome to Our Service, %s!", userName),
        Body:    fmt.Sprintf("Hello %s,\n\nWelcome to our service!", userName),
    }
    
    return s.client.Send(msg)
}
```

### 2. Dependency Injection

Integrate with your application's dependency injection:

```go
package main

import (
    "log"
    "github.com/go-email/go-email"
)

type App struct {
    EmailClient *email.Client
    // Other dependencies
}

func NewApp() (*App, error) {
    // Initialize email client from environment
    emailClient, err := email.QuickClientFromEnv()
    if err != nil {
        return nil, fmt.Errorf("failed to init email: %w", err)
    }
    
    return &App{
        EmailClient: emailClient,
    }, nil
}

func (a *App) NotifyUser(userEmail string, notification string) error {
    return a.EmailClient.Send(&email.Message{
        From:    "notifications@yourapp.com",
        To:      []string{userEmail},
        Subject: "Important Notification",
        Body:    notification,
    })
}
```

## Advanced Integration Patterns

### 1. Email Queue System

Implement an email queue for better reliability:

```go
package queue

import (
    "context"
    "encoding/json"
    "github.com/go-email/go-email"
    "time"
)

type EmailQueue struct {
    client   *email.Client
    queue    chan *email.Message
    workers  int
}

func NewEmailQueue(client *email.Client, workers int) *EmailQueue {
    return &EmailQueue{
        client:  client,
        queue:   make(chan *email.Message, 1000),
        workers: workers,
    }
}

func (eq *EmailQueue) Start(ctx context.Context) {
    for i := 0; i < eq.workers; i++ {
        go eq.worker(ctx)
    }
}

func (eq *EmailQueue) worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case msg := <-eq.queue:
            if err := eq.client.Send(msg); err != nil {
                // Log error and potentially retry
                log.Printf("Failed to send email: %v", err)
                // Implement retry logic here
            }
        }
    }
}

func (eq *EmailQueue) QueueEmail(msg *email.Message) error {
    select {
    case eq.queue <- msg:
        return nil
    default:
        return fmt.Errorf("email queue is full")
    }
}
```

### 2. Template System

Create a template system for consistent emails:

```go
package templates

import (
    "bytes"
    "html/template"
    "github.com/go-email/go-email"
)

type EmailTemplates struct {
    templates map[string]*template.Template
}

func NewEmailTemplates() *EmailTemplates {
    et := &EmailTemplates{
        templates: make(map[string]*template.Template),
    }
    
    // Load templates
    et.LoadTemplate("welcome", `
        <h1>Welcome {{.Name}}!</h1>
        <p>Thank you for joining {{.Company}}.</p>
        <p>Your account has been created with email: {{.Email}}</p>
    `)
    
    et.LoadTemplate("password-reset", `
        <h1>Password Reset Request</h1>
        <p>Click the link below to reset your password:</p>
        <a href="{{.ResetLink}}">Reset Password</a>
        <p>This link expires in {{.ExpiryHours}} hours.</p>
    `)
    
    return et
}

func (et *EmailTemplates) LoadTemplate(name, content string) error {
    tmpl, err := template.New(name).Parse(content)
    if err != nil {
        return err
    }
    et.templates[name] = tmpl
    return nil
}

func (et *EmailTemplates) RenderTemplate(name string, data interface{}) (string, error) {
    tmpl, ok := et.templates[name]
    if !ok {
        return "", fmt.Errorf("template %s not found", name)
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", err
    }
    
    return buf.String(), nil
}

// Usage example
func SendTemplatedEmail(client *email.Client, templates *EmailTemplates) error {
    body, err := templates.RenderTemplate("welcome", map[string]interface{}{
        "Name":    "John Doe",
        "Company": "ACME Corp",
        "Email":   "john@example.com",
    })
    if err != nil {
        return err
    }
    
    return client.Send(&email.Message{
        From:    "welcome@acme.com",
        To:      []string{"john@example.com"},
        Subject: "Welcome to ACME Corp!",
        Body:    body,
        HTML:    true,
    })
}
```

### 3. Multi-Provider Failover

Implement failover between providers:

```go
package failover

import (
    "github.com/go-email/go-email"
    "log"
)

type FailoverEmailClient struct {
    primary   *email.Client
    secondary *email.Client
}

func NewFailoverClient(primaryConfig, secondaryConfig *email.Config) (*FailoverEmailClient, error) {
    primary, err := email.NewClient(primaryConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create primary client: %w", err)
    }
    
    secondary, err := email.NewClient(secondaryConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create secondary client: %w", err)
    }
    
    return &FailoverEmailClient{
        primary:   primary,
        secondary: secondary,
    }, nil
}

func (fc *FailoverEmailClient) Send(msg *email.Message) error {
    // Try primary provider first
    err := fc.primary.Send(msg)
    if err == nil {
        return nil
    }
    
    log.Printf("Primary provider failed: %v, trying secondary", err)
    
    // Fallback to secondary provider
    if err := fc.secondary.Send(msg); err != nil {
        return fmt.Errorf("both providers failed: primary=%v, secondary=%v", err, err)
    }
    
    return nil
}
```

## Production Best Practices

### 1. Configuration Management

```go
package config

import (
    "github.com/go-email/go-email"
    "github.com/kelseyhightower/envconfig"
)

type EmailConfig struct {
    Provider           string `envconfig:"EMAIL_PROVIDER" default:"outlook365"`
    FromAddress        string `envconfig:"EMAIL_FROM" required:"true"`
    MaxRetries         int    `envconfig:"EMAIL_MAX_RETRIES" default:"3"`
    RetryDelaySeconds  int    `envconfig:"EMAIL_RETRY_DELAY" default:"5"`
    
    // Outlook specific
    OutlookTenantID     string `envconfig:"OUTLOOK_TENANT_ID"`
    OutlookClientID     string `envconfig:"OUTLOOK_CLIENT_ID"`
    OutlookClientSecret string `envconfig:"OUTLOOK_CLIENT_SECRET"`
    
    // Gmail specific
    GmailCredsPath string `envconfig:"GMAIL_CREDS_PATH"`
    GmailTokenPath string `envconfig:"GMAIL_TOKEN_PATH"`
}

func LoadEmailConfig() (*EmailConfig, error) {
    var cfg EmailConfig
    if err := envconfig.Process("", &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

func (c *EmailConfig) ToEmailConfig() (*email.Config, error) {
    config := &email.Config{
        Provider: c.Provider,
    }
    
    switch c.Provider {
    case "outlook365":
        config.Outlook = &email.OutlookConfig{
            TenantID:     c.OutlookTenantID,
            ClientID:     c.OutlookClientID,
            ClientSecret: c.OutlookClientSecret,
        }
    case "gmail":
        creds, err := os.ReadFile(c.GmailCredsPath)
        if err != nil {
            return nil, err
        }
        token, err := os.ReadFile(c.GmailTokenPath)
        if err != nil {
            return nil, err
        }
        config.Gmail = &email.GmailConfig{
            CredentialsJSON: creds,
            TokenJSON:       token,
        }
    }
    
    return config, nil
}
```

### 2. Monitoring and Metrics

```go
package monitoring

import (
    "github.com/go-email/go-email"
    "github.com/prometheus/client_golang/prometheus"
    "time"
)

type MetricsEmailClient struct {
    client        *email.Client
    sentCounter   prometheus.Counter
    errorCounter  prometheus.Counter
    durationHist  prometheus.Histogram
}

func NewMetricsClient(client *email.Client) *MetricsEmailClient {
    return &MetricsEmailClient{
        client: client,
        sentCounter: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "emails_sent_total",
            Help: "Total number of emails sent",
        }),
        errorCounter: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "emails_errors_total",
            Help: "Total number of email sending errors",
        }),
        durationHist: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name: "email_send_duration_seconds",
            Help: "Email sending duration in seconds",
        }),
    }
}

func (mc *MetricsEmailClient) Send(msg *email.Message) error {
    start := time.Now()
    err := mc.client.Send(msg)
    duration := time.Since(start).Seconds()
    
    mc.durationHist.Observe(duration)
    
    if err != nil {
        mc.errorCounter.Inc()
        return err
    }
    
    mc.sentCounter.Inc()
    return nil
}
```

### 3. Rate Limiting

```go
package ratelimit

import (
    "context"
    "github.com/go-email/go-email"
    "golang.org/x/time/rate"
)

type RateLimitedClient struct {
    client  *email.Client
    limiter *rate.Limiter
}

func NewRateLimitedClient(client *email.Client, rps int) *RateLimitedClient {
    return &RateLimitedClient{
        client:  client,
        limiter: rate.NewLimiter(rate.Limit(rps), rps),
    }
}

func (rl *RateLimitedClient) Send(ctx context.Context, msg *email.Message) error {
    if err := rl.limiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limit wait failed: %w", err)
    }
    
    return rl.client.SendWithContext(ctx, msg)
}
```

## Common Use Cases

### 1. User Notifications

```go
type NotificationService struct {
    emailClient *email.Client
    fromAddress string
}

func (ns *NotificationService) SendAccountActivation(user User, activationLink string) error {
    return ns.emailClient.Send(&email.Message{
        From:    ns.fromAddress,
        To:      []string{user.Email},
        Subject: "Activate Your Account",
        Body: fmt.Sprintf(`
            <h2>Welcome %s!</h2>
            <p>Please click the link below to activate your account:</p>
            <a href="%s" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px;">Activate Account</a>
            <p>This link will expire in 24 hours.</p>
        `, user.Name, activationLink),
        HTML: true,
    })
}

func (ns *NotificationService) SendPasswordReset(user User, resetToken string) error {
    return ns.emailClient.Send(&email.Message{
        From:    ns.fromAddress,
        To:      []string{user.Email},
        Subject: "Password Reset Request",
        Body: fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>We received a request to reset your password. Use the code below:</p>
            <h2 style="background-color: #f4f4f4; padding: 10px; text-align: center;">%s</h2>
            <p>This code expires in 15 minutes.</p>
            <p>If you didn't request this, please ignore this email.</p>
        `, user.Name, resetToken),
        HTML: true,
    })
}
```

### 2. Transaction Emails

```go
type OrderService struct {
    emailClient *email.Client
}

func (os *OrderService) SendOrderConfirmation(order Order, pdfInvoice []byte) error {
    body := fmt.Sprintf(`
        <h2>Order Confirmation #%s</h2>
        <p>Thank you for your order!</p>
        <h3>Order Details:</h3>
        <table style="border-collapse: collapse; width: 100%%;">
            <tr>
                <th style="border: 1px solid #ddd; padding: 8px;">Item</th>
                <th style="border: 1px solid #ddd; padding: 8px;">Quantity</th>
                <th style="border: 1px solid #ddd; padding: 8px;">Price</th>
            </tr>
    `, order.ID)
    
    for _, item := range order.Items {
        body += fmt.Sprintf(`
            <tr>
                <td style="border: 1px solid #ddd; padding: 8px;">%s</td>
                <td style="border: 1px solid #ddd; padding: 8px;">%d</td>
                <td style="border: 1px solid #ddd; padding: 8px;">$%.2f</td>
            </tr>
        `, item.Name, item.Quantity, item.Price)
    }
    
    body += fmt.Sprintf(`
        </table>
        <h3>Total: $%.2f</h3>
        <p>Your invoice is attached to this email.</p>
    `, order.Total)
    
    return os.emailClient.Send(&email.Message{
        From:    "orders@example.com",
        To:      []string{order.CustomerEmail},
        Subject: fmt.Sprintf("Order Confirmation #%s", order.ID),
        Body:    body,
        HTML:    true,
        Attachments: []email.Attachment{
            {
                Filename: fmt.Sprintf("invoice_%s.pdf", order.ID),
                Content:  pdfInvoice,
                MimeType: "application/pdf",
            },
        },
    })
}
```

### 3. Bulk Marketing Emails

```go
type MarketingService struct {
    emailClient *email.Client
    batchSize   int
}

func (ms *MarketingService) SendCampaign(campaign Campaign, recipients []string) error {
    // Process in batches
    for i := 0; i < len(recipients); i += ms.batchSize {
        end := i + ms.batchSize
        if end > len(recipients) {
            end = len(recipients)
        }
        
        batch := recipients[i:end]
        
        // Send to batch with BCC to hide recipients
        msg := &email.Message{
            From:    campaign.FromAddress,
            To:      []string{campaign.FromAddress}, // Send to self
            Bcc:     batch,                           // BCC actual recipients
            Subject: campaign.Subject,
            Body:    campaign.Body,
            HTML:    campaign.IsHTML,
        }
        
        if err := ms.emailClient.Send(msg); err != nil {
            // Log error but continue with other batches
            log.Printf("Failed to send batch %d: %v", i/ms.batchSize, err)
        }
        
        // Rate limit between batches
        time.Sleep(time.Second)
    }
    
    return nil
}
```

## Error Handling

### Comprehensive Error Handling

```go
func SendEmailWithRetry(client *email.Client, msg *email.Message, maxRetries int) error {
    var lastErr error
    
    for i := 0; i <= maxRetries; i++ {
        err := client.Send(msg)
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        // Check if error is retryable
        if !isRetryableError(err) {
            return fmt.Errorf("non-retryable error: %w", err)
        }
        
        if i < maxRetries {
            // Exponential backoff
            delay := time.Duration(math.Pow(2, float64(i))) * time.Second
            log.Printf("Email send failed (attempt %d/%d), retrying in %v: %v", 
                i+1, maxRetries+1, delay, err)
            time.Sleep(delay)
        }
    }
    
    return fmt.Errorf("failed after %d retries: %w", maxRetries+1, lastErr)
}

func isRetryableError(err error) bool {
    errStr := err.Error()
    
    // Network errors
    if strings.Contains(errStr, "connection refused") ||
       strings.Contains(errStr, "timeout") ||
       strings.Contains(errStr, "temporary failure") {
        return true
    }
    
    // Rate limiting
    if strings.Contains(errStr, "rate limit") ||
       strings.Contains(errStr, "too many requests") {
        return true
    }
    
    // Authentication errors are not retryable
    if strings.Contains(errStr, "authentication") ||
       strings.Contains(errStr, "unauthorized") {
        return false
    }
    
    return false
}
```

## Testing

### Mock Email Client

```go
package mocks

import (
    "context"
    "github.com/go-email/go-email"
)

type MockEmailClient struct {
    SentEmails []email.Message
    SendError  error
}

func NewMockEmailClient() *MockEmailClient {
    return &MockEmailClient{
        SentEmails: make([]email.Message, 0),
    }
}

func (m *MockEmailClient) Send(ctx context.Context, msg *email.Message) error {
    if m.SendError != nil {
        return m.SendError
    }
    
    m.SentEmails = append(m.SentEmails, *msg)
    return nil
}

func (m *MockEmailClient) GetLastEmail() *email.Message {
    if len(m.SentEmails) == 0 {
        return nil
    }
    return &m.SentEmails[len(m.SentEmails)-1]
}

func (m *MockEmailClient) Clear() {
    m.SentEmails = make([]email.Message, 0)
    m.SendError = nil
}
```

### Integration Tests

```go
func TestEmailIntegration(t *testing.T) {
    // Skip in CI if no credentials
    if os.Getenv("CI") == "true" {
        t.Skip("Skipping integration test in CI")
    }
    
    client, err := email.QuickClientFromEnv()
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    
    testEmail := &email.Message{
        From:    os.Getenv("TEST_FROM_EMAIL"),
        To:      []string{os.Getenv("TEST_TO_EMAIL")},
        Subject: "Integration Test Email",
        Body:    "This is a test email from the integration test suite.",
    }
    
    if err := client.Send(testEmail); err != nil {
        t.Errorf("Failed to send test email: %v", err)
    }
}
```

## Performance Optimization

### Connection Pooling

For high-volume applications, consider implementing connection pooling:

```go
type PooledEmailClient struct {
    pool     chan *email.Client
    factory  func() (*email.Client, error)
    maxSize  int
}

func NewPooledClient(factory func() (*email.Client, error), size int) (*PooledEmailClient, error) {
    pool := make(chan *email.Client, size)
    
    // Pre-populate pool
    for i := 0; i < size; i++ {
        client, err := factory()
        if err != nil {
            return nil, fmt.Errorf("failed to create client %d: %w", i, err)
        }
        pool <- client
    }
    
    return &PooledEmailClient{
        pool:    pool,
        factory: factory,
        maxSize: size,
    }, nil
}

func (p *PooledEmailClient) Send(msg *email.Message) error {
    client := <-p.pool
    defer func() { p.pool <- client }()
    
    return client.Send(msg)
}
```

### Async Processing

For better performance, process emails asynchronously:

```go
type AsyncEmailService struct {
    client   *email.Client
    workers  int
    queue    chan emailJob
}

type emailJob struct {
    message *email.Message
    result  chan error
}

func NewAsyncEmailService(client *email.Client, workers int) *AsyncEmailService {
    svc := &AsyncEmailService{
        client:  client,
        workers: workers,
        queue:   make(chan emailJob, 1000),
    }
    
    // Start workers
    for i := 0; i < workers; i++ {
        go svc.worker()
    }
    
    return svc
}

func (s *AsyncEmailService) worker() {
    for job := range s.queue {
        job.result <- s.client.Send(job.message)
    }
}

func (s *AsyncEmailService) SendAsync(msg *email.Message) <-chan error {
    result := make(chan error, 1)
    s.queue <- emailJob{message: msg, result: result}
    return result
}

func (s *AsyncEmailService) SendAsyncWithTimeout(msg *email.Message, timeout time.Duration) error {
    select {
    case err := <-s.SendAsync(msg):
        return err
    case <-time.After(timeout):
        return fmt.Errorf("email send timeout after %v", timeout)
    }
}
```

## Version Information

Check the package version in your application:

```go
import "github.com/go-email/go-email"

func main() {
    fmt.Printf("Using go-email version: %s\n", email.GetVersion())
    
    // Get detailed version info
    info := email.GetVersionInfo()
    fmt.Printf("Version: %s (Major: %d, Minor: %d, Patch: %d)\n", 
        info.Version, info.Major, info.Minor, info.Patch)
}
```

## Summary

The `go-email` package provides a flexible foundation for email functionality in your Go applications. Key integration points:

1. **Simple API** - Easy to integrate with just a few lines of code
2. **Provider Flexibility** - Switch between Outlook and Gmail without changing application code
3. **Production Ready** - Built-in support for retries, context, and error handling
4. **Extensible** - Easy to wrap with your own abstractions for queuing, templates, and monitoring

For more examples and detailed API documentation, visit the [GitHub repository](https://github.com/go-email/go-email).
