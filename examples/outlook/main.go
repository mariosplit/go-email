package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-email/go-email"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Create configuration from environment variables
	config := &email.Config{
		Provider: "outlook365",
		Outlook: &email.OutlookConfig{
			TenantID:     os.Getenv("OUTLOOK_TENANT_ID"),
			ClientID:     os.Getenv("OUTLOOK_CLIENT_ID"),
			ClientSecret: os.Getenv("OUTLOOK_CLIENT_SECRET"),
		},
	}

	// Validate configuration
	if config.Outlook.TenantID == "" || config.Outlook.ClientID == "" || config.Outlook.ClientSecret == "" {
		log.Fatal("Missing required Outlook configuration. Please set OUTLOOK_TENANT_ID, OUTLOOK_CLIENT_ID, and OUTLOOK_CLIENT_SECRET")
	}

	// Create email client
	client, err := email.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create email client: %v", err)
	}

	// Example 1: Send a simple text email
	fmt.Println("Sending simple text email...")
	err = sendSimpleEmail(client)
	if err != nil {
		log.Printf("Failed to send simple email: %v", err)
	} else {
		fmt.Println("✓ Simple email sent successfully!")
	}

	// Example 2: Send HTML email
	fmt.Println("\nSending HTML email...")
	err = sendHTMLEmail(client)
	if err != nil {
		log.Printf("Failed to send HTML email: %v", err)
	} else {
		fmt.Println("✓ HTML email sent successfully!")
	}

	// Example 3: Send email with attachment
	fmt.Println("\nSending email with attachment...")
	err = sendEmailWithAttachment(client)
	if err != nil {
		log.Printf("Failed to send email with attachment: %v", err)
	} else {
		fmt.Println("✓ Email with attachment sent successfully!")
	}
}

func sendSimpleEmail(client *email.Client) error {
	msg := &email.Message{
		From:    "sender@yourdomain.com", // Replace with your email
		To:      []string{"recipient@example.com"},
		Subject: "Test Email from go-email (Outlook)",
		Body:    "This is a test email sent using the go-email package with Outlook 365.",
	}

	return client.Send(msg)
}

func sendHTMLEmail(client *email.Client) error {
	msg := &email.Message{
		From:    "sender@yourdomain.com", // Replace with your email
		To:      []string{"recipient@example.com"},
		Cc:      []string{"cc@example.com"},
		Subject: "HTML Email Test",
		Body: `
			<html>
			<body style="font-family: Arial, sans-serif; color: #333;">
				<h2 style="color: #0078d4;">Hello from go-email!</h2>
				<p>This is an <strong>HTML email</strong> sent using Outlook 365.</p>
				<ul>
					<li>Supports rich formatting</li>
					<li>Can include images and links</li>
					<li>Styled with CSS</li>
				</ul>
				<p>Best regards,<br>
				<em>The go-email team</em></p>
			</body>
			</html>
		`,
		HTML: true,
	}

	return client.Send(msg)
}

func sendEmailWithAttachment(client *email.Client) error {
	// Create a sample text file content
	attachmentContent := []byte(`Sample Attachment Content

This is a test file attached to an email sent using go-email with Outlook 365.

Features demonstrated:
- File attachments
- MIME type detection
- Multiple recipients

Thank you for using go-email!
`)

	msg := &email.Message{
		From:    "sender@yourdomain.com", // Replace with your email
		To:      []string{"recipient@example.com"},
		Bcc:     []string{"archive@example.com"},
		Subject: "Email with Attachment",
		Body: `Hi,

Please find the attached document.

This email demonstrates the attachment feature of the go-email package using Outlook 365.

Best regards,
Your Team`,
		Attachments: []email.Attachment{
			{
				Filename: "test-document.txt",
				Content:  attachmentContent,
				// MimeType is optional - will be auto-detected
			},
		},
	}

	return client.Send(msg)
}
