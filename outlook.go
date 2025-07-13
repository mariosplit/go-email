// outlook.go - Outlook 365 provider implementation using Microsoft Graph API
package email

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

// outlookProvider implements the Provider interface for Outlook 365.
// It uses the Microsoft Graph API to send emails.
type outlookProvider struct {
	client *msgraphsdk.GraphServiceClient
	config *OutlookConfig
}

// newOutlookProvider creates a new Outlook 365 email provider.
// It authenticates using Azure AD client credentials and initializes
// the Microsoft Graph SDK client.
//
// Required Azure AD permissions:
//   - Mail.Send
//
// The from address in messages must be either:
//   - The authenticated user's primary email
//   - An alias of the authenticated user
//   - A shared mailbox the user has "Send As" permissions for
func newOutlookProvider(config *OutlookConfig) (Provider, error) {
	// Create Azure AD credential using client secret
	cred, err := azidentity.NewClientSecretCredential(
		config.TenantID,
		config.ClientID,
		config.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("authentication error: %w", err)
	}

	// Initialize Microsoft Graph client
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("error creating Graph client: %w", err)
	}

	return &outlookProvider{
		client: client,
		config: config,
	}, nil
}

// Send sends an email message using the Microsoft Graph API.
// It constructs a Graph API message from the provided Message struct,
// handles attachments, and sends the email through the sender's mailbox.
func (o *outlookProvider) Send(ctx context.Context, msg *Message) error {
	// Construct the Microsoft Graph message object
	message := o.constructMessage(msg)

	// Add attachments if any
	if err := o.attachFiles(message, msg.Attachments); err != nil {
		return fmt.Errorf("failed to attach files: %w", err)
	}

	// Create send mail request
	requestBody := users.NewItemSendMailPostRequestBody()
	requestBody.SetMessage(message)
	saveToSentItems := true
	requestBody.SetSaveToSentItems(&saveToSentItems)

	// Send the email
	err := o.client.Users().ByUserId(msg.From).SendMail().Post(ctx, requestBody, nil)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// constructMessage builds a Microsoft Graph Message object from our Message struct.
// It sets the subject, body (with appropriate content type), and all recipients.
func (o *outlookProvider) constructMessage(msg *Message) models.Messageable {
	message := models.NewMessage()
	message.SetSubject(&msg.Subject)

	// Set body content and type
	body := models.NewItemBody()
	if msg.HTML {
		contentType := models.HTML_BODYTYPE
		body.SetContentType(&contentType)
	} else {
		contentType := models.TEXT_BODYTYPE
		body.SetContentType(&contentType)
	}
	body.SetContent(&msg.Body)
	message.SetBody(body)

	// Set recipients
	message.SetToRecipients(o.createRecipients(msg.To))

	if len(msg.Cc) > 0 {
		message.SetCcRecipients(o.createRecipients(msg.Cc))
	}

	if len(msg.Bcc) > 0 {
		message.SetBccRecipients(o.createRecipients(msg.Bcc))
	}

	return message
}

// createRecipients converts email addresses to Microsoft Graph Recipient objects.
func (o *outlookProvider) createRecipients(addresses []string) []models.Recipientable {
	recipients := make([]models.Recipientable, len(addresses))
	for i, addr := range addresses {
		recipient := models.NewRecipient()
		emailAddress := models.NewEmailAddress()
		emailAddress.SetAddress(&addr)
		recipient.SetEmailAddress(emailAddress)
		recipients[i] = recipient
	}
	return recipients
}

// attachFiles adds attachments to the message.
// It handles MIME type detection if not specified.
func (o *outlookProvider) attachFiles(message models.Messageable, attachments []Attachment) error {
	if len(attachments) == 0 {
		return nil
	}

	msgAttachments := make([]models.Attachmentable, 0, len(attachments))
	for _, att := range attachments {
		attachment := models.NewFileAttachment()
		attachment.SetName(&att.Filename)
		attachment.SetContentBytes(att.Content)

		// Determine content type
		contentType := att.MimeType
		if contentType == "" {
			contentType = getContentType(att.Filename)
		}
		attachment.SetContentType(&contentType)

		msgAttachments = append(msgAttachments, attachment)
	}

	message.SetAttachments(msgAttachments)
	return nil
}

// getContentType returns the MIME type based on file extension.
// It supports common file types and defaults to application/octet-stream
// for unknown extensions.
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".zip":
		return "application/zip"
	case ".csv":
		return "text/csv"
	case ".xml":
		return "application/xml"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
