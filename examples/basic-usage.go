package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-email/go-email"
)

func main() {
	// Example 1: Using Outlook 365
	outlookExample()

	// Example 2: Using Gmail
	gmailExample()

	// Example 3: Using environment variables
	envExample()

	// Example 4: Sending HTML email with attachments
	htmlExample()
}

func outlookExample() {
	fmt.Println("=== Outlook 365 Example ===")

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
		log.Printf("Failed to create Outlook client: %v\n", err)
		return
	}

	msg := &email.Message{
		From:    "sender@company.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Email from go-email",
		Body:    "This is a test email sent using the go-email package with Outlook 365.",
	}

	// Note: This will fail without valid credentials
	if err := client.Send(msg); err != nil {
		log.Printf("Expected error (no valid credentials): %v\n", err)
	} else {
		fmt.Println("Email sent successfully!")
	}
}

func gmailExample() {
	fmt.Println("\n=== Gmail Example ===")

	// In a real application, you would read these from files
	mockCredentials := []byte(`{"installed":{"client_id":"example","client_secret":"example"}}`)
	mockToken := []byte(`{"access_token":"example","token_type":"Bearer"}`)

	config := &email.Config{
		Provider: "gmail",
		Gmail: &email.GmailConfig{
			CredentialsJSON: mockCredentials,
			TokenJSON:       mockToken,
		},
	}

	client, err := email.NewClient(config)
	if err != nil {
		log.Printf("Failed to create Gmail client: %v\n", err)
		return
	}

	msg := &email.Message{
		From:    "sender@gmail.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Email from go-email",
		Body:    "This is a test email sent using the go-email package with Gmail.",
	}

	// Note: This will fail without valid credentials
	if err := client.Send(msg); err != nil {
		log.Printf("Expected error (no valid credentials): %v\n", err)
	} else {
		fmt.Println("Email sent successfully!")
	}
}

func envExample() {
	fmt.Println("\n=== Environment Variables Example ===")

	// Set example environment variables (in production, these would be set externally)
	os.Setenv("EMAIL_PROVIDER", "outlook365")
	os.Setenv("OUTLOOK_TENANT_ID", "example-tenant")
	os.Setenv("OUTLOOK_CLIENT_ID", "example-client")
	os.Setenv("OUTLOOK_CLIENT_SECRET", "example-secret")

	client, err := email.QuickClientFromEnv()
	if err != nil {
		log.Printf("Failed to create client from env: %v\n", err)
		return
	}

	// QuickSend example
	err = email.QuickSend("outlook365",
		&email.OutlookConfig{
			TenantID:     "example",
			ClientID:     "example",
			ClientSecret: "example",
		},
		"from@example.com",
		"to@example.com",
		"Quick Send Test",
		"This email was sent using QuickSend!")

	if err != nil {
		log.Printf("Expected error (no valid credentials): %v\n", err)
	}

	// Print version information
	fmt.Printf("\nUsing go-email version: %s\n", email.GetVersion())
	versionInfo := email.GetVersionInfo()
	fmt.Printf("Version details: v%d.%d.%d\n", versionInfo.Major, versionInfo.Minor, versionInfo.Patch)
}

func htmlExample() {
	fmt.Println("\n=== HTML Email with Attachments Example ===")

	// Create a mock attachment
	attachmentContent := []byte("This is a test attachment content.")

	msg := &email.Message{
		From: "newsletter@company.com",
		To:   []string{"subscriber@example.com"},
		Cc:   []string{"manager@example.com"},
		Bcc:  []string{"archive@example.com"},
		Subject: "Monthly Newsletter",
		Body: `
			<html>
			<body style="font-family: Arial, sans-serif;">
				<h1 style="color: #333;">Monthly Newsletter</h1>
				<p>Dear Subscriber,</p>
				<p>Welcome to our monthly newsletter! Here are the highlights:</p>
				<ul>
					<li>New feature releases</li>
					<li>Upcoming events</li>
					<li>Customer success stories</li>
				</ul>
				<p>Best regards,<br>The Team</p>
			</body>
			</html>
		`,
		HTML: true,
		Attachments: []email.Attachment{
			{
				Filename: "newsletter.txt",
				Content:  attachmentContent,
				MimeType: "text/plain",
			},
		},
	}

	// Validate the message
	if err := msg.Validate(); err != nil {
		log.Printf("Message validation failed: %v\n", err)
	} else {
		fmt.Println("Message validated successfully!")
		fmt.Printf("- From: %s\n", msg.From)
		fmt.Printf("- To: %v\n", msg.To)
		fmt.Printf("- Subject: %s\n", msg.Subject)
		fmt.Printf("- HTML: %v\n", msg.HTML)
		fmt.Printf("- Attachments: %d\n", len(msg.Attachments))
	}
}
