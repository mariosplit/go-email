// gmail.go - Gmail provider implementation using Gmail API
package email

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// gmailProvider implements the Provider interface for Gmail.
// It uses the Gmail API to send emails via OAuth2 authentication.
type gmailProvider struct {
	service *gmail.Service
	config  *GmailConfig
}

// newGmailProvider creates a new Gmail email provider.
// It requires OAuth2 credentials and a token for authentication.
//
// Required Google OAuth2 scopes:
//   - https://www.googleapis.com/auth/gmail.send
//
// The credentials should be OAuth2 credentials for a desktop application
// created in Google Cloud Console. The token can be obtained using the
// authentication helper functions provided in this package.
func newGmailProvider(config *GmailConfig) (Provider, error) {
	ctx := context.Background()

	// Parse OAuth2 config from credentials
	oauthConfig, err := google.ConfigFromJSON(config.CredentialsJSON, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	// Parse the OAuth2 token
	var token *oauth2.Token
	if len(config.TokenJSON) > 0 {
		token = &oauth2.Token{}
		if err := json.Unmarshal(config.TokenJSON, token); err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}
	} else {
		// If no token provided, guide user to authenticate
		return nil, fmt.Errorf("gmail requires initial OAuth authentication - please use the authentication helper")
	}

	// Create Gmail service with OAuth2 authentication
	service, err := gmail.NewService(ctx, option.WithTokenSource(oauthConfig.TokenSource(ctx, token)))
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %w", err)
	}

	return &gmailProvider{
		service: service,
		config:  config,
	}, nil
}

// Send sends an email message using the Gmail API.
// It constructs a properly formatted RFC 2822 message and sends it
// through the authenticated user's Gmail account.
func (g *gmailProvider) Send(ctx context.Context, msg *Message) error {
	// Create Gmail message
	gmailMsg, err := g.createMessage(msg)
	if err != nil {
		return fmt.Errorf("unable to create message: %w", err)
	}

	// Send the message
	_, err = g.service.Users.Messages.Send("me", gmailMsg).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to send message: %w", err)
	}

	return nil
}

// createMessage constructs a Gmail API message from our Message struct.
// It creates a properly formatted RFC 2822 email with headers, body,
// and attachments encoded in base64.
func (g *gmailProvider) createMessage(msg *Message) (*gmail.Message, error) {
	var message strings.Builder

	// Create email headers
	headers := make(map[string]string)
	headers["From"] = msg.From
	headers["To"] = strings.Join(msg.To, ", ")

	if len(msg.Cc) > 0 {
		headers["Cc"] = strings.Join(msg.Cc, ", ")
	}

	if len(msg.Bcc) > 0 {
		headers["Bcc"] = strings.Join(msg.Bcc, ", ")
	}

	headers["Subject"] = msg.Subject
	headers["MIME-Version"] = "1.0"

	// Handle attachments or simple message
	if len(msg.Attachments) > 0 {
		// Multipart message with attachments
		boundary := fmt.Sprintf("boundary-%d", time.Now().UnixNano())
		headers["Content-Type"] = "multipart/mixed; boundary=" + boundary

		// Write headers
		for k, v := range headers {
			message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
		message.WriteString("\r\n")

		// Write body part
		message.WriteString("--" + boundary + "\r\n")
		if msg.HTML {
			message.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		} else {
			message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		}
		message.WriteString("\r\n")
		message.WriteString(msg.Body)
		message.WriteString("\r\n\r\n")

		// Write attachments
		for _, att := range msg.Attachments {
			g.addAttachment(&message, att, boundary)
		}

		// End boundary
		message.WriteString("--" + boundary + "--\r\n")
	} else {
		// Simple message without attachments
		if msg.HTML {
			headers["Content-Type"] = "text/html; charset=utf-8"
		} else {
			headers["Content-Type"] = "text/plain; charset=utf-8"
		}

		// Write headers
		for k, v := range headers {
			message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
		message.WriteString("\r\n")
		message.WriteString(msg.Body)
	}

	// Encode the entire message in base64 for Gmail API
	raw := base64.URLEncoding.EncodeToString([]byte(message.String()))

	return &gmail.Message{
		Raw: raw,
	}, nil
}

// addAttachment adds a single attachment to the email message.
// It encodes the attachment content in base64 and formats it according
// to RFC 2822 standards with proper MIME headers.
func (g *gmailProvider) addAttachment(message *strings.Builder, att Attachment, boundary string) {
	// Determine MIME type
	mimeType := att.MimeType
	if mimeType == "" {
		mimeType = getContentType(att.Filename)
	}

	// Write attachment headers
	message.WriteString("--" + boundary + "\r\n")
	message.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", mimeType, att.Filename))
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename))
	message.WriteString("\r\n")

	// Encode content in base64
	encoded := base64.StdEncoding.EncodeToString(att.Content)

	// Write encoded content in 76-character lines (RFC 2045 standard)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		message.WriteString(encoded[i:end])
		message.WriteString("\r\n")
	}

	message.WriteString("\r\n")
}
