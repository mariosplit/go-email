package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-email/go-email"
)

func main() {
	// Check if credentials exist
	if _, err := os.Stat("credentials.json"); os.IsNotExist(err) {
		log.Fatal("credentials.json not found. Please download it from Google Cloud Console.")
	}

	// Check if token exists, if not, authenticate first
	if _, err := os.Stat("token.json"); os.IsNotExist(err) {
		fmt.Println("No token found. Starting authentication process...")
		if err := authenticateGmail(); err != nil {
			log.Fatalf("Authentication failed: %v", err)
		}
	}

	// Read credentials and token
	creds, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Failed to read credentials: %v", err)
	}

	token, err := os.ReadFile("token.json")
	if err != nil {
		log.Fatalf("Failed to read token: %v", err)
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

// authenticateGmail performs the OAuth2 authentication flow
func authenticateGmail() error {
	token, err := email.AuthenticateGmailFromFile("credentials.json")
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save token for future use
	err = os.WriteFile("token.json", token, 0600)
	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("✓ Authentication successful! Token saved to token.json")
	return nil
}

func sendSimpleEmail(client *email.Client) error {
	msg := &email.Message{
		From:    "your-email@gmail.com", // Replace with your Gmail address
		To:      []string{"recipient@example.com"},
		Subject: "Test Email from go-email (Gmail)",
		Body:    "This is a test email sent using the go-email package with Gmail API.",
	}

	return client.Send(msg)
}

func sendHTMLEmail(client *email.Client) error {
	msg := &email.Message{
		From:    "your-email@gmail.com", // Replace with your Gmail address
		To:      []string{"recipient@example.com"},
		Cc:      []string{"cc@example.com"},
		Subject: "HTML Email Test",
		Body: `
			<html>
			<body style="font-family: Arial, sans-serif; color: #333;">
				<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
					<h2 style="color: #4285f4;">Hello from go-email!</h2>
					<p>This is an <strong>HTML email</strong> sent using Gmail API.</p>
					<div style="background-color: #f8f9fa; padding: 15px; border-radius: 5px;">
						<h3>Features:</h3>
						<ul>
							<li>Rich HTML formatting</li>
							<li>Inline CSS styling</li>
							<li>Gmail API integration</li>
						</ul>
					</div>
					<p style="margin-top: 20px;">Best regards,<br>
					<em style="color: #4285f4;">The go-email team</em></p>
				</div>
			</body>
			</html>
		`,
		HTML: true,
	}

	return client.Send(msg)
}

func sendEmailWithAttachment(client *email.Client) error {
	// Create a sample CSV file content
	csvContent := []byte(`Name,Email,Role
John Doe,john@example.com,Developer
Jane Smith,jane@example.com,Designer
Bob Johnson,bob@example.com,Manager

This CSV file was attached using the go-email package with Gmail API.
`)

	msg := &email.Message{
		From:    "your-email@gmail.com", // Replace with your Gmail address
		To:      []string{"recipient@example.com"},
		Bcc:     []string{"archive@example.com"},
		Subject: "Email with CSV Attachment",
		Body: `Hi,

Please find the attached CSV file with team information.

This email demonstrates the attachment feature of the go-email package using Gmail API.

Key features:
- Multiple file attachments supported
- Automatic MIME type detection
- Base64 encoding handled automatically

Best regards,
Your Team`,
		Attachments: []email.Attachment{
			{
				Filename: "team-data.csv",
				Content:  csvContent,
				MimeType: "text/csv", // Optional - will be auto-detected
			},
		},
	}

	return client.Send(msg)
}
